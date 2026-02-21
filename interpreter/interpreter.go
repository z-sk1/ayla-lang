package interpreter

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"os"
	"sync"

	"github.com/z-sk1/ayla-lang/parser"
	"golang.org/x/term"
)

type RuntimeError struct {
	Message string
	Line    int
	Column  int
}

type Variable struct {
	Value    Value
	Lifetime int
}

type Environment struct {
	store    map[string]Variable
	methods  map[*TypeInfo]map[string]*Func
	builtins map[string]*BuiltinFunc
	defers   []*parser.FuncCall

	mu     sync.RWMutex
	parent *Environment
}

type Interpreter struct {
	env     *Environment
	typeEnv map[string]*TypeInfo
}

func New() *Interpreter {
	env := &Environment{
		store:    make(map[string]Variable),
		methods:  make(map[*TypeInfo]map[string]*Func),
		builtins: make(map[string]*BuiltinFunc),
		defers:   make([]*parser.FuncCall, 0),
	}

	typeEnv := make(map[string]*TypeInfo)

	i := &Interpreter{
		env: env,
	}

	i.registerBuiltins()
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
	}
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("runtime error at %d:%d: %s\n", e.Line, e.Column, e.Message)
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
	if e.methods[typ] == nil {
		e.methods[typ] = map[string]*Func{}
	}
	e.methods[typ][name] = fn
}

func (e *Environment) GetMethod(typ *TypeInfo, name string) (*Func, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for env := e; env != nil; env = env.parent {
		if m := env.methods[typ]; m != nil {
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

	if from == to {
		return true
	}

	if to.Kind == TypeAny {
		return true
	}

	// aliases are transparent
	if from.Alias {
		return typesAssignable(from.Underlying, to)
	}
	if to.Alias {
		return typesAssignable(from, to.Underlying)
	}

	// arrays: element types must be assignable
	if from.Kind == TypeArray && to.Kind == TypeArray {
		return typesAssignable(from.Elem, to.Elem)
	}

	// maps: keys and values must be assignable
	if from.Kind == TypeMap && to.Kind == TypeMap {
		return typesAssignable(from.Key, to.Key) &&
			typesAssignable(from.Value, to.Value)
	}

	if from.Kind == TypeEnum && to.Kind == TypeEnum {
		return from == to
	}

	if from.Kind == TypeFunc && to.Kind == TypeFunc {
		if len(from.Params) != len(to.Params) {
			return false
		}

		if len(from.Returns) != len(to.Returns) {
			return false
		}

		for i := range from.Params {
			if !typesAssignable(from.Params[i], to.Params[i]) {
				return false
			}
		}

		for i := range from.Returns {
			if !typesAssignable(from.Returns[i], to.Returns[i]) {
				return false
			}
		}

		return true
	}

	// named types: nominal typing
	if from.Kind == TypeNamed || to.Kind == TypeNamed {
		return from == to
	}

	// numeric widening
	if from.Kind == TypeInt && to.Kind == TypeFloat {
		return true
	}

	return false
}

func promoteValueToType(v Value, ti *TypeInfo) Value {
	ti = unwrapAlias(ti)

	switch v := v.(type) {

	case ArrayValue:
		if ti.Kind != TypeArray {
			return v
		}

		return ArrayValue{
			Elements: v.Elements,
			ElemType: ti.Elem, // attach declared element type
		}

	default:
		return v
	}
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

func initBuiltinTypes(typeEnv map[string]*TypeInfo) {
	typeEnv["int"] = &TypeInfo{
		Name:         "int",
		Kind:         TypeInt,
		IsComparable: true,
	}

	typeEnv["float"] = &TypeInfo{
		Name:         "float",
		Kind:         TypeFloat,
		IsComparable: true,
	}

	typeEnv["string"] = &TypeInfo{
		Name:         "string",
		Kind:         TypeString,
		IsComparable: true,
	}

	typeEnv["bool"] = &TypeInfo{
		Name:         "bool",
		Kind:         TypeBool,
		IsComparable: true,
	}

	typeEnv["nil"] = &TypeInfo{
		Name:         "nil",
		Kind:         TypeNil,
		IsComparable: true,
	}

	typeEnv["thing"] = &TypeInfo{
		Name: "thing",
		Kind: TypeAny,
	}

	typeEnv["error"] = &TypeInfo{
		Name: "error",
		Kind: TypeError,
	}
}

func (i *Interpreter) registerBuiltins() {
	env := i.env

	env.builtins["toInt"] = &BuiltinFunc{
		Name:  "toInt",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]
			v = unwrapNamed(v)

			switch v.Type() {
			case INT:
				return IntValue{V: v.(IntValue).V}, nil
			case FLOAT:
				return IntValue{V: int(v.(FloatValue).V)}, nil
			case BOOL:
				if v.(BoolValue).V {
					return IntValue{V: 1}, nil
				}
				return IntValue{V: 0}, nil
			case STRING:
				n, err := strconv.Atoi(v.(StringValue).V)
				if err != nil {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("could not parse string to int: %s", err.Error()))
				}
				return IntValue{V: n}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "unsupported toInt() parse")
			}
		},
	}

	env.builtins["toFloat"] = &BuiltinFunc{
		Name:  "toFloat",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]
			v = unwrapNamed(v)

			switch v.Type() {
			case FLOAT:
				return FloatValue{V: v.(FloatValue).V}, nil
			case INT:
				return FloatValue{V: float64(v.(IntValue).V)}, nil
			case BOOL:
				if v.(BoolValue).V {
					return FloatValue{V: 1.0}, nil
				}
				return FloatValue{V: 0.0}, nil
			case STRING:
				n, err := strconv.ParseFloat(v.(StringValue).V, 64)
				if err != nil {
					return NilValue{}, NewRuntimeError(node, "could not convert string to float")
				}
				return FloatValue{V: n}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "unsupported toFloat() parse")
			}
		},
	}

	env.builtins["toString"] = &BuiltinFunc{
		Name:  "toString",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]
			v = unwrapNamed(v)

			switch v.Type() {
			case INT,
				FLOAT,
				STRING,
				BOOL,
				ARR,
				STRUCT,
				TUPLE,
				NIL:
				return StringValue{V: v.String()}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "unsupported toString() parse")
			}

		},
	}

	env.builtins["toBool"] = &BuiltinFunc{
		Name:  "toBool",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]
			v = unwrapNamed(v)

			switch v.Type() {
			case BOOL:
				return BoolValue{V: v.(BoolValue).V}, nil
			case INT:
				return BoolValue{V: v.(IntValue).V != 0}, nil
			case FLOAT:
				return BoolValue{V: v.(FloatValue).V != 0}, nil
			case STRING:
				s := strings.ToLower(v.(StringValue).V)
				if s == "true" || s == "yes" || s == "1" {
					return BoolValue{V: true}, nil
				}
				if s == "false" || s == "no" || s == "0" || s == "" {
					return BoolValue{V: false}, nil
				}
				return NilValue{}, NewRuntimeError(node, "unsupported toBool() parse")
			default:
				return NilValue{}, NewRuntimeError(node, "unsupported toBool() parse")
			}
		},
	}

	env.builtins["toArr"] = &BuiltinFunc{
		Name:  "toArr",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			if len(args) == 0 {
				return NilValue{}, NewRuntimeError(node, "toArr requires at least one argument")
			}

			elemType := unwrapAlias(i.typeInfoFromValue(args[0]))

			elements := make([]Value, 0, len(args))
			elements = append(elements, args[0])

			for idx := 1; idx < len(args); idx++ {
				t := unwrapAlias(i.typeInfoFromValue(args[idx]))

				if !typesAssignable(t, elemType) {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("toArr argument %d expected %s but got %s (use []thing for mixed arrays)", idx, elemType.Name, t.Name))
				}

				elements = append(elements, args[idx])
			}

			return ArrayValue{
				Elements: elements,
				ElemType: elemType,
			}, nil
		},
	}

	env.builtins["ord"] = &BuiltinFunc{
		Name:  "ord",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			s, ok := args[0].(StringValue)
			if !ok {
				return NilValue{}, fmt.Errorf("ord expects string")
			}

			r := []rune(s.V)
			if len(r) != 1 {
				return NilValue{}, fmt.Errorf("ord expects single character")
			}

			return IntValue{V: int(r[0])}, nil
		},
	}

	env.builtins["chr"] = &BuiltinFunc{
		Name:  "chr",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			if len(args) != 1 {
				return NilValue{}, fmt.Errorf("chr expects 1 argument")
			}

			v, ok := args[0].(IntValue)
			if !ok {
				return NilValue{}, fmt.Errorf("chr expects int")
			}

			return StringValue{V: string(rune(v.V))}, nil
		},
	}

	env.builtins["len"] = &BuiltinFunc{
		Name:  "len",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]
			switch v.Type() {
			case STRING:
				return IntValue{V: len(v.(StringValue).V)}, nil
			case ARR:
				return IntValue{V: len(v.(ArrayValue).Elements)}, nil
			default:
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("len() not supported for type %s", v.Type()))
			}
		},
	}

	env.builtins["typeof"] = &BuiltinFunc{
		Name:  "typeof",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]

			switch v := v.(type) {
			case ArrayValue:
				return StringValue{V: "[]" + v.ElemType.Name}, nil
			}

			return StringValue{V: string(v.Type())}, nil
		},
	}

	env.builtins["put"] = &BuiltinFunc{
		Name:  "put",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			if len(args) == 0 {
				fmt.Print()
				return NilValue{}, nil
			}

			for _, v := range args {
				fmt.Print(v.String())
			}
			return NilValue{}, nil
		},
	}

	env.builtins["putln"] = &BuiltinFunc{
		Name:  "putln",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			if len(args) == 0 {
				fmt.Println()
				return NilValue{}, nil
			}

			if len(args) > 1 {
				val := args[1:]

				fmt.Print(args[0].String())

				for _, v := range val {
					fmt.Println(" " + v.String())
				}

				return NilValue{}, nil
			}

			fmt.Println(args[0].String())

			return NilValue{}, nil
		},
	}

	env.builtins["explode"] = &BuiltinFunc{
		Name:  "explode",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			var msg string

			switch v := args[0].(type) {
			case StringValue:
				msg = v.V
			case ErrorValue:
				msg = v.V.Error()
			default:
				msg = v.String()
			}

			return NilValue{}, NewRuntimeError(node, msg)
		},
	}

	env.builtins["scanln"] = &BuiltinFunc{
		Name:  "scanln",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			ident, ok := node.Args[0].(*parser.Identifier)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "scanln expects a variable")
			}

			varName := ident.Value

			// is it const?
			if v, ok := i.env.Get(varName); ok {
				if _, isConst := v.(ConstValue); isConst {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot assign to const: %s", varName))
				}
			} else {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown var: %s", varName))
			}

			var input string
			fmt.Scanln(&input)
			i.env.Set(varName, StringValue{V: input})

			return NilValue{}, nil
		},
	}

	env.builtins["scankey"] = &BuiltinFunc{
		Name:  "scankey",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			ident, ok := node.Args[0].(*parser.Identifier)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "scankey expects a variable")
			}

			varName := ident.Value

			// is it const?
			v, ok := i.env.Get(varName)
			if ok {
				if _, isConst := v.(ConstValue); isConst {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot assign to const: %s", varName))
				}
			} else {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown var: %s", varName))
			}

			ch, err := readKey()
			if err != nil {
				return NilValue{}, NewRuntimeError(node, err.Error())
			}

			expectedTI := i.typeInfoFromValue(v)
			expectedTI = unwrapAlias(expectedTI)

			var newVal Value
			switch expectedTI.Kind {
			case TypeString:
				newVal = StringValue{V: string(ch)}
			case TypeInt:
				newVal = IntValue{V: int(ch)}
			default:
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("scankey only supports int or string variables, got %s", string(v.Type())))
			}

			i.env.Set(varName, newVal)
			return NilValue{}, nil
		},
	}

	env.builtins["push"] = &BuiltinFunc{
		Name:  "push",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			arr, ok := args[0].(ArrayValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "push expects (arr, val)")
			}

			val := args[1]

			arr.Elements = append(arr.Elements, val)

			// write back
			if ident, ok := node.Args[0].(*parser.Identifier); ok {
				env.Set(ident.Value, arr)
			}

			return NilValue{}, nil
		},
	}

	env.builtins["pop"] = &BuiltinFunc{
		Name:  "pop",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			arr, ok := args[0].(ArrayValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "pop expects array")
			}

			if len(arr.Elements) == 0 {
				return NilValue{}, NewRuntimeError(node, "pop from empty array")
			}

			// get last element
			lastIdx := len(arr.Elements) - 1
			val := arr.Elements[lastIdx]

			// shrink array
			arr.Elements = arr.Elements[:lastIdx]

			// write back
			if ident, ok := node.Args[0].(*parser.Identifier); ok {
				env.Set(ident.Value, arr)
			}

			return val, nil
		},
	}

	env.builtins["insert"] = &BuiltinFunc{
		Name:  "insert",
		Arity: 3,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			arr, ok := args[0].(ArrayValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "insert expects (arr, index, val)")
			}

			idxVal, ok := args[1].(IntValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "insert expects (arr, index, val)")
			}
			idx := idxVal.V

			if idx < 0 || idx > len(arr.Elements) {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("insert index %d out of bounds", idx))
			}

			val := args[2]

			// actual insert
			arr.Elements = append(arr.Elements[:idx], append([]Value{val}, arr.Elements[idx:]...)...)

			// write back
			if ident, ok := node.Args[0].(*parser.Identifier); ok {
				env.Set(ident.Value, arr)
			}

			return NilValue{}, nil
		},
	}

	env.builtins["remove"] = &BuiltinFunc{
		Name:  "remove",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			arr, ok := args[0].(ArrayValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "remove expects (arr, index)")
			}

			idxVal, ok := args[1].(IntValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "remove expects (arr, index)")
			}
			idx := idxVal.V

			if idx < 0 || idx > len(arr.Elements) {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("remove index %d is out of bounds", idx))
			}

			removed := arr.Elements[idx]

			// remove element
			arr.Elements = append(arr.Elements[:idx], arr.Elements[idx+1:]...)

			if ident, ok := node.Args[0].(*parser.Identifier); ok {
				env.Set(ident.Value, arr)
			}

			return removed, nil
		},
	}

	env.builtins["clear"] = &BuiltinFunc{
		Name:  "clear",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			arr, ok := args[0].(ArrayValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "clear expects (arr)")
			}

			arr.Elements = arr.Elements[:0]

			if ident, ok := node.Args[0].(*parser.Identifier); ok {
				env.Set(ident.Value, arr)
			}

			return NilValue{}, nil
		},
	}

	env.builtins["wait"] = &BuiltinFunc{
		Name:  "wait",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			intVal, ok := args[0].(IntValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("wait arg: %v, must be int (milliseconds)", intVal.V))
			}

			ms := intVal.V
			if ms < 0 {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("wait duration: %d, cannot be negative", ms))
			}

			time.Sleep(time.Duration(ms) * time.Millisecond)
			return NilValue{}, nil
		},
	}

	env.builtins["randi"] = &BuiltinFunc{
		Name:  "randi",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			switch len(args) {
			case 0:
				n := rand.Intn(2)
				return IntValue{V: n}, nil
			case 1:
				maxV, ok := args[0].(IntValue)
				if !ok {
					return NilValue{}, NewRuntimeError(node, "randi expects (int <= 0)")
				}

				if maxV.V <= 0 {
					return NilValue{}, NewRuntimeError(node, "randi expects (int <= 0)")
				}

				max := maxV.V
				n := rand.Intn(max) + 1

				return IntValue{V: n}, nil
			case 2:
				minV, ok1 := args[0].(IntValue)
				maxV, ok2 := args[1].(IntValue)

				if !ok1 || !ok2 {
					return NilValue{}, NewRuntimeError(node, "randi expects 0-2 args")
				}

				min := minV.V
				max := maxV.V

				if min > max {
					min, max = max, min
				}

				n := rand.Intn(max-min+1) + min
				return IntValue{V: n}, nil
			}
			return NilValue{}, NewRuntimeError(node, "invalid amount of args, randi expects 0-2 args")
		},
	}

	env.builtins["randf"] = &BuiltinFunc{
		Name:  "randf",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			switch len(args) {
			case 0:
				n := rand.Float64()
				return FloatValue{V: n}, nil
			case 1:
				maxV, ok := toFloat(args[0])
				if !ok {
					return NilValue{}, NewRuntimeError(node, "randf expects 0-2 args")
				}

				n := rand.Float64() * maxV
				return FloatValue{V: n}, nil
			case 2:
				minV, ok1 := toFloat(args[0])
				maxV, ok2 := toFloat(args[1])

				if !ok1 || !ok2 {
					return NilValue{}, NewRuntimeError(node, "randf expects 0-2 args")
				}

				if minV > maxV {
					minV, maxV = maxV, minV
				}

				n := rand.Float64()*(maxV-minV+1) + minV
				return FloatValue{V: n}, nil
			}
			return NilValue{}, NewRuntimeError(node, "invalid amount of args, randf expects 0-2 args")
		},
	}

	env.builtins["sin"] = &BuiltinFunc{
		Name:  "sin",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			val := args[0]

			switch v := val.(type) {
			case IntValue:
				val = FloatValue{V: float64(v.V)}
			}

			return FloatValue{V: math.Sin(val.(FloatValue).V)}, nil
		},
	}

	env.builtins["cos"] = &BuiltinFunc{
		Name:  "cos",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			val := args[0]

			switch v := val.(type) {
			case IntValue:
				val = FloatValue{V: float64(v.V)}
			}

			return FloatValue{V: math.Cos(val.(FloatValue).V)}, nil
		},
	}
}

func unwrapStructType(t *TypeInfo) (*TypeInfo, error) {
	cur := t

	for {
		if cur.Kind == TypeStruct {
			return cur, nil
		}

		if cur.Kind == TypeNamed && cur.Underlying != nil {
			cur = cur.Underlying
			continue
		}

		return nil, fmt.Errorf("%s is not a struct type", t.Name)
	}
}

func (i *Interpreter) tickLifetimes() {
	for name, v := range i.env.store {
		if v.Lifetime > 0 {
			v.Lifetime--
		}

		if v.Lifetime == 0 {
			delete(i.env.store, name)
			continue
		}

		i.env.store[name] = v
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

func (i *Interpreter) EvalStatements(stmts []parser.Statement) (ControlSignal, error) {
	for _, s := range stmts {
		sig, err := i.EvalStatement(s)
		if err != nil {
			return SignalNone{}, err
		}

		switch sig.(type) {
		case SignalReturn, SignalBreak, SignalContinue, ErrorValue:
			return sig, nil
		}

		i.tickLifetimes()
	}
	return SignalNone{}, nil
}

func (i *Interpreter) EvalBlock(stmts []parser.Statement, newScope bool) (ControlSignal, error) {
	blockEnv := NewEnvironment(i.env)
	oldEnv := i.env

	if newScope {
		i.env = blockEnv
	}

	sig, err := i.EvalStatements(stmts)

	i.env = oldEnv
	return sig, err
}

func (i *Interpreter) EvalStatement(s parser.Statement) (ControlSignal, error) {
	if s == nil {
		return SignalNone{}, nil
	}

	switch stmt := s.(type) {
	case *parser.VarStatement:
		var val Value
		var err error

		var expectedTI *TypeInfo
		if stmt.Type != nil {
			expectedTI, err = i.resolveTypeNode(stmt.Type)
			if err != nil {
				return SignalNone{}, err
			}
			expectedTI = unwrapAlias(expectedTI)
		}

		if stmt.Value != nil {
			val, err = i.evalExprWithExpectedType(stmt.Value, expectedTI)
		} else if expectedTI != nil {
			val, err = i.defaultValueFromTypeInfo(stmt, expectedTI)
			if err != nil {
				return SignalNone{}, err
			}
		} else {
			val = UninitializedValue{}
		}

		if err != nil {
			return SignalNone{}, err
		}
		if v, ok := val.(ErrorValue); ok {
			return v, nil
		}

		val, err = i.assignWithType(stmt, val, expectedTI)
		if err != nil {
			return SignalNone{}, err
		}

		if stmt.Lifetime != nil {
			lifetime, err := i.EvalExpression(stmt.Lifetime)
			if err != nil {
				return SignalNone{}, err
			}

			if lifetime.(IntValue).V > 0 {
				i.env.DefineWithLifetime(stmt.Name.Value, val, lifetime.(IntValue).V+1) // +1 because the var statement itself also decrements it
				return SignalNone{}, nil
			}
		}

		i.env.Define(stmt.Name.Value, val)
		return SignalNone{}, nil

	case *parser.VarStatementBlock:
		for _, decl := range stmt.Decls {
			i.EvalStatement(decl)
		}

		return SignalNone{}, nil

	case *parser.VarStatementNoKeyword:
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return NilValue{}, err
		}

		// variable must not exist
		if _, ok := i.env.Get(stmt.Name.Value); ok {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("cant redeclare var: %s", stmt.Name.Value))
		}

		if stmt.Lifetime != nil {
			lifetime, err := i.EvalExpression(stmt.Lifetime)
			if err != nil {
				return SignalNone{}, err
			}

			if lifetime.(IntValue).V > 0 {
				i.env.DefineWithLifetime(stmt.Name.Value, val, lifetime.(IntValue).V+1) // +1 because the var statement itself also decrements it
				return SignalNone{}, nil
			}
		}

		i.env.Define(stmt.Name.Value, val)
		return SignalNone{}, nil

	case *parser.MultiVarStatement:
		if stmt.Values == nil {
			var expectedTI *TypeInfo
			var err error

			if stmt.Type != nil {
				expectedTI, err = i.resolveTypeNode(stmt.Type)
				if err != nil {
					return SignalNone{}, err
				}
			}

			for _, name := range stmt.Names {
				if _, ok := i.env.Get(name.Value); ok {
					return SignalNone{}, NewRuntimeError(stmt,
						fmt.Sprintf("cannot redeclare var: %s", name.Value))
				}

				var v Value
				if expectedTI != nil {
					v, err = i.defaultValueFromTypeInfo(stmt, expectedTI)
					if err != nil {
						return SignalNone{}, err
					}
				} else {
					v = UninitializedValue{}
				}

				if stmt.Lifetime != nil {
					lifetime, err := i.EvalExpression(stmt.Lifetime)
					if err != nil {
						return SignalNone{}, err
					}

					if lifetime.(IntValue).V > 0 {
						i.env.DefineWithLifetime(name.Value, v, lifetime.(IntValue).V+1) // +1 because the var statement itself also decrements it
					}
				} else {
					i.env.Define(name.Value, v)
				}
			}

			return SignalNone{}, nil
		}

		var values []Value

		if len(stmt.Values) == 1 {
			val, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
			}
			if v, ok := val.(ErrorValue); ok {
				return v, nil
			}

			if tup, ok := val.(TupleValue); ok {
				values = tup.Values
			} else {
				return SignalNone{}, NewRuntimeError(stmt, "multi assign expected multiple values")
			}
		} else {
			values = make([]Value, 0, len(stmt.Values))

			for idx, expr := range stmt.Values {
				v, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}
				if v, ok := v.(ErrorValue); ok {
					return v, nil
				}

				values[idx] = v
			}
		}

		if len(stmt.Values) != len(stmt.Names) {
			return SignalNone{}, NewRuntimeError(stmt,
				fmt.Sprintf("expected %d values, got %d",
					len(stmt.Names), len(stmt.Values)))
		}

		var expectedTI *TypeInfo
		var err error
		if stmt.Type != nil {
			expectedTI, err = i.resolveTypeNode(stmt.Type)
			if err != nil {
				return SignalNone{}, err
			}
		}

		for idx, name := range stmt.Names {
			if _, ok := i.env.Get(name.Value); ok {
				return SignalNone{}, NewRuntimeError(stmt,
					fmt.Sprintf("cannot redeclare var: %s", name.Value))
			}

			v, err := i.assignWithType(stmt, values[idx], expectedTI)
			if err != nil {
				return SignalNone{}, err
			}

			if stmt.Lifetime != nil {
				lifetimeVal, err := i.EvalExpression(stmt.Lifetime)
				if err != nil {
					return SignalNone{}, err
				}

				lifetime := lifetimeVal.(IntValue).V
				if lifetime > 0 {
					i.env.DefineWithLifetime(name.Value, v, lifetime+1)
				}
			} else {
				i.env.Define(name.Value, v)
			}
		}

	case *parser.MultiVarStatementNoKeyword:
		if len(stmt.Values) != len(stmt.Names) {
			return SignalNone{}, NewRuntimeError(stmt,
				fmt.Sprintf("expected %d values, got %d",
					len(stmt.Names), len(stmt.Values)))
		}

		var values []Value

		if len(stmt.Values) == 1 {
			val, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
			}
			if v, ok := val.(ErrorValue); ok {
				return v, nil
			}

			if tup, ok := val.(TupleValue); ok {
				values = tup.Values
			} else {
				return SignalNone{}, NewRuntimeError(stmt, "multi assign expected multiple values")
			}
		} else {
			values = make([]Value, 0, len(stmt.Values))

			for idx, expr := range stmt.Values {
				v, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}
				if v, ok := v.(ErrorValue); ok {
					return v, nil
				}

				values[idx] = v
			}
		}

		if len(stmt.Values) != len(stmt.Names) {
			return SignalNone{}, NewRuntimeError(stmt,
				fmt.Sprintf("expected %d values, got %d",
					len(stmt.Names), len(stmt.Values)))
		}

		var expectedTI *TypeInfo

		for idx, name := range stmt.Names {
			if _, ok := i.env.Get(name.Value); ok {
				return SignalNone{}, NewRuntimeError(stmt,
					fmt.Sprintf("cannot redeclare var: %s", name.Value))
			}

			v, err := i.assignWithType(stmt, values[idx], expectedTI)
			if err != nil {
				return SignalNone{}, err
			}

			if stmt.Lifetime != nil {
				lifetimeVal, err := i.EvalExpression(stmt.Lifetime)
				if err != nil {
					return SignalNone{}, err
				}

				lifetime := lifetimeVal.(IntValue).V
				if lifetime > 0 {
					i.env.DefineWithLifetime(name.Value, v, lifetime+1)
				}
			} else {
				i.env.Define(name.Value, v)
			}
		}

		return SignalNone{}, nil

	case *parser.ConstStatementBlock:
		for _, decl := range stmt.Decls {
			i.EvalStatement(decl)
		}

		return SignalNone{}, nil

	case *parser.ConstStatement:
		var val Value
		var err error

		var expectedTI *TypeInfo
		if stmt.Type != nil {
			expectedTI, err = i.resolveTypeNode(stmt.Type)
			if err != nil {
				return SignalNone{}, err
			}
		}

		if stmt.Value != nil {
			val, err = i.evalExprWithExpectedType(stmt.Value, expectedTI)
		} else if expectedTI != nil {
			val, err = i.defaultValueFromTypeInfo(stmt, expectedTI)
			if err != nil {
				return SignalNone{}, err
			}
		} else {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("const %s must be initalised with a value", stmt.Name.Value))
		}

		// check if variable already exist
		if _, ok := i.env.Get(stmt.Name.Value); ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("cant redeclare const: %s", stmt.Name.Value))
		}

		val, err = i.assignWithType(stmt, val, expectedTI)
		if err != nil {
			return SignalNone{}, err
		}

		if stmt.Lifetime != nil {
			lifetime, err := i.EvalExpression(stmt.Lifetime)
			if err != nil {
				return SignalNone{}, err
			}

			if lifetime.(IntValue).V > 0 {
				i.env.DefineWithLifetime(stmt.Name.Value, ConstValue{Value: val}, lifetime.(IntValue).V+1) // +1 because the var statement itself also decrements it
				return SignalNone{}, nil
			}
		}

		// store const val
		i.env.Define(stmt.Name.Value, ConstValue{Value: val})
		return SignalNone{}, nil

	case *parser.MultiConstStatement:
		if stmt.Values == nil {
			var names string

			for _, name := range stmt.Names {
				if name == stmt.Names[len(stmt.Names)-1] {
					names = names + name.Value
				} else {
					names = names + (name.Value + ", ")
				}
			}

			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("constants, %s, must be initialised", names))
		}

		var values []Value

		if len(stmt.Values) == 1 {
			val, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
			}
			if v, ok := val.(ErrorValue); ok {
				return v, nil
			}

			if tup, ok := val.(TupleValue); ok {
				values = tup.Values
			} else {
				return SignalNone{}, NewRuntimeError(stmt, "multi assign expected multiple values")
			}
		} else {
			values = make([]Value, 0, len(stmt.Values))

			for idx, expr := range stmt.Values {
				v, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}
				if v, ok := v.(ErrorValue); ok {
					return v, nil
				}

				values[idx] = v
			}
		}

		if len(stmt.Values) != len(stmt.Names) {
			return SignalNone{}, NewRuntimeError(stmt,
				fmt.Sprintf("expected %d values, got %d",
					len(stmt.Names), len(stmt.Values)))
		}

		var expectedTI *TypeInfo
		var err error
		if stmt.Type != nil {
			expectedTI, err = i.resolveTypeNode(stmt.Type)
			if err != nil {
				return SignalNone{}, err
			}
		}

		for idx, name := range stmt.Names {
			if _, ok := i.env.Get(name.Value); ok {
				return SignalNone{}, NewRuntimeError(stmt,
					fmt.Sprintf("cannot redeclare var: %s", name.Value))
			}

			v, err := i.assignWithType(stmt, values[idx], expectedTI)
			if err != nil {
				return SignalNone{}, err
			}

			if stmt.Lifetime != nil {
				lifetimeVal, err := i.EvalExpression(stmt.Lifetime)
				if err != nil {
					return SignalNone{}, err
				}

				lifetime := lifetimeVal.(IntValue).V
				if lifetime > 0 {
					i.env.DefineWithLifetime(name.Value, v, lifetime+1)
				}
			} else {
				i.env.Define(name.Value, v)
			}
		}

		return SignalNone{}, nil

	case *parser.TypeStatement:
		switch t := stmt.Type.(type) {
		case *parser.IdentType:
			underlying, ok := i.typeEnv[t.Name]
			if !ok {
				return SignalNone{}, NewRuntimeError(t, fmt.Sprintf("unknown type: %s", t.Name))
			}

			if stmt.Alias {
				i.typeEnv[stmt.Name.Value] = &TypeInfo{
					Name:       stmt.Name.Value,
					Kind:       underlying.Kind,
					Underlying: underlying,
					Alias:      true,
				}
			} else {
				i.typeEnv[stmt.Name.Value] = &TypeInfo{
					Name:       stmt.Name.Value,
					Kind:       TypeNamed,
					Underlying: underlying,
				}
			}

			return SignalNone{}, nil

		case *parser.StructType:
			fields := make(map[string]*TypeInfo)

			for _, f := range t.Fields {
				fieldTI, err := i.resolveTypeNode(f.Type)
				if err != nil {
					return SignalNone{}, NewRuntimeError(t, err.Error())
				}
				fields[f.Name.Value] = fieldTI
			}

			ti := &TypeInfo{
				Name:   stmt.Name.Value,
				Kind:   TypeStruct,
				Fields: fields,
			}

			if stmt.Alias {
				ti.Alias = true
			}

			i.typeEnv[stmt.Name.Value] = ti
			return SignalNone{}, nil
		}

	case *parser.AssignmentStatement:
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}
		if v, ok := val.(ErrorValue); ok {
			return v, nil
		}

		existingVal, ok := i.env.Get(stmt.Name.Value)
		if !ok {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("assignment to undefined variable: %s", stmt.Name.Value))
		}

		if _, isConst := existingVal.(ConstValue); isConst {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("cannot reassign to const: %s", stmt.Name.Value))
		}

		switch existingVal.(type) {
		case UninitializedValue:
			i.env.Set(stmt.Name.Value, val)
			return SignalNone{}, nil
		}

		expectedTI := unwrapAlias(i.typeInfoFromValue(existingVal))

		v, err := i.assignWithType(stmt, val, expectedTI)
		if err != nil {
			return SignalNone{}, err
		}

		i.env.Set(stmt.Name.Value, v)
		return SignalNone{}, nil

	case *parser.MultiAssignmentStatement:

		var values []Value

		if len(stmt.Values) == 1 {
			val, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
			}
			if v, ok := val.(ErrorValue); ok {
				return v, nil
			}

			if tup, ok := val.(TupleValue); ok {
				values = tup.Values
			} else {
				return SignalNone{}, NewRuntimeError(stmt, "multi assign expected multiple values")
			}
		} else {
			values = make([]Value, 0, len(stmt.Values))

			for idx, expr := range stmt.Values {
				v, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}
				if v, ok := v.(ErrorValue); ok {
					return v, nil
				}

				values[idx] = v
			}
		}

		if len(stmt.Values) != len(stmt.Names) {
			return SignalNone{}, NewRuntimeError(stmt,
				fmt.Sprintf("expected %d values, got %d",
					len(stmt.Names), len(stmt.Values)))
		}

		var expectedTI *TypeInfo

		for idx, name := range stmt.Names {
			if _, ok := i.env.Get(name.Value); ok {
				return SignalNone{}, NewRuntimeError(stmt,
					fmt.Sprintf("cannot redeclare var: %s", name.Value))
			}

			v, err := i.assignWithType(stmt, values[idx], expectedTI)
			if err != nil {
				return SignalNone{}, err
			}

			i.env.Set(name.Value, v)
		}

	case *parser.IndexAssignmentStatement:
		leftVal, err := i.EvalExpression(stmt.Left)
		if err != nil {
			return NilValue{}, err
		}

		arrVal, ok := leftVal.(ArrayValue)
		if !ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("assignment to non-array: %v", leftVal.String()))
		}

		index, err := i.EvalExpression(stmt.Index)
		if err != nil {
			return SignalNone{}, err
		}

		idxVal, ok := index.(IntValue)
		if !ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("array index: %v, must be int", idxVal.V))
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(arrVal.Elements) {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("array index: %d, out of bounds", idx))
		}

		newVal, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}

		if newVal == nil {
			return SignalNone{}, nil
		}

		arrVal.Elements[idx] = newVal
		return SignalNone{}, nil

	case *parser.MemberAssignmentStatement:
		objVal, err := i.EvalExpression(stmt.Object)
		if err != nil {
			return SignalNone{}, nil
		}

		structVal, ok := objVal.(*StructValue)
		if !ok {
			return SignalNone{}, NewRuntimeError(s, "cannot assign field on non-struct")
		}

		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}

		if _, ok := structVal.Fields[stmt.Field.Value]; !ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("unknown struct field: %s", stmt.Field.Value))
		}

		expectedType := structVal.TypeName.Fields[stmt.Field.Value]

		if val.Type() != valueTypeOf(expectedType) {
			return NilValue{}, NewRuntimeError(stmt, fmt.Sprintf("field '%s' expects %v but got %v", stmt.Field.Value, expectedType, val.Type()))
		}

		structVal.Fields[stmt.Field.Value] = val
		return SignalNone{}, nil

	case *parser.EnumStatement:
		if _, ok := i.env.Get(stmt.Name.Value); ok {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("cannot redeclare enum: %s", stmt.Name.Value))
		}

		enumType := &TypeInfo{
			Name:     stmt.Name.Value,
			Kind:     TypeEnum,
			Variants: make(map[string]int),
		}

		for idx, ident := range stmt.Variants {
			name := ident.Value

			if _, exists := enumType.Variants[name]; exists {
				return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("duplicate enum variant: %s", name))
			}

			enumType.Variants[name] = idx
		}

		i.typeEnv[stmt.Name.Value] = enumType

		i.env.Define(stmt.Name.Value, TypeValue{
			TypeName: enumType,
		})

		return SignalNone{}, nil

	case *parser.FuncStatement:
		paramTypes := make([]*TypeInfo, 0)
		paramNames := make([]string, 0)

		returnTypes := make([]*TypeInfo, 0)
		returnNames := make([]string, 0)

		err := i.checkFuncStatement(stmt)
		if err != nil {
			return SignalNone{}, err
		}

		for _, typ := range stmt.ReturnTypes {
			ti, err := i.resolveTypeNode(typ)
			if err != nil {
				return SignalNone{}, err
			}

			ti = unwrapAlias(ti)

			returnTypes = append(returnTypes, ti)
			paramNames = append(paramNames, ti.Name)
		}

		for _, param := range stmt.Params {
			ti, err := i.resolveTypeNode(param.Type)
			if err != nil {
				return SignalNone{}, err
			}

			ti = unwrapAlias(ti)

			paramTypes = append(paramTypes, ti)
			paramNames = append(paramNames, ti.Name)
		}

		typeInfo := &TypeInfo{
			Name:    fmt.Sprintf("fun(%s) (%s)", strings.Join(paramNames, ", "), strings.Join(returnNames, ", ")),
			Kind:    TypeFunc,
			Returns: returnTypes,
			Params:  paramTypes,
		}

		i.env.Define(stmt.Name.Value, &Func{Params: stmt.Params, Body: stmt.Body, TypeName: typeInfo, Env: i.env})
		return SignalNone{}, nil

	case *parser.MethodStatement:
		recvType, err := i.resolveTypeNode(stmt.Receiver.Type)
		if err != nil {
			return ErrorValue{
				V: NewRuntimeError(stmt, err.Error()),
			}, nil
		}

		params := append(
			[]*parser.ParametersClause{
				{
					Name: stmt.Receiver.Name,
					Type: stmt.Receiver.Type,
				},
			},
			stmt.Params...,
		)

		i.env.SetMethod(recvType, stmt.Name.Value, &Func{
			Params: params,
			Body:   stmt.Body,
			Env:    i.env,
		})

		return SignalNone{}, nil

	case *parser.ReturnStatement:
		values := []Value{}

		for _, expr := range stmt.Values {
			v, err := i.EvalExpression(expr)
			if err != nil {
				return SignalNone{}, err
			}
			values = append(values, v)
		}

		return SignalReturn{Values: values}, nil

	case *parser.ExpressionStatement:
		_, err := i.EvalExpression(stmt.Expression)
		if err != nil {
			return SignalNone{}, err
		}

		return SignalNone{}, nil

	case *parser.IfStatement:
		if stmt.Condition == nil {
			return SignalNone{}, NewRuntimeError(s, "if statement missing condition")
		}
		if stmt.Consequence == nil {
			return SignalNone{}, NewRuntimeError(s, "if statement missing consequence")
		}
		cond, err := i.EvalExpression(stmt.Condition)
		if err != nil {
			return SignalNone{}, err
		}

		truthy, err := isTruthy(cond)
		if err != nil {
			return SignalNone{}, NewRuntimeError(stmt, err.Error())
		}
		if truthy {
			if stmt.Consequence != nil {
				return i.EvalBlock(stmt.Consequence, true)
			}
		} else {
			if stmt.Alternative != nil {
				return i.EvalBlock(stmt.Alternative, true)
			}
		}
		return SignalNone{}, nil

	case *parser.SpawnStatement:
		go func() {
			defer func() {
				if r := recover(); r != nil {

				}
			}()

			i.EvalBlock(stmt.Body, true)
		}()

		return SignalNone{}, nil

	case *parser.SwitchStatement:
		switchVal, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}

		for _, c := range stmt.Cases {
			caseVal, err := i.EvalExpression(c.Expr)
			if err != nil {
				return SignalNone{}, err
			}

			if !valuesEqual(switchVal, caseVal) {
				continue
			}

			sig, err := i.EvalBlock(c.Body, true)
			if err != nil {
				return SignalNone{}, err
			}

			if _, ok := sig.(SignalNone); !ok {
				return sig, nil
			}

			return SignalNone{}, nil
		}

		if stmt.Default != nil {
			sig, err := i.EvalBlock(stmt.Default.Body, true)
			if err != nil {
				return SignalNone{}, err
			}
			if _, ok := sig.(SignalNone); !ok {
				return sig, nil
			}
		}

		return SignalNone{}, nil

	case *parser.WithStatement:
		val, err := i.EvalExpression(stmt.Expr)
		if err != nil {
			return SignalNone{}, err
		}

		oldEnv := i.env
		i.env = NewEnvironment(oldEnv)

		i.env.Define("it", ConstValue{Value: val})

		sig, err := i.EvalStatements(stmt.Body)

		i.env = oldEnv

		return sig, err

	case *parser.ForStatement:
		loopEnv := NewEnvironment(i.env)
		oldEnv := i.env
		i.env = loopEnv

		i.EvalStatement(stmt.Init)
		for {
			cond, err := i.EvalExpression(stmt.Condition)
			if err != nil {
				return SignalNone{}, err
			}

			truthy, err := isTruthy(cond)
			if err != nil {
				return SignalNone{}, NewRuntimeError(stmt, err.Error())
			}

			if !truthy {
				break
			}

			sig, err := i.EvalBlock(stmt.Body, false)
			if err != nil {
				return SignalNone{}, err
			}

			switch sig.(type) {
			case SignalBreak:
				return SignalNone{}, nil
			case SignalContinue:
				i.EvalStatement(stmt.Post)
				continue
			case SignalReturn:
				return sig, nil
			}
			i.EvalStatement(stmt.Post)
		}

		i.env = oldEnv
		return SignalNone{}, nil

	case *parser.ForRangeStatement:
		iterable, err := i.EvalExpression(stmt.Expr)
		if err != nil {
			return SignalNone{}, err
		}

		iterable = unwrapNamed(iterable)

		switch v := iterable.(type) {
		case ArrayValue:
			for idx, elem := range v.Elements {
				oldEnv := i.env
				i.env = NewEnvironment(oldEnv)

				if stmt.Key != nil && stmt.Key.Value != "_" {
					i.env.Define(stmt.Key.Value, IntValue{V: idx})
				}

				if stmt.Value != nil && stmt.Value.Value != "_" {
					i.env.Define(stmt.Value.Value, elem)
				}

				sig, err := i.EvalBlock(stmt.Body, false)

				i.env = oldEnv

				if err != nil {
					return SignalNone{}, err
				}

				switch sig.(type) {
				case SignalBreak:
					return SignalNone{}, nil
				case SignalContinue:
					continue
				case SignalReturn:
					return sig, nil
				}
			}
		case MapValue:
			for k, val := range v.Entries {
				oldEnv := i.env
				i.env = NewEnvironment(oldEnv)

				if stmt.Key != nil && stmt.Key.Value != "_" {
					i.env.Define(stmt.Key.Value, k)
				}

				if stmt.Value != nil && stmt.Value.Value != "_" {
					i.env.Define(stmt.Value.Value, val)
				}

				sig, err := i.EvalBlock(stmt.Body, false)

				i.env = oldEnv

				if err != nil {
					return SignalNone{}, err
				}

				switch sig.(type) {
				case SignalBreak:
					return SignalNone{}, nil
				case SignalContinue:
					continue
				case SignalReturn:
					return sig, nil
				}
			}
		case StringValue:
			for idx, s := range v.V {
				oldEnv := i.env
				i.env = NewEnvironment(oldEnv)

				if stmt.Key != nil && stmt.Key.Value != "_" {
					i.env.Define(stmt.Key.Value, IntValue{V: idx})
				}

				if stmt.Value != nil && stmt.Value.Value != "_" {
					i.env.Define(stmt.Value.Value, StringValue{V: string(s)})
				}

				sig, err := i.EvalBlock(stmt.Body, false)

				i.env = oldEnv

				if err != nil {
					return SignalNone{}, err
				}

				switch sig.(type) {
				case SignalBreak:
					return SignalNone{}, nil
				case SignalContinue:
					continue
				case SignalReturn:
					return sig, nil
				}
			}
		case IntValue:
			for idx := range v.V {
				oldEnv := i.env
				i.env = NewEnvironment(oldEnv)

				if stmt.Key != nil && stmt.Key.Value != "_" {
					i.env.Define(stmt.Key.Value, IntValue{V: idx})
				}

				if stmt.Value != nil && stmt.Value.Value != "_" {
					return SignalNone{}, NewRuntimeError(stmt, "integer range expects 1 variable")
				}

				sig, err := i.EvalBlock(stmt.Body, false)

				i.env = oldEnv

				if err != nil {
					return SignalNone{}, err
				}

				switch sig.(type) {
				case SignalBreak:
					return SignalNone{}, nil
				case SignalContinue:
					continue
				case SignalReturn:
					return sig, nil
				}
			}
		default:
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("range expects (array|map|int|string), but got %s", unwrapAlias(i.typeInfoFromValue(iterable)).Name))
		}

		return SignalNone{}, nil

	case *parser.WhileStatement:
		for {
			cond, err := i.EvalExpression(stmt.Condition)
			if err != nil {
				return SignalNone{}, err
			}

			truthy, err := isTruthy(cond)
			if err != nil {
				return SignalNone{}, NewRuntimeError(stmt, err.Error())
			}

			if !truthy {
				break
			}

			sig, err := i.EvalBlock(stmt.Body, true)
			if err != nil {
				return SignalNone{}, err
			}

			switch sig.(type) {
			case SignalBreak:
				return SignalNone{}, nil
			case SignalContinue:
				continue
			case SignalReturn:
				return sig, nil
			}
		}
		return SignalNone{}, nil

	case *parser.DeferStatement:
		i.env.AddDefer(stmt.Call)
		return SignalNone{}, nil

	case *parser.BreakStatement:
		return SignalBreak{}, nil

	case *parser.ContinueStatement:
		return SignalContinue{}, nil
	}

	return SignalNone{}, nil
}

func (i *Interpreter) EvalExpression(e parser.Expression) (Value, error) {
	if e == nil {
		return NilValue{}, nil
	}

	switch expr := e.(type) {
	case *parser.IntLiteral:
		return IntValue{V: expr.Value}, nil

	case *parser.FloatLiteral:
		return FloatValue{V: expr.Value}, nil

	case *parser.StringLiteral:
		return StringValue{V: expr.Value}, nil

	case *parser.BoolLiteral:
		return BoolValue{V: expr.Value}, nil

	case *parser.NilLiteral:
		return NilValue{}, nil

	case *parser.TupleLiteral:
		values := make([]Value, 0, len(expr.Values))

		for _, e := range expr.Values {
			v, err := i.EvalExpression(e)
			if err != nil {
				return NilValue{}, err
			}
			values = append(values, v)
		}

		return TupleValue{Values: values}, nil

	case *parser.MemberExpression:
		leftVal, err := i.EvalExpression(expr.Left)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := leftVal.(ErrorValue); ok {
			return v, nil
		}

		return evalMemberExpression(expr, leftVal, expr.Field.Value)

	case *parser.Identifier:
		if expr.Value == "_" {
			return ErrorValue{
				V: NewRuntimeError(expr, "cannot use '_' as a value"),
			}, nil
		}

		v, ok := i.env.Get(expr.Value)
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("undefined variable: %s", expr.Value)),
			}, nil
		}

		if c, isConst := v.(ConstValue); isConst {
			return c.Value, nil
		}

		return v, nil

	case *parser.ArrayLiteral:
		return i.evalArrayLiteral(expr, nil)

	case *parser.MapLiteral:
		return i.evalMapLiteral(expr, nil)

	case *parser.FuncLiteral:
		paramTypes := make([]*TypeInfo, 0)
		paramNames := make([]string, 0)

		returnTypes := make([]*TypeInfo, 0)
		returnNames := make([]string, 0)

		err := i.checkFuncLiteral(expr)
		if err != nil {
			return ErrorValue{
				V: err,
			}, nil
		}

		for _, typ := range expr.ReturnTypes {
			ti, err := i.resolveTypeNode(typ)
			if err != nil {
				return NilValue{}, err
			}

			ti = unwrapAlias(ti)

			returnTypes = append(returnTypes, ti)
			paramNames = append(paramNames, ti.Name)
		}

		for _, param := range expr.Params {
			ti, err := i.resolveTypeNode(param.Type)
			if err != nil {
				return NilValue{}, err
			}

			ti = unwrapAlias(ti)

			paramTypes = append(paramTypes, ti)
			returnNames = append(returnNames, ti.Name)
		}

		typeInfo := &TypeInfo{
			Name:    fmt.Sprintf("fun(%s) (%s)", strings.Join(paramNames, ", "), strings.Join(returnNames, ", ")),
			Kind:    TypeFunc,
			Returns: returnTypes,
			Params:  paramTypes,
		}

		return &Func{
			Params:   expr.Params,
			Body:     expr.Body,
			TypeName: typeInfo,
			Env:      i.env,
		}, nil

	case *parser.FuncCall:
		return i.evalCall(expr)

	case *parser.MethodCall:
		return i.evalMethodCall(expr)

	case *parser.IndexExpression:
		left, err := i.EvalExpression(expr.Left)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := left.(ErrorValue); ok {
			return v, nil
		}

		index, err := i.EvalExpression(expr.Index)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := index.(ErrorValue); ok {
			return v, nil
		}

		val, err := i.evalIndexExpression(expr, left, index)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := val.(ErrorValue); ok {
			return v, nil
		}

		return val, nil

	case *parser.StructLiteral:
		val, ok := i.typeEnv[expr.TypeName.Value]
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("unknown struct type %s", expr.TypeName.Value)),
			}, nil
		}

		structTI, err := unwrapStructType(val)
		if err != nil {
			return ErrorValue{
				V: NewRuntimeError(expr, err.Error()),
			}, nil
		}

		if structTI.Kind != TypeStruct {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("%s is not a struct type", structTI.Name)),
			}, nil
		}

		fields := make(map[string]Value)

		for name, e := range expr.Fields {
			expectedType, ok := structTI.Fields[name]
			if !ok {
				return ErrorValue{
					V: NewRuntimeError(
						expr,
						fmt.Sprintf(
							"unknown field '%s' in struct %s",
							name,
							structTI.Name,
						),
					),
				}, nil
			}

			v, err := i.EvalExpression(e)
			if err != nil {
				return NilValue{}, err
			}

			actualTI := unwrapAlias(i.typeInfoFromValue(v))
			expectedTI := unwrapAlias(expectedType)

			if !(typesAssignable(actualTI, expectedTI)) {
				return ErrorValue{
					V: NewRuntimeError(
						expr,
						fmt.Sprintf(
							"field '%s' expects %v but got %v",
							name,
							expectedType.Name,
							v.Type(),
						),
					),
				}, nil
			}

			fields[name] = v
		}

		return &StructValue{
			TypeName: structTI,
			Fields:   fields,
		}, nil

	case *parser.AnonymousStructLiteral:
		fields := make(map[string]Value)
		fieldTypes := make(map[string]*TypeInfo)

		for name, e := range expr.Fields {
			v, err := i.EvalExpression(e)
			if err != nil {
				return NilValue{}, err
			}

			if expected, ok := fieldTypes[name]; ok {
				actualTI := unwrapAlias(i.typeInfoFromValue(v))
				expectedTI := unwrapAlias(expected)

				if !(typesAssignable(actualTI, expectedTI)) {
					return ErrorValue{
						V: NewRuntimeError(
							expr,
							fmt.Sprintf(
								"field '%s' expects %v but got %v",
								name,
								expected.Name,
								string(v.Type()),
							),
						),
					}, nil
				}
			}

			fields[name] = v
			fieldTypes[name] = i.typeInfoFromValue(v)
		}

		return &StructValue{
			TypeName: &TypeInfo{
				Name:   "<anon>",
				Fields: fieldTypes,
			},
			Fields: fields,
		}, nil

	case *parser.TypeAssertExpression:
		val, err := i.EvalExpression(expr.Expr)
		if err != nil {
			return NilValue{}, err
		}

		staticTI := unwrapAlias(i.typeInfoFromValue(val))
		if staticTI.Kind != TypeAny {
			return NilValue{}, NewRuntimeError(expr, "type assertion only allowed on 'thing'")
		}

		targetTI, err := i.resolveTypeNode(expr.Type)
		if err != nil {
			return NilValue{}, err
		}

		inner := unwrapNamed(val)
		actualTI := unwrapAlias(i.typeInfoFromValue(inner))

		if !typesAssignable(actualTI, targetTI) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("type mismatch: %s asserted as %s", actualTI.Name, targetTI.Name)),
			}, nil
		}

		return promoteValueToType(inner, targetTI), nil

	case *parser.InfixExpression:
		if expr.Operator == "&&" {
			left, err := i.EvalExpression(expr.Left)
			if err != nil {
				return NilValue{}, err
			}

			lTruthy, err := isTruthy(left)
			if err != nil {
				return NilValue{}, NewRuntimeError(expr, err.Error())
			}

			if !lTruthy {
				return BoolValue{V: false}, nil
			}

			right, err := i.EvalExpression(expr.Right)
			if err != nil {
				return NilValue{}, err
			}

			rTruthy, err := isTruthy(right)
			if err != nil {
				return NilValue{}, NewRuntimeError(expr, err.Error())
			}

			return BoolValue{V: rTruthy}, nil
		}

		if expr.Operator == "||" {
			left, err := i.EvalExpression(expr.Left)
			if err != nil {
				return NilValue{}, err
			}

			lTruthy, err := isTruthy(left)
			if err != nil {
				return NilValue{}, NewRuntimeError(expr, err.Error())
			}

			if lTruthy {
				return BoolValue{V: true}, nil
			}

			right, err := i.EvalExpression(expr.Right)
			if err != nil {
				return NilValue{}, err
			}

			rTruthy, err := isTruthy(right)
			if err != nil {
				return NilValue{}, NewRuntimeError(expr, err.Error())
			}

			return BoolValue{V: rTruthy}, nil
		}

		left, err := i.EvalExpression(expr.Left)
		if err != nil {
			return NilValue{}, err
		}

		right, err := i.EvalExpression(expr.Right)
		if err != nil {
			return NilValue{}, err
		}

		return i.evalInfix(expr, left, expr.Operator, right)

	case *parser.PrefixExpression:
		right, err := i.EvalExpression(expr.Right)
		if err != nil {
			return NilValue{}, err
		}

		return evalPrefix(expr, expr.Operator, right)

	case *parser.GroupedExpression:
		return i.EvalExpression(expr.Expression)

	case *parser.InExpression:
		elem, err := i.EvalExpression(expr.Left)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := elem.(ErrorValue); ok {
			return v, nil
		}

		set, err := i.EvalExpression(expr.Right)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := set.(ErrorValue); ok {
			return v, nil
		}

		switch s := set.(type) {
		case MapValue:
			_, ok := s.Entries[elem]
			return BoolValue{V: ok}, nil

		case ArrayValue:
			for _, v := range s.Elements {
				if valuesEqual(v, elem) {
					return BoolValue{V: true}, nil
				}
			}
			return BoolValue{V: false}, nil
		case StringValue:
			if strings.Contains(s.V, elem.(StringValue).V) {
				return BoolValue{V: true}, nil
			}
			return BoolValue{V: false}, nil
		}

		return ErrorValue{
			V: NewRuntimeError(expr, "in expects map or array or string"),
		}, nil

	case *parser.InterpolatedString:
		is := expr
		var out strings.Builder

		for _, part := range is.Parts {
			val, err := i.EvalExpression(part)
			if err != nil {
				return NilValue{}, err
			}
			out.WriteString(val.String())
		}

		return StringValue{V: out.String()}, nil

	default:
		return ErrorValue{
			V: NewRuntimeError(expr, fmt.Sprintf("unhandled expression type: %T", e)),
		}, nil
	}
}

func (i *Interpreter) assignWithType(node parser.Node, v Value, expected *TypeInfo) (Value, error) {
	if expected == nil {
		return v, nil
	}

	expected = unwrapAlias(expected)
	actual := unwrapAlias(i.typeInfoFromValue(v))

	// special case: []thing absorbs array elem types
	if expected.Kind == TypeArray && expected.Elem.Kind == TypeAny {
		if arr, ok := v.(ArrayValue); ok {
			arr.ElemType = expected.Elem
			v = arr
			actual = expected
		}
	}

	if !typesAssignable(actual, expected) {
		return NilValue{}, NewRuntimeError(
			node,
			fmt.Sprintf(
				"type mismatch: expected '%s' but got '%s'",
				expected.Name,
				actual.Name,
			),
		)
	}

	v = promoteValueToType(v, expected)

	if expected.Kind == TypeAny {
		v = NamedValue{
			TypeName: expected,
			Value:    v,
		}
	}

	return v, nil
}

func (i *Interpreter) evalExprWithExpectedType(expr parser.Expression, expected *TypeInfo) (Value, error) {
	if expected != nil {
		expected = unwrapAlias(expected)
	}

	switch e := expr.(type) {

	case *parser.ArrayLiteral:
		if expected != nil && expected.Kind == TypeArray {
			return i.evalArrayLiteral(e, expected.Elem)
		}
		return i.EvalExpression(e)

	case *parser.FuncCall:
		if expected != nil &&
			expected.Kind == TypeArray &&
			expected.Elem.Kind == TypeAny &&
			e.Callee.(*parser.Identifier).Value == "toArr" {

			return i.evalToArrWithExpectedElem(e, expected.Elem)
		}
		return i.EvalExpression(e)

	case *parser.MapLiteral:
		if expected != nil && expected.Kind == TypeMap {
			return i.evalMapLiteral(e, expected)
		}
		return i.evalMapLiteral(e, nil)

	default:
		return i.EvalExpression(expr)
	}
}

func (i *Interpreter) evalArrayLiteral(expr *parser.ArrayLiteral, expected *TypeInfo) (Value, error) {
	elements := []Value{}
	var elemType *TypeInfo

	if expected != nil {
		elemType = unwrapAlias(expected)
	}

	for idx, el := range expr.Elements {
		val, err := i.EvalExpression(el)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := val.(ErrorValue); ok {
			return v, nil
		}

		valType := unwrapAlias(i.typeInfoFromValue(val))

		if elemType == nil {
			elemType = valType
		} else {
			if !typesAssignable(valType, elemType) {
				return ErrorValue{
					V: NewRuntimeError(
						expr,
						fmt.Sprintf(
							"array element %d expected %s but got %s (use []thing for mixed arrays)",
							idx,
							elemType.Name,
							valType.Name,
						),
					),
				}, nil
			}
		}

		val = promoteValueToType(val, elemType)

		elements = append(elements, val)
	}

	if elemType == nil {
		return ErrorValue{
			V: NewRuntimeError(expr, "cannot infer type of empty array"),
		}, nil
	}

	return ArrayValue{
		Elements: elements,
		ElemType: elemType,
	}, nil
}

func (i *Interpreter) evalToArrWithExpectedElem(
	node *parser.FuncCall,
	expectedElem *TypeInfo,
) (Value, error) {

	args := make([]Value, len(node.Args))
	for idx, arg := range node.Args {
		v, err := i.EvalExpression(arg)
		if err != nil {
			return NilValue{}, err
		}
		args[idx] = v
	}

	elements := make([]Value, 0, len(args))

	for idx, v := range args {
		t := unwrapAlias(i.typeInfoFromValue(v))

		if !typesAssignable(t, expectedElem) {
			return ErrorValue{
				V: NewRuntimeError(
					node,
					fmt.Sprintf(
						"toArr argument %d expected %s but got %s",
						idx,
						expectedElem.Name,
						t.Name,
					),
				),
			}, nil
		}

		elements = append(elements, promoteValueToType(v, expectedElem))
	}

	return ArrayValue{
		Elements: elements,
		ElemType: expectedElem,
	}, nil
}

func (i *Interpreter) evalMapLiteral(expr *parser.MapLiteral, expected *TypeInfo) (Value, error) {
	if len(expr.Pairs) == 0 {
		if expected == nil || expected.Kind != TypeMap {
			return ErrorValue{
				V: NewRuntimeError(expr, "cannot infer type of empty map"),
			}, nil
		}

		return MapValue{
			Entries:   map[Value]Value{},
			KeyType:   expected.Key,
			ValueType: expected.Value,
		}, nil
	}

	k0, err := i.EvalExpression(expr.Pairs[0].Key)
	if err != nil {
		return NilValue{}, err
	}
	if v, ok := k0.(ErrorValue); ok {
		return v, nil
	}

	v0, err := i.EvalExpression(expr.Pairs[0].Value)
	if err != nil {
		return NilValue{}, err
	}
	if v, ok := v0.(ErrorValue); ok {
		return v, nil
	}

	keyTI := unwrapAlias(i.typeInfoFromValue(k0))
	valTI := unwrapAlias(i.typeInfoFromValue(v0))

	if expected != nil {
		if !isComparableValue(k0) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("map key type %s is not comparable", keyTI.Name)),
			}, nil
		}

		if !typesAssignable(keyTI, expected.Key) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("type mismatch: map key 0 expected %s but got %s", expected.Key.Name, keyTI.Name)),
			}, nil
		}
		keyTI = expected.Key

		if !typesAssignable(valTI, expected.Value) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("type mismatch: map value 0 expected %s but got %s", expected.Value.Name, valTI.Name)),
			}, nil
		}
		valTI = expected.Value
	}

	elems := map[Value]Value{}

	for idx, e := range expr.Pairs {
		k, err := i.EvalExpression(e.Key)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := k.(ErrorValue); ok {
			return v, nil
		}

		v, err := i.EvalExpression(e.Value)
		if err != nil {
			return NilValue{}, err
		}

		kt := unwrapAlias(i.typeInfoFromValue(k))
		vt := unwrapAlias(i.typeInfoFromValue(v))

		if keyTI.Kind == TypeAny && !isComparableValue(k) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("map key %d is not comparable", idx)),
			}, nil
		}

		if !typesAssignable(kt, keyTI) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("map key %d expected %s but got %s", idx, keyTI.Name, kt.Name)),
			}, nil
		}

		if !typesAssignable(vt, valTI) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("map value %d expected %s but got %s", idx, valTI.Name, vt.Name)),
			}, nil
		}

		elems[k] = v
	}

	return MapValue{
		Entries:   elems,
		KeyType:   keyTI,
		ValueType: valTI,
	}, nil
}

func (i *Interpreter) evalCall(e *parser.FuncCall) (Value, error) {
	if ident, ok := e.Callee.(*parser.Identifier); ok {
		if ti, ok := i.typeEnv[ident.Value]; ok {
			if len(e.Args) != 1 {
				return ErrorValue{
					V: NewRuntimeError(e, "type cast expects 1 arg"),
				}, nil
			}
			return i.evalTypeCast(ti, e.Args[0], e)
		}
	}

	return i.evalFuncCall(e)
}

func (i *Interpreter) evalTypeCast(target *TypeInfo, arg parser.Expression, node parser.Node) (Value, error) {
	v, err := i.EvalExpression(arg)
	if err != nil {
		return NilValue{}, err
	}

	v = unwrapNamed(v)

	switch target.Kind {
	case TypeInt:
		var val int

		switch v := v.(type) {
		case IntValue:
			val = v.V
		case FloatValue:
			val = int(v.V)
		default:
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("int type cast does not support %s, try the function toInt to parse non-numeric types", string(v.Type()))),
			}, nil
		}

		return IntValue{V: val}, nil
	case TypeFloat:
		var val float64

		switch v := v.(type) {
		case IntValue:
			val = float64(v.V)
		case FloatValue:
			val = v.V
		default:
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("float type cast does not support %s, try the function toFloat to parse non-numeric types", string(v.Type()))),
			}, nil
		}

		return FloatValue{V: val}, nil
	case TypeString:
		if s, ok := v.(StringValue); ok {
			return s, nil
		}

		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("string cast does not support %s, try the function toString to parse other types", string(v.Type()))),
		}, nil
	case TypeBool:
		var val bool

		switch v := v.(type) {
		case BoolValue:
			val = v.V
		default:
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("bool type cast does not support %s, try the function toBool to parse other types", string(v.Type()))),
			}, nil
		}

		return BoolValue{V: val}, nil

	case TypeArray:
		if a, ok := v.(ArrayValue); ok {
			return a, nil
		}

		return ErrorValue{
			NewRuntimeError(node, fmt.Sprintf("array cast does not support %s, try the function toArr to construct arrays", string(v.Type()))),
		}, nil
	case TypeNamed:
		base := target.Underlying

		casted, err := i.evalTypeCast(base, arg, node)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := casted.(ErrorValue); ok {
			return v, nil
		}

		if sv, ok := casted.(*StructValue); ok {
			sv.TypeName = target
			return sv, nil
		}

		return NamedValue{
			TypeName: target,
			Value:    casted,
		}, nil
	case TypeError:
		s, ok := v.(StringValue)
		if ok {
			return ErrorValue{V: errors.New(s.V)}, nil
		}

		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("error cast does not support %s", string(v.Type()))),
		}, nil
	default:
		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("unknown type cast: %s", target.Name)),
		}, nil
	}
}

func (i *Interpreter) evalArgs(exprs []parser.Expression) ([]Value, error) {
	args := []Value{}
	for _, a := range exprs {
		v, err := i.EvalExpression(a)
		if err != nil {
			return nil, err
		}
		if t, ok := v.(TupleValue); ok {
			args = append(args, t.Values...)
		} else {
			args = append(args, v)
		}
	}
	return args, nil
}

func (i *Interpreter) evalFuncCall(expr *parser.FuncCall) (Value, error) {
	// builtin
	if ident, ok := expr.Callee.(*parser.Identifier); ok {
		if b, ok := i.env.builtins[ident.Value]; ok {
			args, err := i.evalArgs(expr.Args)
			if err != nil {
				return NilValue{}, err
			}
			if b.Arity >= 0 && len(args) != b.Arity {
				return NilValue{}, NewRuntimeError(expr,
					fmt.Sprintf("expected %d args, got %d", b.Arity, len(args)))
			}
			return b.Fn(i, expr, args)
		}
	}

	// user-defined
	val, err := i.EvalExpression(expr.Callee)
	if err != nil {
		return NilValue{}, err
	}
	if v, ok := val.(ErrorValue); ok {
		return v, nil
	}

	fn, ok := val.(*Func)
	if !ok {
		return ErrorValue{
			V: NewRuntimeError(expr, fmt.Sprintf("expected 'function' but got '%s'", unwrapAlias(i.typeInfoFromValue(val)).Name)),
		}, nil
	}

	args, err := i.evalArgs(expr.Args)
	if err != nil {
		return NilValue{}, err
	}

	return i.callFunction(fn, args, expr)
}

func (i *Interpreter) evalMethodCall(expr *parser.MethodCall) (Value, error) {
	recv, err := i.EvalExpression(expr.Receiver)
	if err != nil {
		return NilValue{}, err
	}

	recvType := unwrapAlias(i.typeInfoFromValue(recv))

	fn, ok := i.env.GetMethod(recvType, expr.Name.Value)
	if !ok {
		return ErrorValue{
			V: NewRuntimeError(expr,
				fmt.Sprintf("type %s has no method %s",
					recvType.Name, expr.Name.Value)),
		}, nil
	}

	args, err := i.evalArgs(expr.Args)
	if err != nil {
		return NilValue{}, err
	}

	// inject receiver as first argument
	args = append([]Value{recv}, args...)

	return i.callFunction(fn, args, expr)
}

func (i *Interpreter) callFunction(fn *Func, args []Value, callNode parser.Node) (Value, error) {
	if len(fn.Params) != len(args) {
		return ErrorValue{
			V: NewRuntimeError(callNode,
				fmt.Sprintf("expected %d args, got %d",
					len(fn.Params), len(args))),
		}, nil
	}

	// new env
	newEnv := NewEnvironment(fn.Env)

	// bind params
	for idx, param := range fn.Params {
		val := args[idx]

		if param.Type != nil {
			expected, err := i.resolveTypeNode(param.Type)
			if err != nil {
				return NilValue{}, err
			}

			actual := unwrapAlias(i.typeInfoFromValue(val))

			val, err = i.assignWithType(callNode, val, expected)
			if err != nil {
				return NilValue{}, NewRuntimeError(callNode, fmt.Sprintf("param '%s' expected '%s' but got '%s'", param.Name.Value, expected.Name, actual.Name))
			}
		}

		newEnv.Define(param.Name.Value, val)
	}

	// execute
	prevEnv := i.env
	i.env = newEnv

	sig, err := i.EvalBlock(fn.Body, false)

	deferErr := i.runDefers(newEnv)

	i.env = prevEnv

	if err != nil {
		return NilValue{}, err
	}
	if deferErr != nil {
		return NilValue{}, deferErr
	}

	// handle return
	if ret, ok := sig.(SignalReturn); ok {
		if len(fn.TypeName.Returns) > 0 && len(fn.TypeName.Returns) != len(ret.Values) {
			return ErrorValue{
				V: NewRuntimeError(callNode,
					fmt.Sprintf("expected %d return values, got %d",
						len(fn.TypeName.Returns), len(ret.Values))),
			}, nil
		}

		for idx, expectedType := range fn.TypeName.Returns {
			actual := ret.Values[idx]

			if err != nil {
				return ErrorValue{
					V: NewRuntimeError(callNode, err.Error()),
				}, nil
			}
			expectedTI := unwrapAlias(expectedType)

			if expectedTI.Name == "error" {
				if _, isNil := actual.(NilValue); isNil {
					continue
				}
			}

			if unwrapAlias(i.typeInfoFromValue(actual)).Kind == TypeError {
				return actual, nil
			}

			actual, err = i.assignWithType(callNode, actual, expectedTI)
			if err != nil {
				return NilValue{}, err
			}

			ret.Values[idx] = actual
		}

		if len(fn.TypeName.Returns) > 1 {
			return TupleValue{Values: ret.Values}, nil
		}
		return ret.Values[0], nil
	}

	return NilValue{}, nil
}

func (i *Interpreter) evalIndexExpression(node parser.Expression, left, idx Value) (Value, error) {
	if nv, ok := left.(NamedValue); ok && nv.TypeName.Kind == TypeAny {
		return ErrorValue{
			V: NewRuntimeError(node, "cannot index value of type 'thing' without type assertion"),
		}, nil
	}

	left = unwrapNamed(left)

	typ := i.typeInfoFromValue(left)

	switch typ.Kind {
	case TypeArray:
		arr, ok := left.(*ArrayValue)
		if !ok {
			v := left.(ArrayValue)
			arr = &v
		}

		idxVal, ok := idx.(IntValue)
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("array index: %v, must be int", idxVal.V)),
			}, nil
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(arr.Elements) {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("array index: %d, out of bounds", idx)),
			}, nil
		}

		elem := arr.Elements[idx]

		if arr.ElemType.Kind == TypeAny {
			elem = NamedValue{
				TypeName: arr.ElemType,
				Value:    elem,
			}
		}

		return elem, nil

	case TypeString:
		idxVal, ok := idx.(IntValue)
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("string index: %v, must be int", idxVal.V)),
			}, nil
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(left.(StringValue).V) {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("string index: %d, out of bounds", idx)),
			}, nil
		}

		r := []rune(left.(StringValue).V)
		return StringValue{V: string(r[idx])}, nil

	case TypeMap:
		mv := left.(MapValue)

		// 1. type check key
		keyType := unwrapAlias(i.typeInfoFromValue(idx))

		if mv.KeyType.Kind == TypeAny {
			if !isComparableValue(idx) {
				return ErrorValue{
					V: NewRuntimeError(
						node,
						"value of this type cannot be used as map key",
					),
				}, nil
			}
		} else {
			if !typesAssignable(keyType, mv.KeyType) {
				return ErrorValue{
					V: NewRuntimeError(
						node,
						fmt.Sprintf(
							"map index expected %s but got %s",
							mv.KeyType.Name,
							keyType.Name,
						),
					),
				}, nil
			}
		}

		val, ok := mv.Entries[idx]
		if !ok {
			return NilValue{}, nil
		}

		if mv.ValueType.Kind == TypeAny {
			return NamedValue{
				TypeName: mv.ValueType,
				Value:    val,
			}, nil
		}

		return val, nil

	default:
		var typeStr string
		typeInt := 0

		switch int(typ.Kind) {
		case 0:
			typeStr = "int"
		case 1:
			typeStr = "float"
		case 3:
			typeStr = "bool"
		case 5:
			typeStr = "nil"
		case 6:
			typeStr = "struct"
		case 8:
			typeStr = "thing"
		default:
			typeStr = ""
			typeInt = int(typ.Kind)
		}

		if typeStr != "" {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("indexing is not allowed with type: '%s'", typeStr)),
			}, nil
		}

		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("indexing is not allowed with type: %d", typeInt)),
		}, nil
	}
}

func evalMemberExpression(node parser.Expression, left Value, field string) (Value, error) {
	switch obj := left.(type) {
	case *StructValue:

		val, ok := obj.Fields[field]
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("unknown field %s", field)),
			}, nil
		}

		expectedType := obj.TypeName.Fields[field]

		if val.Type() != valueTypeOf(expectedType) {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("field '%s' type '%v' should be '%v'", field, val.Type(), expectedType)),
			}, nil
		}

		return val, nil
	case TypeValue:
		if obj.TypeName.Kind != TypeEnum {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("type '%s' has no members", obj.TypeName.Name)),
			}, nil
		}

		idx, ok := obj.TypeName.Variants[field]
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("unknown enum variant '%s.%s'", obj.TypeName.Name, field)),
			}, nil
		}

		return EnumValue{
			Enum:    obj.TypeName,
			Variant: field,
			Index:   idx,
		}, nil
	}

	return ErrorValue{
		V: NewRuntimeError(node, fmt.Sprintf("member expression expects enums or structs, but got '%s'", string(left.Type()))),
	}, nil
}

func (i *Interpreter) evalInfix(node *parser.InfixExpression, left Value, op string, right Value) (Value, error) {
	if _, ok := left.(ErrorValue); ok {
		return left, nil
	}
	if _, ok := right.(ErrorValue); ok {
		return right, nil
	}

	if left.Type() == ERROR {
		return evalErrorInfix(node, left.(ErrorValue), op, right)
	}
	if right.Type() == ERROR {
		return evalErrorInfix(node, right.(ErrorValue), op, left)
	}

	// nil handling
	if _, ok := left.(NilValue); ok {
		return evalNilInfix(node, op, right)
	}
	if _, ok := right.(NilValue); ok {
		return evalNilInfix(node, op, left)
	}

	// strict any handling
	leftTI := unwrapAlias(i.typeInfoFromValue(left))
	rightTI := unwrapAlias(i.typeInfoFromValue(right))

	if leftTI.Kind == TypeAny || rightTI.Kind == TypeAny {
		return ErrorValue{
			V: NewRuntimeError(
				node,
				"cannot use 'thing' in operations, assert a type first",
			),
		}, nil
	}

	// named values
	lnv, lok := left.(NamedValue)
	rnv, rok := right.(NamedValue)

	if lok || rok {
		if !lok || !rok || lnv.TypeName != rnv.TypeName {
			return ErrorValue{
				V: NewRuntimeError(
					node,
					"cannot operate on mismatched named types (try casting)",
				),
			}, nil
		}

		// Same named type → unwrap
		ul := unwrapNamed(left)
		ur := unwrapNamed(right)

		res, err := i.evalInfix(node, ul, op, ur)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := res.(ErrorValue); ok {
			return v, nil
		}

		// Re-wrap result
		return NamedValue{
			TypeName: lnv.TypeName,
			Value:    res,
		}, nil
	}

	if left.Type() == INT && right.Type() == FLOAT {
		return evalFloatInfix(node, FloatValue{V: float64(left.(IntValue).V)}, op, right.(FloatValue))
	}

	if left.Type() == FLOAT && right.Type() == INT {
		return evalFloatInfix(node, left.(FloatValue), op, FloatValue{V: float64(right.(IntValue).V)})
	}

	// type mismatch check
	if left.Type() != right.Type() {
		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("type mismatch: '%s' %s '%s'", left.Type(), op, right.Type())),
		}, nil
	}

	switch left.Type() {
	case INT:
		return evalIntInfix(node, left.(IntValue), op, right.(IntValue))
	case FLOAT:
		return evalFloatInfix(node, left.(FloatValue), op, right.(FloatValue))
	case STRING:
		return evalStringInfix(node, left.(StringValue), op, right.(StringValue))
	case BOOL:
		return evalBoolInfix(node, left.(BoolValue), op, right.(BoolValue))
	}

	return ErrorValue{
		V: NewRuntimeError(node, "unsupported operand types"),
	}, nil
}

func evalIntInfix(node *parser.InfixExpression, left IntValue, op string, right IntValue) (Value, error) {
	switch op {
	case "+":
		return IntValue{V: left.V + right.V}, nil
	case "-":
		return IntValue{V: left.V - right.V}, nil
	case "*":
		return IntValue{V: left.V * right.V}, nil
	case "/":
		if right.V == 0 {
			return ErrorValue{
				V: NewRuntimeError(node, "undefined: division by zero"),
			}, nil
		}

		return FloatValue{V: float64(left.V) / float64(right.V)}, nil

	case "%":
		if right.V == 0 {
			return ErrorValue{
				V: NewRuntimeError(node, "undefined: mod by zero"),
			}, nil
		}

		return IntValue{V: left.V % right.V}, nil
	case "==":
		return BoolValue{V: left.V == right.V}, nil
	case "!=":
		return BoolValue{V: left.V != right.V}, nil
	case ">":
		return BoolValue{V: left.V > right.V}, nil
	case "<":
		return BoolValue{V: left.V < right.V}, nil
	case ">=":
		return BoolValue{V: left.V >= right.V}, nil
	case "<=":
		return BoolValue{V: left.V <= right.V}, nil
	}

	return ErrorValue{
		V: NewRuntimeError(node, fmt.Sprintf("invalid operator %d %s %d", left.V, op, right.V)),
	}, nil
}

func evalFloatInfix(node *parser.InfixExpression, left FloatValue, op string, right FloatValue) (Value, error) {
	switch op {
	case "+":
		return FloatValue{V: left.V + right.V}, nil
	case "-":
		return FloatValue{V: left.V - right.V}, nil
	case "*":
		return FloatValue{V: left.V * right.V}, nil
	case "/":
		if right.V == 0 {
			return ErrorValue{
				V: NewRuntimeError(node, "undefined: division by zero"),
			}, nil
		}

		return FloatValue{V: left.V / right.V}, nil
	case "==":
		return BoolValue{V: left.V == right.V}, nil
	case "!=":
		return BoolValue{V: left.V != right.V}, nil
	case ">":
		return BoolValue{V: left.V > right.V}, nil
	case "<":
		return BoolValue{V: left.V < right.V}, nil
	case ">=":
		return BoolValue{V: left.V >= right.V}, nil
	}

	return ErrorValue{
		V: NewRuntimeError(node, fmt.Sprintf("invalid operator %f %s %f", left.V, op, right.V)),
	}, nil
}

func evalStringInfix(node *parser.InfixExpression, left StringValue, op string, right StringValue) (Value, error) {
	switch op {
	case "+":
		return StringValue{V: left.V + right.V}, nil
	case "==":
		return BoolValue{V: left.V == right.V}, nil
	case "!=":
		return BoolValue{V: left.V != right.V}, nil
	}

	return ErrorValue{
		V: NewRuntimeError(node, fmt.Sprintf("invalid operator %s %s %s", left.V, op, right.V)),
	}, nil
}

func evalBoolInfix(node *parser.InfixExpression, left BoolValue, op string, right BoolValue) (Value, error) {
	switch op {
	case "==":
		return BoolValue{V: left.V == right.V}, nil
	case "!=":
		return BoolValue{V: left.V != right.V}, nil
	}

	return ErrorValue{
		V: NewRuntimeError(node, fmt.Sprintf("invalid operator %t %s %t", left.V, op, right.V)),
	}, nil
}

func evalNilInfix(node *parser.InfixExpression, op string, other Value) (Value, error) {
	switch op {
	case "==":
		_, isNil := other.(NilValue)
		return BoolValue{V: isNil}, nil
	case "!=":
		_, isNil := other.(NilValue)
		return BoolValue{V: !isNil}, nil
	default:
		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("invalid operator nil %s %s", op, other.String())),
		}, nil
	}
}

func evalErrorInfix(node *parser.InfixExpression, left ErrorValue, op string, right Value) (Value, error) {
	re, ok := right.(ErrorValue)
	if !ok {
		switch op {
		case "==":
			return BoolValue{V: false}, nil
		case "!=":
			return BoolValue{V: true}, nil
		}
	}

	switch op {
	case "==":
		return BoolValue{V: left.V.Error() == re.V.Error()}, nil
	case "!=":
		return BoolValue{V: left.V.Error() != re.V.Error()}, nil
	default:
		return ErrorValue{
			V: NewRuntimeError(node, "invalid operator for error"),
		}, nil
	}
}

func evalPrefix(node *parser.PrefixExpression, operator string, right Value) (Value, error) {
	switch operator {
	case "!":
		rTruthy, err := isTruthy(right)
		if err != nil {
			return ErrorValue{
				V: NewRuntimeError(node, err.Error()),
			}, nil
		}

		return BoolValue{V: !rTruthy}, nil
	case "-":
		switch v := right.(type) {
		case IntValue:
			return IntValue{V: -v.V}, nil
		case FloatValue:
			return FloatValue{V: -v.V}, nil
		default:
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("invalid operand, %s, for unary '-'", right.String())),
			}, nil
		}
	default:
		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("unknown prefix operator: %s", operator)),
		}, nil
	}
}

func isTruthy(val Value) (bool, error) {
	b, ok := val.(BoolValue)
	if !ok {
		return false, fmt.Errorf("condition must be boolean")
	}
	return b.V, nil
}
