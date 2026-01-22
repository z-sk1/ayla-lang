package interpreter

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"os"
	"sync"

	"github.com/z-sk1/ayla-lang/parser"
	"golang.org/x/term"
)

type Func struct {
	Params      []*parser.ParametersClause
	Body        []parser.Statement
	ReturnTypes []*parser.Identifier
	Env         *Environment
}

type BuiltinFunc struct {
	Name  string
	Arity int
	Fn    func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error)
}

type RuntimeError struct {
	Message string
	Line    int
	Column  int
}

type Environment struct {
	store    map[string]Value
	funcs    map[string]*Func
	builtins map[string]*BuiltinFunc
	mu       sync.RWMutex
	parent   *Environment
}

type Interpreter struct {
	env     *Environment
	typeEnv map[string]*TypeInfo
}

func New() *Interpreter {
	env := &Environment{
		store:    make(map[string]Value),
		funcs:    make(map[string]*Func),
		builtins: make(map[string]*BuiltinFunc),
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
		store:    make(map[string]Value),
		funcs:    make(map[string]*Func),
		builtins: parent.builtins,
		parent:   parent,
	}
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("runtime error at %d:%d: %s", e.Line, e.Column, e.Message)
}

func NewRuntimeError(node parser.Node, msg string) RuntimeError {
	line, col := node.Pos()
	return RuntimeError{Message: msg, Line: line, Column: col}
}

func (e *Environment) Get(name string) (Value, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if v, ok := e.store[name]; ok {
		return v, true
	}

	if e.parent != nil {
		return e.parent.Get(name)
	}

	return nil, false
}

func (e *Environment) Define(name string, val Value) Value {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.store[name] = val
	return val
}

func (e *Environment) Set(name string, val Value) Value {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.store[name]; ok {
		e.store[name] = val
		return val
	}

	if e.parent != nil {
		return e.parent.Set(name, val)
	}

	return nil
}

func (e *Environment) GetFunc(name string) (*Func, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if v, ok := e.funcs[name]; ok {
		return v, true
	}

	if e.parent != nil {
		return e.parent.GetFunc(name)
	}

	return nil, false
}

func (e *Environment) SetFunc(name string, f *Func) *Func {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.funcs[name] = f
	return f
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

func assignable(expected *TypeInfo, v Value) bool {
	// Named value → must match exactly
	if nv, ok := v.(NamedValue); ok {
		return nv.TypeName == expected
	}

	if sv, ok := v.(*StructValue); ok {
		return sv.TypeName == expected
	}

	if valueTypeOf(expected) == FLOAT && v.Type() == INT {
		return true
	}

	// Plain value → must match underlying kind
	return valueTypeOf(expected) == v.Type()
}

func typesAssignable(from, to *TypeInfo) bool {
	if from == to {
		return true
	}

	// aliases are transparent
	if from.Alias {
		return typesAssignable(from.Underlying, to)
	}
	if to.Alias {
		return typesAssignable(from, to.Underlying)
	}

	// named types must match exactly
	if from.Kind == TypeNamed || to.Kind == TypeNamed {
		return from == to
	}

	if from.Kind == TypeInt && to.Kind == TypeFloat {
		return true
	}

	return false
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
		Name: "int",
		Kind: TypeInt,
	}

	typeEnv["float"] = &TypeInfo{
		Name: "float",
		Kind: TypeFloat,
	}

	typeEnv["string"] = &TypeInfo{
		Name: "string",
		Kind: TypeString,
	}

	typeEnv["bool"] = &TypeInfo{
		Name: "bool",
		Kind: TypeBool,
	}

	typeEnv["arr"] = &TypeInfo{
		Name: "arr",
		Kind: TypeArray,
	}

	typeEnv["nil"] = &TypeInfo{
		Name: "nil",
		Kind: TypeNil,
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
					return NilValue{}, NewRuntimeError(node, "could not convert string to int")
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
			return ArrayValue{Elements: args}, nil
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

	env.builtins["type"] = &BuiltinFunc{
		Name:  "type",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]
			return StringValue{V: string(v.Type())}, nil
		},
	}

	env.builtins["explode"] = &BuiltinFunc{
		Name:  "explode",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			for _, v := range args {
				if v.Type() != NIL {
					fmt.Print(v.String())
				}
			}
			return NilValue{}, nil
		},
	}

	env.builtins["explodeln"] = &BuiltinFunc{
		Name:  "explodeln",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			for _, v := range args {
				if v.Type() != NIL {
					fmt.Println(v.String())
				}
			}
			return NilValue{}, nil
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

func (i *Interpreter) EvalStatements(stmts []parser.Statement) (ControlSignal, error) {
	for _, s := range stmts {
		sig, err := i.EvalStatement(s)
		if err != nil {
			return SignalNone{}, err
		}

		switch sig.(type) {
		case SignalReturn, SignalBreak, SignalContinue:
			return sig, nil
		}
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

		switch {
		case stmt.Value != nil:
			// egg x = expr
			v, err := i.EvalExpression(stmt.Value)
			if err != nil {
				return SignalNone{}, err
			}
			val = v

		case stmt.Type != nil:
			// egg x int
			expectedTI, ok := i.typeInfoFromIdent(stmt.Type.Value)
			if !ok {
				return SignalNone{}, NewRuntimeError(
					stmt,
					"unknown type: "+stmt.Type.Value,
				)
			}

			val, err = defaultValueFromTypeInfo(stmt, expectedTI)

		default:
			// egg x
			val = NilValue{}
		}

		if err != nil {
			return SignalNone{}, err
		}

		if stmt.Type != nil {
			expectedTI, ok := i.typeInfoFromIdent(stmt.Type.Value)
			if !ok {
				return SignalNone{}, NewRuntimeError(
					stmt,
					"unknown type: "+stmt.Type.Value,
				)
			}

			actualTI := i.typeInfoFromValue(val)

			actualTI = unwrapAlias(actualTI)
			expectedTI = unwrapAlias(expectedTI)

			if !typesAssignable(actualTI, expectedTI) {
				return SignalNone{}, NewRuntimeError(
					stmt,
					fmt.Sprintf(
						"type mismatch: '%s' assigned to '%s'",
						actualTI.Name,
						expectedTI.Name,
					),
				)
			}
		}

		// variable must not exist
		if _, ok := i.env.store[stmt.Name]; ok {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("cant redeclare var: %s", stmt.Name))
		}

		i.env.Define(stmt.Name, val)
		return SignalNone{}, nil

	case *parser.MultiVarStatement:
		var values []Value

		if stmt.Value != nil {
			val, err := i.EvalExpression(stmt.Value)
			if err != nil {
				return SignalNone{}, err
			}

			tuple, ok := val.(TupleValue)
			if !ok {
				return SignalNone{}, NewRuntimeError(stmt, "multi var assignment expects tuple value")
			}

			if len(tuple.Values) != len(stmt.Names) {
				return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("expected %d values, got %d", len(stmt.Names), len(tuple.Values)))
			}

			values = tuple.Values
		}

		for idx, name := range stmt.Names {
			if _, ok := i.env.store[name]; ok {
				return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("cannot redeclare var: %s", name))
			}

			var v Value = NilValue{}

			switch {
			case stmt.Value != nil:
				v = values[idx]
			case stmt.Type != nil:
				var err error
				expectedTI, ok := i.typeInfoFromIdent(stmt.Type.Value)
				if !ok {
					return SignalNone{}, NewRuntimeError(
						stmt,
						"unknown type: "+stmt.Type.Value,
					)
				}

				v, err = defaultValueFromTypeInfo(stmt, expectedTI)
				if err != nil {
					return SignalNone{}, err
				}
			default:
				v = NilValue{}
			}

			if stmt.Type != nil {
				expectedTI, ok := i.typeInfoFromIdent(stmt.Type.Value)
				if !ok {
					return SignalNone{}, NewRuntimeError(
						stmt,
						"unknown type: "+stmt.Type.Value,
					)
				}

				actualTI := i.typeInfoFromValue(v)

				expectedTI = unwrapAlias(expectedTI)
				actualTI = unwrapAlias(actualTI)

				if !typesAssignable(actualTI, expectedTI) {
					return SignalNone{}, NewRuntimeError(
						stmt,
						fmt.Sprintf(
							"type mismatch: '%s' assigned to '%s'",
							actualTI.Name,
							expectedTI.Name,
						),
					)
				}
			}

			i.env.Define(name, v)
		}

		return SignalNone{}, nil

	case *parser.ConstStatement:
		var val Value
		var err error

		if stmt.Value == nil {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("const %s must be initalised with a value", stmt.Name))
		} else {
			val, err = i.EvalExpression(stmt.Value)
			if err != nil {
				return SignalNone{}, err
			}
		}

		if stmt.Type != nil {
			expectedTI, ok := i.typeInfoFromIdent(stmt.Type.Value)
			if !ok {
				return SignalNone{}, NewRuntimeError(
					stmt,
					"unknown type: "+stmt.Type.Value,
				)
			}

			actualTI := i.typeInfoFromValue(val)

			expectedTI = unwrapAlias(expectedTI)
			actualTI = unwrapAlias(actualTI)

			if !typesAssignable(actualTI, expectedTI) {
				return SignalNone{}, NewRuntimeError(
					stmt,
					fmt.Sprintf(
						"type mismatch: '%s' assigned to '%s'",
						actualTI.Name,
						expectedTI.Name,
					),
				)
			}
		}

		// check if variable already exist
		if _, ok := i.env.store[stmt.Name]; ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("cant redeclare const: %s", stmt.Name))
		}

		// store const val
		i.env.Define(stmt.Name, ConstValue{Value: val})
		return SignalNone{}, nil

	case *parser.MultiConstStatement:
		var values []Value

		if stmt.Value == nil {
			var names string

			for _, name := range stmt.Names {
				if name == stmt.Names[len(stmt.Names)-1] {
					names = names + name
				} else {
					names = names + (name + ", ")
				}
			}

			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("constants, %s, must be initialised", names))
		} else {
			var err error
			val, err := i.EvalExpression(stmt.Value)
			if err != nil {
				return SignalNone{}, err
			}

			tuple, ok := val.(TupleValue)
			if !ok {
				return SignalNone{}, NewRuntimeError(stmt, "multi const statement expects tuple value")
			}

			if len(tuple.Values) != len(stmt.Names) {
				return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("expected %d values, got %d", len(tuple.Values), len(stmt.Names)))
			}

			values = tuple.Values
		}

		for idx, name := range stmt.Names {
			if _, ok := i.env.store[name]; ok {
				return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("cannot redeclare const: %s", name))
			}

			var v Value = NilValue{}

			if stmt.Value != nil {
				v = values[idx]
			}

			if stmt.Type != nil {
				expectedTI, ok := i.typeInfoFromIdent(stmt.Type.Value)
				if !ok {
					return SignalNone{}, NewRuntimeError(
						stmt,
						"unknown type: "+stmt.Type.Value,
					)
				}

				actualTI := i.typeInfoFromValue(v)

				expectedTI = unwrapAlias(expectedTI)
				actualTI = unwrapAlias(actualTI)

				if !typesAssignable(actualTI, expectedTI) {
					return SignalNone{}, NewRuntimeError(
						stmt,
						fmt.Sprintf(
							"type mismatch: '%s' assigned to '%s'",
							actualTI.Name,
							expectedTI.Name,
						),
					)
				}
			}

			i.env.Define(name, v)
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
				i.typeEnv[stmt.Name] = &TypeInfo{
					Name:       stmt.Name,
					Kind:       underlying.Kind,
					Underlying: underlying,
					Alias:      true,
				}
			} else {
				i.typeEnv[stmt.Name] = &TypeInfo{
					Name:       stmt.Name,
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
				Name:   stmt.Name,
				Kind:   TypeStruct,
				Fields: fields,
			}

			if stmt.Alias {
				ti.Alias = true
			}

			i.typeEnv[stmt.Name] = ti
			return SignalNone{}, nil
		}

	case *parser.AssignmentStatement:
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}

		existingVal, ok := i.env.Get(stmt.Name)
		if !ok {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("assignment to undefined variable: %s", stmt.Name))
		}

		if _, isConst := existingVal.(ConstValue); isConst {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("cannot reassign to const: %s", stmt.Name))
		}

		expected := existingVal.Type()
		actual := val.Type()

		if expected != actual {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("type mismatch: '%s' assigned to a '%s'", string(actual), string(expected)))
		}

		i.env.Set(stmt.Name, val)
		return SignalNone{}, nil

	case *parser.MultiAssignmentStatement:
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}

		for idx, name := range stmt.Names {
			existingVal, ok := i.env.Get(name)
			if !ok {
				return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("assignment to undefined variable: %s", name))
			}

			if _, isConst := existingVal.(ConstValue); isConst {
				return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("cannot reassign to const: %s", name))
			}

			tuple, ok := val.(TupleValue)
			if !ok {
				return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("multi assign expects tuple value, got %s", string(val.Type())))
			}

			if len(tuple.Values) != len(stmt.Names) {
				return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("multi assign expected %d values, got %d", len(stmt.Names), len(tuple.Values)))
			}

			expected := existingVal.Type()
			actual := tuple.Values[idx].Type()

			if expected != actual {
				return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("type mismatch: '%s' assigned to '%s'", string(actual), string(expected)))
			}

			i.env.Set(name, tuple.Values[idx])
		}

		return SignalNone{}, nil

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

	case *parser.FuncStatement:
		i.env.SetFunc(stmt.Name, &Func{Params: stmt.Params, Body: stmt.Body, ReturnTypes: stmt.ReturnTypes, Env: i.env})
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

	case *parser.BreakStatement:
		return SignalBreak{}, nil

	case *parser.ContinueStatement:
		return SignalContinue{}, nil
	}

	return SignalNone{}, nil
}

func (i *Interpreter) EvalExpression(e parser.Expression) (Value, error) {
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

		return evalMemberExpression(expr, leftVal, expr.Field.Value)

	case *parser.Identifier:
		v, ok := i.env.Get(expr.Value)
		if !ok {
			return NilValue{}, NewRuntimeError(e, fmt.Sprintf("undefined variable: %s", expr.Value))
		}

		if c, isConst := v.(ConstValue); isConst {
			return c.Value, nil
		}

		return v, nil

	case *parser.ArrayLiteral:
		elements := []Value{}
		for _, el := range expr.Elements {
			val, err := i.EvalExpression(el)
			if err != nil {
				return NilValue{}, err
			}

			elements = append(elements, val)
		}
		return ArrayValue{Elements: elements}, nil

	case *parser.FuncCall:
		return i.evalCall(expr)

	case *parser.IndexExpression:
		left, err := i.EvalExpression(expr.Left)
		if err != nil {
			return NilValue{}, err
		}

		index, err := i.EvalExpression(expr.Index)
		if err != nil {
			return NilValue{}, err
		}

		val, err := evalIndexExpression(expr, left, index)
		if err != nil {
			return NilValue{}, err
		}

		return val, nil

	case *parser.StructLiteral:
		val, ok := i.typeEnv[expr.TypeName.Value]
		if !ok {
			return NilValue{}, NewRuntimeError(
				expr,
				"unknown struct type "+expr.TypeName.Value,
			)
		}

		structTI, err := unwrapStructType(val)
		if err != nil {
			return NilValue{}, NewRuntimeError(expr, err.Error())
		}

		if structTI.Kind != TypeStruct {
			return NilValue{}, NewRuntimeError(
				expr,
				structTI.Name+" is not a struct type",
			)
		}

		fields := make(map[string]Value)

		for name, e := range expr.Fields {
			expectedType, ok := structTI.Fields[name]
			if !ok {
				return NilValue{}, NewRuntimeError(
					expr,
					fmt.Sprintf(
						"unknown field '%s' in struct %s",
						name,
						structTI.Name,
					),
				)
			}

			v, err := i.EvalExpression(e)
			if err != nil {
				return NilValue{}, err
			}

			actualTI := unwrapAlias(i.typeInfoFromValue(v))
			expectedTI := unwrapAlias(expectedType)

			if !(typesAssignable(actualTI, expectedTI)) {
				return NilValue{}, NewRuntimeError(
					expr,
					fmt.Sprintf(
						"field '%s' expects %v but got %v",
						name,
						expectedType.Name,
						v.Type(),
					),
				)
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
					return NilValue{}, NewRuntimeError(
						expr,
						fmt.Sprintf(
							"field '%s' expects %v but got %v",
							name,
							expected.Name,
							string(v.Type()),
						),
					)
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

		return evalInfix(expr, left, expr.Operator, right)

	case *parser.PrefixExpression:
		right, err := i.EvalExpression(expr.Right)
		if err != nil {
			return NilValue{}, err
		}

		return evalPrefix(expr, expr.Operator, right)

	case *parser.GroupedExpression:
		return i.EvalExpression(expr.Expression)

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

		return &StringValue{V: out.String()}, nil

	default:
		return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("unhandled expression type: %T", e))
	}
}

func (i *Interpreter) evalCall(e *parser.FuncCall) (Value, error) {
	if ti, ok := i.typeEnv[e.Name]; ok {
		if len(e.Args) != 1 {
			return NilValue{}, NewRuntimeError(e, "type cast expects 1 arg")
		}
		return i.evalTypeCast(ti, e.Args[0], e)
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
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("int type cast does not support %s, try the function toInt to parse non-numeric types", string(v.Type())))
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
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("float type cast does not support %s, try the function toFloat to parse non-numeric types", string(v.Type())))
		}

		return FloatValue{V: val}, nil
	case TypeString:
		if s, ok := v.(StringValue); ok {
			return s, nil
		}

		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("string cast does not support %s, try the function toString to parse other types", string(v.Type())))
	case TypeBool:
		var val bool

		switch v := v.(type) {
		case BoolValue:
			val = v.V
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("bool type cast does not support %s, try the function toBool to parse other types", string(v.Type())))
		}

		return BoolValue{V: val}, nil

	case TypeArray:
		if a, ok := v.(ArrayValue); ok {
			return a, nil
		}

		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("array cast does not support %s, try the function toArr to construct arrays", string(v.Type())))
	case TypeNamed:
		base := target.Underlying

		casted, err := i.evalTypeCast(base, arg, node)
		if err != nil {
			return NilValue{}, err
		}

		if sv, ok := casted.(*StructValue); ok {
			sv.TypeName = target
			return sv, nil
		}

		return NamedValue{
			TypeName: target,
			Value:    casted,
		}, nil
	default:
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown type cast: %s", target.Name))
	}
}

func (i *Interpreter) evalFuncCall(expr *parser.FuncCall) (Value, error) {
	// built in?
	if b, ok := i.env.builtins[expr.Name]; ok {
		args := []Value{}

		for _, a := range expr.Args {
			v, err := i.EvalExpression(a)
			if err != nil {
				return NilValue{}, err
			}

			if t, ok := v.(TupleValue); ok {
				args = append(args, t.Values...) // flatten
			} else {
				args = append(args, v)
			}
		}

		if b.Arity >= 0 && len(args) != b.Arity {
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("expected %d args, got %d", b.Arity, len(args)))
		}

		return b.Fn(i, expr, args)
	}

	// user-defined
	fn, ok := i.env.GetFunc(expr.Name)
	if !ok {
		return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("unknown function: %s", expr.Name))
	}

	if len(fn.Params) != len(expr.Args) {
		return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("expected %d args, got %d", len(fn.Params), len(expr.Args)))
	}

	// create new env for func call
	newEnv := NewEnvironment(fn.Env)

	// set params
	for idx, param := range fn.Params {
		val, err := i.EvalExpression(expr.Args[idx])
		if err != nil {
			return NilValue{}, err
		}

		// enforce type if parameter has one
		if param.Type != nil {
			actual := string(val.Type())
			expected := param.Type.Value

			if expected == "float" && val.Type() == INT {
				val = FloatValue{V: float64(val.(IntValue).V)}
			} else if actual != expected {
				return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("parameter '%s' expected %v, got %v", param.Value, expected, actual))
			}
		}

		newEnv.Define(param.Value, val)
	}

	// execute body
	prevEnv := i.env
	i.env = newEnv

	sig, err := i.EvalBlock(fn.Body, false)

	i.env = prevEnv

	if err != nil {
		return NilValue{}, err
	}

	if ret, ok := sig.(SignalReturn); ok {
		if len(fn.ReturnTypes) > 0 {
			if len(fn.ReturnTypes) != len(ret.Values) {
				return NilValue{}, NewRuntimeError(expr,
					fmt.Sprintf("expected %d return values, got %d",
						len(fn.ReturnTypes), len(ret.Values)))
			}

			for i, expectedType := range fn.ReturnTypes {
				actual := ret.Values[i]

				if expectedType != nil && string(actual.Type()) != expectedType.Value {
					return NilValue{}, NewRuntimeError(
						expr,
						fmt.Sprintf(
							"return %d, expected %s, got %s",
							i+1,
							expectedType.Value,
							string(actual.Type()),
						),
					)
				}
			}
		}

		for i, expectedType := range fn.ReturnTypes {
			actual := ret.Values[i]

			if expectedType != nil && string(actual.Type()) != expectedType.Value {
				return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("return %d, expected %s, got %s", i+1, expectedType.Value, string(actual.Type())))
			}
		}

		if len(ret.Values) == 1 {
			return ret.Values[0], nil
		}

		return TupleValue{Values: ret.Values}, nil
	}

	return NilValue{}, nil
}

func evalIndexExpression(node parser.Expression, left, idx Value) (Value, error) {
	switch left := left.(type) {
	case ArrayValue:
		idxVal, ok := idx.(IntValue)
		if !ok {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("array index: %v, must be int", idxVal.V))
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(left.Elements) {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("array index: %d, out of bounds", idx))
		}

		return left.Elements[idx], nil
	case StringValue:
		idxVal, ok := idx.(IntValue)
		if !ok {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("string index: %v, must be int", idxVal.V))
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(left.V) {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("string index: %d, out of bounds", idx))
		}

		r := []rune(left.V)
		return &StringValue{V: string(r[idx])}, nil

	default:
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("indexing is not allowed with type: %T", left.Type()))
	}
}

func evalMemberExpression(node parser.Expression, left Value, field string) (Value, error) {
	obj, ok := left.(*StructValue)
	if !ok {
		return NilValue{}, NewRuntimeError(node, "not a struct")
	}

	val, ok := obj.Fields[field]
	if !ok {
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown field %s", field))
	}

	expectedType := obj.TypeName.Fields[field]

	if val.Type() != valueTypeOf(expectedType) {
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("field '%s' type %v should be %v", field, val.Type(), expectedType))
	}

	return val, nil
}

func evalInfix(node *parser.InfixExpression, left Value, op string, right Value) (Value, error) {
	// nil handling first
	if _, ok := left.(NilValue); ok {
		return evalNilInfix(node, op, right)
	}
	if _, ok := right.(NilValue); ok {
		return evalNilInfix(node, op, left)
	}

	// named values
	lnv, lok := left.(NamedValue)
	rnv, rok := right.(NamedValue)

	if lok || rok {
		if !lok || !rok || lnv.TypeName != rnv.TypeName {
			return NilValue{}, NewRuntimeError(
				node,
				"cannot operate on mismatched named types (try casting)",
			)
		}

		// Same named type → unwrap
		ul := unwrapNamed(left)
		ur := unwrapNamed(right)

		res, err := evalInfix(node, ul, op, ur)
		if err != nil {
			return NilValue{}, err
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
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("type mismatch: %s %s %s", left.Type(), op, right.Type()))
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

	return NilValue{}, NewRuntimeError(node, "unsupported operand types")
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
			return NilValue{}, NewRuntimeError(node, "undefined: division by zero")
		}

		return IntValue{V: left.V / right.V}, nil
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

	return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operator %d %s %d", left.V, op, right.V))
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
			return NilValue{}, NewRuntimeError(node, "undefined: division by zero")
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

	return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operator %f %s %f", left.V, op, right.V))
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

	return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operator %s %s %s", left.V, op, right.V))
}

func evalBoolInfix(node *parser.InfixExpression, left BoolValue, op string, right BoolValue) (Value, error) {
	switch op {
	case "==":
		return BoolValue{V: left.V == right.V}, nil
	case "!=":
		return BoolValue{V: left.V != right.V}, nil
	}

	return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operator %t %s %t", left.V, op, right.V))
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
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operator nil %s %s", op, other.String()))
	}
}

func evalPrefix(node *parser.PrefixExpression, operator string, right Value) (Value, error) {
	switch operator {
	case "!":
		rTruthy, err := isTruthy(right)
		if err != nil {
			return NilValue{}, NewRuntimeError(node, err.Error())
		}

		return BoolValue{V: !rTruthy}, nil
	case "-":
		switch v := right.(type) {
		case IntValue:
			return IntValue{V: -v.V}, nil
		case FloatValue:
			return FloatValue{V: -v.V}, nil
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operand, %s, for unary '-'", right.String()))
		}
	default:
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown prefix operator: %s", operator))
	}
}

func isTruthy(val Value) (bool, error) {
	b, ok := val.(BoolValue)
	if !ok {
		return false, fmt.Errorf("condition must be boolean")
	}
	return b.V, nil
}
