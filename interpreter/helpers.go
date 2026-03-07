package interpreter

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/z-sk1/ayla-lang/parser"
	"golang.org/x/term"
)

func SetupAylaDirs() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	libDir := filepath.Join(home, ".ayla", "lib")

	err = os.MkdirAll(libDir, 0755)
	if err != nil {
		return "", err
	}

	return libDir, nil
}

func New(path string) *Interpreter {
	dir := filepath.Dir(path)

	env := &Environment{
		store:    make(map[string]Variable),
		methods:  make(map[string]map[string]*Func),
		builtins: make(map[string]*BuiltinFunc),
		defers:   make([]*parser.FuncCall, 0),
		mu:       sync.RWMutex{},
	}

	typeEnv := make(map[string]TypeValue)

	wd, _ := os.Getwd()

	i := &Interpreter{
		Env:           env,
		modules:       make(map[string]ModuleValue),
		nativeModules: make(map[string]NativeLoader),
		currentDir:    dir,
	}

	libDir, err := SetupAylaDirs()
	if err != nil {
		fmt.Println("Ayla dir error:", err)
	}

	i.modulePaths = []string{
		".",
		"./lib",
		filepath.Join(wd, "lib"),
		filepath.Join(i.currentDir, "lib"),
		libDir,
	}

	if env := os.Getenv("AYLA_PATH"); env != "" {
		i.modulePaths = append(i.modulePaths, filepath.SplitList(env)...)
	}

	i.registerBuiltins()
	i.registerNativeModules()
	initBuiltinTypes(typeEnv)

	i.typeEnv = typeEnv

	return i
}

func NewWithEnv(env *Environment, path string) *Interpreter {
	typeEnv := make(map[string]TypeValue)

	dir := filepath.Dir(path)

	wd, _ := os.Getwd()

	i := &Interpreter{
		Env:           env,
		modules:       make(map[string]ModuleValue),
		nativeModules: make(map[string]NativeLoader),
		currentDir:    dir,
	}

	libDir, err := SetupAylaDirs()
	if err != nil {
		fmt.Println("Ayla dir error:", err)
	}

	i.modulePaths = []string{
		".",
		wd,
		"./lib",
		filepath.Join(wd, "lib"),
		filepath.Join(i.currentDir, "lib"),
		libDir,
	}

	if env := os.Getenv("AYLA_PATH"); env != "" {
		i.modulePaths = append(i.modulePaths, filepath.SplitList(env)...)
	}

	i.registerBuiltins()
	i.registerNativeModules()
	initBuiltinTypes(typeEnv)

	i.typeEnv = typeEnv

	return i
}

func NewEnvironment(parent *Environment) *Environment {
	return &Environment{
		store:    make(map[string]Variable),
		defers:   make([]*parser.FuncCall, 0),
		builtins: parent.builtins,
		parent:   parent,
		mu:       sync.RWMutex{},
	}
}

func NewRuntimeError(node parser.Node, msg string) RuntimeError {
	if node == nil {
		return RuntimeError{Message: msg, Line: -1, Column: -1}
	}

	line, col := node.Pos()
	return RuntimeError{Message: msg, Line: line, Column: col}
}

func (e *Environment) Get(name string) (Value, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if v, ok := e.store[name]; ok {
		return v.Value, true
	}

	if e.parent != nil {
		return e.parent.Get(name)
	}

	return nil, false
}

func (e *Environment) GetLocal(name string) (Value, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if v, ok := e.store[name]; ok {
		return v.Value, true
	}

	return nil, false
}

func (e *Environment) Define(name string, val Value) Value {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.store[name] = Variable{Value: val, Lifetime: -1}
	return val
}

func (e *Environment) DefineWithLifetime(name string, val Value, lifetime int) Value {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.store[name] = Variable{Value: val, Lifetime: lifetime}
	return val
}

func (e *Environment) Set(name string, val Value) Value {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.store[name]; ok {
		e.store[name] = Variable{Value: val, Lifetime: -1}
		return val
	}

	if e.parent != nil {
		return e.parent.Set(name, val)
	}

	return nil
}

func (e *Environment) SetMethod(typ *TypeInfo, name string, fn *Func) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.methods[typ.Name] == nil {
		e.methods[typ.Name] = map[string]*Func{}
	}
	e.methods[typ.Name][name] = fn
}

func (e *Environment) GetMethod(typ *TypeInfo, name string) (*Func, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for env := e; env != nil; env = env.parent {
		if m := env.methods[typ.Name]; m != nil {
			if fn, ok := m[name]; ok {
				return fn, true
			}
		}
	}
	return nil, false
}

func (e *Environment) AddDefer(call *parser.FuncCall) {
	e.defers = append(e.defers, call)
}

func (i *Interpreter) runDefers(env *Environment) error {
	for j := len(env.defers) - 1; j >= 0; j-- {
		_, err := i.evalFuncCall(env.defers[j])
		if err != nil {
			return err
		}
	}
	env.defers = nil
	return nil
}

func toFloat(v Value) (float64, bool) {
	switch x := v.(type) {
	case FloatValue:
		return x.V, true
	case IntValue:
		return float64(x.V), true
	default:
		return 0, false
	}
}

func typesAssignable(from, to *TypeInfo) bool {
	if from == nil || to == nil {
		return false
	}

	if typesIdentical(from, to) {
		return true
	}

	if to.Kind == TypeAny {
		return true
	}

	if from.Alias {
		return typesAssignable(from.Underlying, to)
	}
	if to.Alias {
		return typesAssignable(from, to.Underlying)
	}

	if to.Kind == TypeNamed {
		return sameUnderlying(from, to)
	}

	if from.Kind == TypeNamed {
		return typesIdentical(from, to)
	}

	switch {

	case from.Kind == TypeArray && to.Kind == TypeArray:
		return typesIdentical(from.Elem, to.Elem)

	case from.Kind == TypeFixedArray && to.Kind == TypeFixedArray:
		return typesIdentical(from.Elem, to.Elem) &&
			from.Size == to.Size

	case from.Kind == TypeMap && to.Kind == TypeMap:
		return typesIdentical(from.Key, to.Key) &&
			typesIdentical(from.Value, to.Value)

	case from.Kind == TypeFunc && to.Kind == TypeFunc:
		if len(from.Params) != len(to.Params) ||
			len(from.Returns) != len(to.Returns) {
			return false
		}
		for i := range from.Params {
			if !typesIdentical(from.Params[i], to.Params[i]) {
				return false
			}
		}
		for i := range from.Returns {
			if !typesIdentical(from.Returns[i], to.Returns[i]) {
				return false
			}
		}
		return true
	}

	if from.Kind == TypeInt && to.Kind == TypeFloat {
		return true
	}

	return false
}

func sameUnderlying(a, b *TypeInfo) bool {
	if a == nil || b == nil {
		return false
	}

	ua := a
	ub := b

	if a.Kind == TypeNamed {
		ua = a.Underlying
	}
	if b.Kind == TypeNamed {
		ub = b.Underlying
	}

	return typesIdentical(ua, ub)
}

func typesIdentical(a, b *TypeInfo) bool {
	if a == nil || b == nil {
		return false
	}

	if a == b {
		return true
	}

	if a.Kind != b.Kind {
		return false
	}

	switch a.Kind {

	case TypeInt, TypeFloat, TypeString, TypeBool, TypeAny:
		return true

	case TypeArray:
		return typesIdentical(a.Elem, b.Elem)

	case TypeFixedArray:
		return typesIdentical(a.Elem, b.Elem) &&
			a.Size == b.Size

	case TypeMap:
		return typesIdentical(a.Key, b.Key) &&
			typesIdentical(a.Value, b.Value)

	case TypeStruct:
		if len(a.Fields) != len(b.Fields) {
			return false
		}
		for name, af := range a.Fields {
			bf, ok := b.Fields[name]
			if !ok || !typesIdentical(af, bf) {
				return false
			}
		}
		return true

	case TypeFunc:
		if len(a.Params) != len(b.Params) ||
			len(a.Returns) != len(b.Returns) {
			return false
		}
		for i := range a.Params {
			if !typesIdentical(a.Params[i], b.Params[i]) {
				return false
			}
		}
		for i := range a.Returns {
			if !typesIdentical(a.Returns[i], b.Returns[i]) {
				return false
			}
		}
		return true

	case TypeEnum:
		return a == b // enums are nominal

	case TypeNamed:
		return a == b // named types are nominal

	default:
		return false
	}
}

func (i *Interpreter) promoteValueToType(v Value, ti *TypeInfo) Value {
	ti = unwrapAlias(ti)

	actual := i.typeInfoFromValue(v)

	if actual == ti {
		return v
	}

	if ti.Kind == TypeNamed {
		return NamedValue{
			TypeName: ti,
			Value:    v,
		}
	}

	switch v := v.(type) {
	case ArrayValue:
		if ti.Kind == TypeArray {
			return ArrayValue{
				Elements: v.Elements,
				ElemType: ti.Elem,
				Capacity: v.Capacity,
				Fixed:    v.Fixed,
			}
		}
	}

	return v
}

func unwrapNamed(v Value) Value {
	for {
		if nv, ok := v.(NamedValue); ok {
			v = nv.Value
		} else {
			return v
		}
	}
}

func unwrapAlias(t *TypeInfo) *TypeInfo {
	for t != nil && t.Alias {
		t = t.Underlying
	}
	return t
}

func capacityFromType(ti *TypeInfo, elems []Value) int {
	if ti.Kind == TypeFixedArray {
		return ti.Size
	}
	return len(elems)
}

func readKey() (rune, error) {
	fd := int(os.Stdin.Fd())

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return 0, err
	}
	defer term.Restore(fd, oldState)

	var buf [1]byte
	_, err = os.Stdin.Read(buf[:])
	if err != nil {
		return 0, err
	}

	if buf == [1]byte{'\r'} {
		buf = [1]byte{'\n'}
	}

	return rune(buf[0]), err
}

func (env *Environment) assignInput(node parser.Node, varName string, val Value, input string) error {
	switch val.(type) {

	case IntValue:
		n, err := strconv.Atoi(input)
		if err != nil {
			return NewRuntimeError(node, "invalid int input")
		}
		env.Set(varName, IntValue{V: n})

	case FloatValue:
		f, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return NewRuntimeError(node, "invalid float input")
		}
		env.Set(varName, FloatValue{V: f})

	case BoolValue:
		b, err := strconv.ParseBool(input)
		if err != nil {
			return NewRuntimeError(node, "invalid bool input")
		}
		env.Set(varName, BoolValue{V: b})

	case StringValue:
		env.Set(varName, StringValue{V: input})

	case UninitializedValue, NilValue:
		return NewRuntimeError(node, "variable must have a type before scan")

	default:
		return NewRuntimeError(node, "unsupported type for scan")
	}

	return nil
}

func (i *Interpreter) tickLifetimes() {
	for name, v := range i.Env.store {
		if v.Lifetime > 0 {
			v.Lifetime--
		}

		if v.Lifetime == 0 {
			delete(i.Env.store, name)
			continue
		}

		i.Env.store[name] = v
	}
}

func (i *Interpreter) checkFuncStatement(fn *parser.FuncStatement) error {
	hasValueReturn := false
	hasEmptyReturn := false

	for _, stmt := range fn.Body {
		if r, ok := stmt.(*parser.ReturnStatement); ok {
			if len(r.Values) > 0 {
				hasValueReturn = true
			} else {
				hasEmptyReturn = true
			}
		}
	}

	if hasValueReturn && len(fn.ReturnTypes) == 0 {
		return NewRuntimeError(fn, "function returns a value but has no return type")
	}

	if hasEmptyReturn && len(fn.ReturnTypes) > 0 {
		return NewRuntimeError(fn, "missing return value")
	}

	if len(fn.ReturnTypes) > 0 && !hasValueReturn {
		return NewRuntimeError(fn, "function must return a value")
	}

	return nil
}

func (i *Interpreter) checkFuncLiteral(fn *parser.FuncLiteral) error {
	hasValueReturn := false
	hasEmptyReturn := false

	for _, stmt := range fn.Body {
		if r, ok := stmt.(*parser.ReturnStatement); ok {
			if len(r.Values) > 0 {
				hasValueReturn = true
			} else {
				hasEmptyReturn = true
			}
		}
	}

	if hasValueReturn && len(fn.ReturnTypes) == 0 {
		return NewRuntimeError(fn, "function returns a value but has no return type")
	}

	if hasEmptyReturn && len(fn.ReturnTypes) > 0 {
		return NewRuntimeError(fn, "missing return value")
	}

	if len(fn.ReturnTypes) > 0 && !hasValueReturn {
		return NewRuntimeError(fn, "function must return a value")
	}

	return nil
}

func (i *Interpreter) checkMethodStatement(fn *parser.MethodStatement) error {
	hasValueReturn := false
	hasEmptyReturn := false

	for _, stmt := range fn.Body {
		if r, ok := stmt.(*parser.ReturnStatement); ok {
			if len(r.Values) > 0 {
				hasValueReturn = true
			} else {
				hasEmptyReturn = true
			}
		}
	}

	if hasValueReturn && len(fn.ReturnTypes) == 0 {
		return NewRuntimeError(fn, "method returns a value but has no return type")
	}

	if hasEmptyReturn && len(fn.ReturnTypes) > 0 {
		return NewRuntimeError(fn, "missing return value")
	}

	if len(fn.ReturnTypes) > 0 && !hasValueReturn {
		return NewRuntimeError(fn, "method must return a value")
	}

	return nil
}

func runValidator(t *TypeInfo, fields map[string]Value) error {
	if t.Validator != nil {
		return t.Validator(fields)
	}
	if t.Underlying != nil && t.Underlying.Validator != nil {
		return t.Underlying.Validator(fields)
	}
	return nil
}
