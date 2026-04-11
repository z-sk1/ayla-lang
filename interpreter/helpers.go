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
		store:    make(map[string]*Variable),
		builtins: make(map[string]*BuiltinFunc),
		defers:   make([]*parser.FuncCall, 0),
		mu:       sync.RWMutex{},
	}

	TypeEnv := make(map[string]TypeValue)

	wd, _ := os.Getwd()

	i := &Interpreter{
		Env:          env,
		pointerCache: make(map[*TypeInfo]*TypeInfo),
		currentDir:   dir,
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
	initBuiltinTypes(TypeEnv)

	i.TypeEnv = TypeEnv

	return i
}

func NewWithEnv(env *Environment, path string) *Interpreter {
	TypeEnv := make(map[string]TypeValue)

	dir := filepath.Dir(path)

	wd, _ := os.Getwd()

	i := &Interpreter{
		Env:          env,
		pointerCache: make(map[*TypeInfo]*TypeInfo),
		currentDir:   dir,
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
	initBuiltinTypes(TypeEnv)

	i.TypeEnv = TypeEnv

	return i
}

func NewEnvironment(parent *Environment) *Environment {
	return &Environment{
		store:    make(map[string]*Variable),
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

func (e *Environment) Get(name string) (Value, bool, bool) {
	e.mu.RLock()
	v, ok := e.store[name]
	e.mu.RUnlock()

	if ok {
		return v.Value, true, v.isConst
	}

	if e.parent != nil {
		return e.parent.Get(name)
	}

	return nil, false, false
}

func (e *Environment) GetLocal(name string) (Value, bool, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	v, ok := e.store[name]
	if ok {
		return v.Value, true, v.isConst
	}

	return nil, false, false
}

func (e *Environment) GetVar(name string) (*Variable, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	v, ok := e.store[name]
	if ok {
		return v, true
	}

	if e.parent != nil {
		return e.parent.GetVar(name)
	}

	return nil, false
}

func (e *Environment) Define(name string, val Value, isConst bool) Value {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.store[name] = &Variable{Value: val, Lifetime: -1, isConst: isConst}
	return val
}

func (e *Environment) DefineWithLifetime(name string, val Value, lifetime int, isConst bool) Value {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.store[name] = &Variable{Value: val, Lifetime: lifetime, isConst: isConst}
	return val
}

func (e *Environment) Set(name string, val Value) Value {
	e.mu.Lock()
	defer e.mu.Unlock()

	if v, ok := e.store[name]; ok {
		v.Value = val // update existing variable
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

	if typ.Methods == nil {
		typ.Methods = make(map[string]*Func)
	}

	typ.Methods[name] = fn

	if typ.MethodTypes == nil {
		typ.MethodTypes = make(map[string]*TypeInfo)
	}

	typ.MethodTypes[name] = fn.TypeName
}

func (e *Environment) GetMethod(typ *TypeInfo, name string) (*Func, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if typ.Methods == nil {
		return nil, false
	}

	fn, ok := typ.Methods[name]
	return fn, ok
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

func (i *Interpreter) pointerTo(t *TypeInfo) *TypeInfo {
	if pt, ok := i.pointerCache[t]; ok {
		return pt
	}

	pt := &TypeInfo{
		Name: "*" + t.Name,
		Kind: TypePointer,
		Elem: t,
	}

	i.pointerCache[t] = pt
	return pt
}

func aylaValueToGoValue(v Value) any {
	switch val := v.(type) {
	case IntValue:
		return val.V
	case FloatValue:
		return val.V
	case StringValue:
		return val.V
	case BoolValue:
		return val.V
	default:
		return v.String()
	}
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

func TypesAssignable(from, to *TypeInfo) bool {
	from = UnwrapAlias(from)
	to = UnwrapAlias(to)

	if from == nil || to == nil {
		return false
	}

	if typesIdentical(from, to) {
		return true
	}

	if to.Kind == TypeInterface {
		return implementsInterface(from, to)
	}

	if to.Kind == TypeInterface {
		return implementsInterface(from, to)
	}
	if to.Kind == TypeNamed && to.Underlying != nil && to.Underlying.Kind == TypeInterface {
		return implementsInterface(from, to.Underlying)
	}

	if from.Kind == to.Kind && (from.Kind == TypeInt || from.Kind == TypeFloat) {
		if from.Min == nil && from.Max == nil {
			return true
		}

		if rangeMismatch(from, to) {
			return false
		}

		return true
	}

	if from.Kind == TypeNil && to.Kind == TypePointer {
		return true
	}

	if from.Kind == TypeNamed {
		if to.Kind == TypeInterface {
			return implementsInterface(from, to)
		}
		if to.Kind == TypeNamed && to.Underlying != nil && to.Underlying.Kind == TypeInterface {
			return implementsInterface(from, to.Underlying)
		}
		return typesIdentical(from, to)
	}

	switch {

	case from.Kind == TypePointer && to.Kind == TypePointer:
		return typesIdentical(from.Elem, to.Elem)

	case from.Kind == TypeArray && to.Kind == TypeArray:
		return TypesAssignable(from.Elem, to.Elem)

	case from.Kind == TypeFixedArray && to.Kind == TypeFixedArray:
		return TypesAssignable(from.Elem, to.Elem) &&
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
	case TypeInt, TypeFloat:
		if a.Min != nil || b.Min != nil {
			if a.Min == nil || b.Min == nil || *a.Min != *b.Min {
				return false
			}
		}
		if a.Max != nil || b.Max != nil {
			if a.Max == nil || b.Max == nil || *a.Max != *b.Max {
				return false
			}
		}
		return true

	case TypeString, TypeBool:
		return true

	case TypeInterface:
		return implementsInterface(a, b)

	case TypePointer:
		return typesIdentical(a.Elem, b.Elem)

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
		return a == b

	case TypeNamed:
		return a == b

	default:
		return false
	}
}

func (i *Interpreter) promoteValueToType(v Value, ti *TypeInfo) Value {
	base := UnwrapAlias(ti)

	switch val := v.(type) {

	case UntypedValue:
		converted := i.promoteValueToType(val.Value, ti)

		if ti.Kind == TypeNamed {
			return NamedValue{
				TypeName: ti,
				Value:    converted,
			}
		}

		return converted
	}

	if base.Kind == TypeNamed && base.Underlying != nil {
		base = base.Underlying
	}
	if base.Kind == TypeInterface {
		return InterfaceValue{
			TypeInfo: ti,
			Value:    v,
		}
	}

	actual := i.TypeInfoFromValue(v)

	if actual == ti {
		return v
	}

	if ti.Kind == TypeNamed {
		return NamedValue{
			TypeName: ti,
			Value:    v,
		}
	}

	switch base.Kind {

	case TypeArray:
		if arr, ok := v.(ArrayValue); ok {
			return ArrayValue{
				Elements: arr.Elements,
				ElemType: base.Elem,
				Capacity: arr.Capacity,
				Fixed:    arr.Fixed,
			}
		}

	case TypeFloat:
		if iv, ok := v.(IntValue); ok {
			return FloatValue{V: float64(iv.V)}
		}

	case TypeInt:
		if fv, ok := v.(FloatValue); ok {
			return IntValue{V: int(fv.V)}
		}
	}

	return v
}

func implementsInterface(ti, iface *TypeInfo) bool {
	for name, expected := range iface.MethodTypes {
		actual, ok := ti.MethodTypes[name]
		if !ok {
			m, ok2 := ti.Methods[name]
			if !ok2 {
				return false
			}
			actual = m.TypeName
		}
		if !TypesAssignable(actual, expected) {
			return false
		}
	}
	return true
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

func UnwrapAlias(t *TypeInfo) *TypeInfo {
	for t != nil && t.Alias {
		t = t.Underlying
	}
	return t
}

func UnwrapUntyped(v Value) Value {
	if v, ok := v.(UntypedValue); ok {
		return v.Value
	}
	return v
}

func UnwrapFully(v Value) Value {
	for {
		switch val := v.(type) {
		case NamedValue:
			v = val.Value
		case UntypedValue:
			v = val.Value
		case InterfaceValue:
			v = val.Value
		default:
			return v
		}
	}
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

func (i *Interpreter) assignInput(node parser.Node, ass Assignable, val Value, input string, name string) error {
	switch val.(type) {

	case IntValue:
		n, err := strconv.Atoi(input)
		if err != nil {
			return NewRuntimeError(node, fmt.Sprintf("%s: invalid int input", name))
		}
		ass.Set(i, IntValue{V: n})

	case FloatValue:
		f, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return NewRuntimeError(node, fmt.Sprintf("%s: invalid float input", name))
		}
		ass.Set(i, FloatValue{V: f})

	case BoolValue:
		b, err := strconv.ParseBool(input)
		if err != nil {
			return NewRuntimeError(node, fmt.Sprintf("%s: invalid bool input", name))
		}
		ass.Set(i, BoolValue{V: b})

	case StringValue:
		ass.Set(i, StringValue{V: input})

	default:
		err := inferAndAssign(ass, input, i)
		if err != nil {
			return NewRuntimeError(node, fmt.Sprintf("%s: %s", name, err.Error()))
		}

		return nil
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

func MapKey(v Value) string {
	v = UnwrapFully(v)
	switch x := v.(type) {

	case IntValue:
		return fmt.Sprintf("i:%d", x.V)

	case StringValue:
		return "s:" + x.V

	case BoolValue:
		return fmt.Sprintf("b:%v", x.V)

	case *PointerValue:
		return fmt.Sprintf("p:%p", x.Target)

	case EnumValue:
		return fmt.Sprintf("e:%s:%d", x.Enum.Name, x.Index)

	default:
		panic("unhashable map key")
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

func (i *Interpreter) assignWithType(node parser.Node, v Value, expected *TypeInfo) (Value, error) {
	if expected == nil {
		return v, nil
	}

	baseExpected := UnwrapAlias(expected)

	// promote literals first
	if uv, ok := v.(UntypedValue); ok {
		promoted := i.promoteValueToType(uv, expected)
		inner := UnwrapFully(promoted)
		innerTI := UnwrapAlias(i.TypeInfoFromValue(inner))
		if baseExpected.Kind == TypeInterface && len(baseExpected.MethodTypes) > 0 {
			if !implementsInterface(innerTI, baseExpected) {
				return NilValue{}, NewRuntimeError(node,
					fmt.Sprintf("type '%s' does not implement '%s'",
						innerTI.Name, expected.Name))
			}
		}
		v = promoted
	}

	actual := UnwrapAlias(i.TypeInfoFromValue(v))

	if baseExpected.Kind == TypeInterface || (baseExpected.Kind == TypeNamed && baseExpected.Underlying != nil && baseExpected.Underlying.Kind == TypeInterface) {
		iface := baseExpected
		if baseExpected.Kind == TypeNamed {
			iface = baseExpected.Underlying
		}
		actual := UnwrapAlias(i.TypeInfoFromValue(v))
		inner := UnwrapFully(v)
		innerTI := UnwrapAlias(i.TypeInfoFromValue(inner))

		if !implementsInterface(innerTI, iface) {
			for name := range iface.MethodTypes {
				_, inMethodTypes := innerTI.MethodTypes[name]
				_, inMethods := innerTI.Methods[name]
				if !inMethodTypes && !inMethods {
					return NilValue{}, NewRuntimeError(node,
						fmt.Sprintf("type '%s' does not implement '%s' (missing method '%s')",
							actual.Name, expected.Name, name))
				}
			}
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("type '%s' does not implement '%s'",
					actual.Name, expected.Name))
		}
	}

	if !TypesAssignable(actual, baseExpected) {
		if node == nil {
			return NilValue{}, fmt.Errorf(
				"type mismatch: expected '%s' but got '%s'",
				expected.Name,
				actual.Name,
			)
		}

		return NilValue{}, NewRuntimeError(
			node,
			fmt.Sprintf(
				"type mismatch: expected '%s' but got '%s'",
				expected.Name,
				actual.Name,
			),
		)
	}

	v = i.promoteValueToType(v, expected)

	if err := validateRange(node, v, baseExpected); err != nil {
		return NilValue{}, err
	}

	if arr, ok := v.(ArrayValue); ok && baseExpected.Kind == TypeArray {
		for _, el := range arr.Elements {
			if err := validateRange(node, el, baseExpected.Elem); err != nil {
				return NilValue{}, err
			}
		}
	}

	if baseExpected.Min != nil || baseExpected.Max != nil {
		switch val := v.(type) {
		case IntValue:
			val.TypeInfo = expected
			v = val
		case FloatValue:
			val.TypeInfo = expected
			v = val
		}
	}

	return v, nil
}

func (i *Interpreter) paramWithType(node parser.Node, pname string, v Value, expected *TypeInfo) (Value, error) {
	if expected == nil {
		return v, nil
	}

	baseExpected := UnwrapAlias(expected)

	// promote literals first
	if uv, ok := v.(UntypedValue); ok {
		promoted := i.promoteValueToType(uv, expected)
		inner := UnwrapFully(promoted)
		innerTI := UnwrapAlias(i.TypeInfoFromValue(inner))
		if baseExpected.Kind == TypeInterface && len(baseExpected.MethodTypes) > 0 {
			if !implementsInterface(innerTI, baseExpected) {
				return NilValue{}, NewRuntimeError(node,
					fmt.Sprintf("param '%s' of type '%s' does not implement '%s'",
						pname, innerTI.Name, expected.Name))
			}
		}
		v = promoted
	}

	actual := UnwrapAlias(i.TypeInfoFromValue(v))

	if baseExpected.Kind == TypeInterface || (baseExpected.Kind == TypeNamed && baseExpected.Underlying != nil && baseExpected.Underlying.Kind == TypeInterface) {
		iface := baseExpected
		if baseExpected.Kind == TypeNamed {
			iface = baseExpected.Underlying
		}
		actual := UnwrapAlias(i.TypeInfoFromValue(v))
		inner := UnwrapFully(v)
		innerTI := UnwrapAlias(i.TypeInfoFromValue(inner))

		if !implementsInterface(innerTI, iface) {
			for name := range iface.MethodTypes {
				_, inMethodTypes := innerTI.MethodTypes[name]
				_, inMethods := innerTI.Methods[name]
				if !inMethodTypes && !inMethods {
					return NilValue{}, NewRuntimeError(node,
						fmt.Sprintf("param '%s' of type '%s' does not implement '%s' (missing method '%s')",
							pname, actual.Name, expected.Name, name))
				}
			}
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("param '%s' of type '%s' does not implement '%s'",
					pname, actual.Name, expected.Name))
		}
	}

	if !TypesAssignable(actual, baseExpected) {
		if node == nil {
			return NilValue{}, fmt.Errorf(
				"type mismatch: param '%s' expected '%s' but got '%s'",
				pname,
				expected.Name,
				actual.Name,
			)
		}

		return NilValue{}, NewRuntimeError(
			node,
			fmt.Sprintf(
				"type mismatch: param '%s' expected '%s' but got '%s'",
				pname,
				expected.Name,
				actual.Name,
			),
		)
	}

	v = i.promoteValueToType(v, expected)

	if err := validateRange(node, v, baseExpected); err != nil {
		return NilValue{}, err
	}

	if arr, ok := v.(ArrayValue); ok && baseExpected.Kind == TypeArray {
		for _, el := range arr.Elements {
			if err := validateRange(node, el, baseExpected.Elem); err != nil {
				return NilValue{}, err
			}
		}
	}

	if baseExpected.Min != nil || baseExpected.Max != nil {
		switch val := v.(type) {
		case IntValue:
			val.TypeInfo = expected
			v = val
		case FloatValue:
			val.TypeInfo = expected
			v = val
		}
	}

	return v, nil
}

func validateRange(node parser.Node, v Value, expected *TypeInfo) error {
	if expected.Min != nil || expected.Max != nil {
		switch val := v.(type) {
		case IntValue:
			x := float64(val.V)

			if expected.Min != nil && x < *expected.Min {
				return NewRuntimeError(node, fmt.Sprintf("value %v below minimum %v", x, *expected.Min))
			}

			if expected.Max != nil && x > *expected.Max {
				return NewRuntimeError(node, fmt.Sprintf("value %v above maximum %v", x, *expected.Max))
			}

		case FloatValue:
			x := val.V

			if expected.Min != nil && x < *expected.Min {
				return NewRuntimeError(node, fmt.Sprintf("value %v below minimum %v", x, *expected.Min))
			}

			if expected.Max != nil && x > *expected.Max {
				return NewRuntimeError(node, fmt.Sprintf("value %v above maximum %v", x, *expected.Max))
			}
		}
	}

	return nil
}

func rangeMismatch(src, dst *TypeInfo) bool {
	if dst.Min == nil && dst.Max == nil {
		return false
	}

	// src must be inside dst

	if dst.Min != nil {
		if src.Min == nil || *src.Min < *dst.Min {
			return true
		}
	}

	if dst.Max != nil {
		if src.Max == nil || *src.Max > *dst.Max {
			return true
		}
	}

	return false
}

func inferAndAssign(ass Assignable, input string, i *Interpreter) error {
	if n, err := strconv.Atoi(input); err == nil {
		return ass.Set(i, IntValue{V: n})
	}
	if f, err := strconv.ParseFloat(input, 64); err == nil {
		return ass.Set(i, FloatValue{V: f})
	}
	if input == "yes" {
		return ass.Set(i, BoolValue{V: true})
	}
	if input == "no" {
		return ass.Set(i, BoolValue{V: false})
	}
	return ass.Set(i, StringValue{V: input})
}

func resolveAssignableArg(arg Value) (Assignable, bool) {
	if ptr, ok := arg.(*PointerValue); ok {
		return PointerTarget{Ptr: ptr}, true
	}
	ass, ok := arg.(Assignable)
	return ass, ok
}

func (i *Interpreter) resolveAssignableTarget(expr parser.Expression) (Assignable, error) {

	switch e := expr.(type) {

	case *parser.Identifier:
		v, ok := i.Env.GetVar(e.Value)
		if !ok {
			return nil, fmt.Errorf("undefined variable: %s", e.Value)
		}

		return VariableTarget{
			Name: e.Value,
			Var:  v,
		}, nil

	case *parser.MemberExpression:

		objVal, err := i.EvalExpression(e.Left)
		if err != nil {
			return nil, err
		}

		// pointer auto deref
		if ptr, ok := objVal.(*PointerValue); ok {
			objVal, err = ptr.Target.Get(i)
			if err != nil {
				return nil, err
			}
		}

		structVal, ok := objVal.(*StructValue)
		if !ok {
			return nil, fmt.Errorf("cannot assign field on non-struct, got %s", i.TypeInfoFromValue(objVal).Name)
		}

		if _, ok := structVal.Fields[e.Field.Value]; !ok {
			return nil, fmt.Errorf("unknown struct field: %s", e.Field.Value)
		}

		structTI := structVal.TypeName
		if structTI.Kind == TypeNamed {
			structTI = structTI.Underlying
		}

		fieldType, ok := structTI.Fields[e.Field.Value]
		if !ok {
			return nil, fmt.Errorf("unknown struct field: %s", e.Field.Value)
		}

		return MemberTarget{
			Struct:    structVal,
			Field:     e.Field.Value,
			FieldType: UnwrapAlias(fieldType),
		}, nil

	case *parser.IndexExpression:
		var leftVal Value
		var err error

		if ident, ok := e.Left.(*parser.Identifier); ok {
			v, ok := i.Env.GetVar(ident.Value)
			if !ok {
				return nil, fmt.Errorf("undefined variable: %s", ident.Value)
			}
			leftVal = v.Value
		} else {
			leftVal, err = i.EvalExpression(e.Left)
			if err != nil {
				return nil, err
			}
		}

		indexVal, err := i.EvalExpression(e.Index)
		indexVal = UnwrapFully(indexVal)

		switch val := leftVal.(type) {

		case MapValue:

			keyType := UnwrapAlias(i.TypeInfoFromValue(indexVal))

			if val.KeyType.Kind == TypeInterface {

				if !isComparableValue(indexVal) {
					return nil, fmt.Errorf("value of this type cannot be used as map key")
				}

			} else {

				if !TypesAssignable(keyType, val.KeyType) {
					return nil, fmt.Errorf(
						"map index expected %s but got %s",
						val.KeyType.Name,
						keyType.Name,
					)
				}

				if err := validateRange(e, indexVal, val.KeyType); err != nil {
					return nil, err
				}
			}

			return MapIndexTarget{
				Map:       &val,
				Key:       indexVal,
				KeyType:   val.KeyType,
				ValueType: val.ValueType,
			}, nil

		case ArrayValue:

			idxVal, ok := indexVal.(IntValue)
			if !ok {
				return nil, fmt.Errorf("index must be int")
			}

			idx := idxVal.V

			if idx < 0 || idx >= len(val.Elements) {
				return nil, fmt.Errorf("index: %d out of bounds", idx)
			}

			return ArrayIndexTarget{
				Array:    &val,
				Index:    idx,
				ElemType: val.ElemType,
			}, nil
		}

	case *parser.PrefixExpression:
		if e.Operator == "*" {
			ptrVal, err := i.EvalExpression(e.Right)
			if err != nil {
				return nil, err
			}

			ptr, ok := ptrVal.(*PointerValue)
			if !ok {
				return nil, fmt.Errorf("cannot assign through non-pointer")
			}

			return PointerTarget{
				Ptr: ptr,
			}, nil
		}
	}
	return nil, fmt.Errorf("invalid assignment target")
}

func copyValue(v Value) Value {
	switch val := v.(type) {

	case *StructValue:
		newFields := map[string]Value{}
		for k, f := range val.Fields {
			newFields[k] = copyValue(f)
		}
		return &StructValue{
			TypeName: val.TypeName,
			Fields:   newFields,
			Native:   val.Native,
		}

	case ArrayValue:
		newArr := make([]Value, len(val.Elements))
		for i, e := range val.Elements {
			newArr[i] = copyValue(e)
		}
		return ArrayValue{
			Elements: newArr,
			ElemType: val.ElemType,
			Capacity: val.Capacity,
			Fixed:    val.Fixed,
		}

	default:
		return v
	}
}
