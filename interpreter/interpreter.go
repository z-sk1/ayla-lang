package interpreter

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/z-sk1/ayla-lang/parser"
)

type Func struct {
	Params []*parser.ParametersClause
	Body   []parser.Statement
}

type BuiltinFunc struct {
	Name  string
	Arity int
	Fn    func(node *parser.FuncCall, args []Value) (Value, error)
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
}

type Interpreter struct {
	env     *Environment
	typeEnv map[string]*StructType
}

func New() *Interpreter {
	env := NewEnvironment()
	typeEnv := make(map[string]*StructType)
	registerBuiltins(env)
	return &Interpreter{env: env, typeEnv: typeEnv}
}

func NewEnvironment() *Environment {
	return &Environment{
		store:    make(map[string]Value),
		funcs:    make(map[string]*Func),
		builtins: make(map[string]*BuiltinFunc),
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
	val, ok := e.store[name]
	return val, ok
}

func (e *Environment) Set(name string, val Value) Value {
	e.store[name] = val
	return val
}

func (e *Environment) GetFunc(name string) (*Func, bool) {
	f, ok := e.funcs[name]
	return f, ok
}

func (e *Environment) SetFunc(name string, f *Func) {
	e.funcs[name] = f
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

func registerBuiltins(env *Environment) {
	env.builtins["int"] = &BuiltinFunc{
		Name:  "int",
		Arity: 1,
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]

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
				return NilValue{}, NewRuntimeError(node, "unsupported int() conversion")
			}
		},
	}

	env.builtins["float"] = &BuiltinFunc{
		Name:  "float",
		Arity: 1,
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]

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
				return NilValue{}, NewRuntimeError(node, "unsupported float() conversion")
			}
		},
	}

	env.builtins["string"] = &BuiltinFunc{
		Name:  "string",
		Arity: 1,
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]

			switch v.Type() {
			case STRING:
				return StringValue{V: v.(StringValue).V}, nil
			case BOOL:
				if v.(BoolValue).V {
					return StringValue{V: "true"}, nil
				}
				return StringValue{V: "false"}, nil
			case INT:
				n := strconv.Itoa(v.(IntValue).V)
				return StringValue{V: n}, nil
			case FLOAT:
				n := strconv.FormatFloat(v.(FloatValue).V, 'f', -1, 64)
				return StringValue{V: n}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "unsupported string() conversion")
			}
		},
	}

	env.builtins["bool"] = &BuiltinFunc{
		Name:  "bool",
		Arity: 1,
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]

			switch v.Type() {
			case BOOL:
				return BoolValue{V: v.(BoolValue).V}, nil
			case INT:
				return BoolValue{V: v.(IntValue).V != 0}, nil
			case FLOAT:
				return BoolValue{V: v.(FloatValue).V != 0}, nil
			case STRING:
				return BoolValue{V: v.(StringValue).V != ""}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "unsupported bool() conversion")
			}
		},
	}

	env.builtins["len"] = &BuiltinFunc{
		Name:  "len",
		Arity: 1,
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]
			switch v.Type() {
			case STRING:
				return IntValue{V: len(v.(StringValue).V)}, nil
			case ARRAY:
				return IntValue{V: len(v.(ArrayValue).Elements)}, nil
			default:
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("len() not supported for type %s", v.Type()))
			}
		},
	}

	env.builtins["type"] = &BuiltinFunc{
		Name:  "type",
		Arity: 1,
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]
			return StringValue{V: string(v.Type())}, nil
		},
	}

	env.builtins["explode"] = &BuiltinFunc{
		Name:  "explode",
		Arity: -1,
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
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
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
			for _, v := range args {
				if v.Type() != NIL {
					fmt.Println(v.String())
				}
			}
			return NilValue{}, nil
		},
	}

	env.builtins["tsaln"] = &BuiltinFunc{
		Name:  "tsaln",
		Arity: 1,
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
			ident, ok := node.Args[0].(*parser.Identifier)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "tsaln expects a variable")
			}

			varName := ident.Value

			// is it const?
			if v, ok := env.Get(varName); ok {
				if _, isConst := v.(ConstValue); isConst {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot assign to const %s", varName))
				}
			}

			var input string
			fmt.Scanln(&input)
			env.Set(varName, StringValue{V: input})

			return NilValue{}, nil
		},
	}

	env.builtins["push"] = &BuiltinFunc{
		Name:  "push",
		Arity: 2,
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
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
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
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
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
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
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
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
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
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
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
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
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
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
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
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

func (i *Interpreter) EvalStatement(s parser.Statement) (ControlSignal, error) {
	if s == nil {
		return NilValue{}, RuntimeError{Message: "EvalStatement got nil statement"}
	}

	switch stmt := s.(type) {
	case *parser.VarStatement:
		var val Value

		if stmt.Value == nil {
			val = NilValue{}
		} else {
			var err error
			val, err = i.EvalExpression(stmt.Value)
			if err != nil {
				return SignalNone{}, err
			}
		}

		if stmt.Type != nil {
			if string(val.Type()) != stmt.Type.Value {
				if val.Type() == INT && stmt.Type.Value == "float" {
					intVal := val.(IntValue)
					val = FloatValue{V: float64(intVal.V)}
				} else if val.Type() != BOOL && stmt.Type.Value == "bool" {
					val = BoolValue{V: isTruthy(val)}
				} else {
					return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("type mismatch: '%s' assigned to a '%s'", string(val.Type()), stmt.Type.Value))
				}
			}
		}

		// variable must not exist
		if _, ok := i.env.Get(stmt.Name); ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("cant redeclare var: %s", stmt.Name))
		}

		i.env.Set(stmt.Name, val)
		return SignalNone{}, nil

	case *parser.ConstStatement:
		var val Value

		if stmt.Value == nil {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("const, %s, must be initialised", stmt.Name))
		} else {
			var err error
			val, err = i.EvalExpression(stmt.Value)
			if err != nil {
				return SignalNone{}, err
			}
		}

		if stmt.Type != nil {
			if string(val.Type()) != stmt.Type.Value {
				if val.Type() == INT && stmt.Type.Value == "float" {
					intVal := val.(IntValue)
					val = FloatValue{V: float64(intVal.V)}
				} else if val.Type() != BOOL && stmt.Type.Value == "bool" {
					val = BoolValue{V: isTruthy(val)}
				} else {
					return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("type mismatch: '%s' assigned to a '%s'", string(val.Type()), stmt.Type.Value))
				}
			}
		}

		// check if variable already exist
		if _, ok := i.env.Get(stmt.Name); ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("cant redeclare const: %s", stmt.Name))
		}

		// store const val
		i.env.Set(stmt.Name, ConstValue{Value: val})
		return SignalNone{}, nil

	case *parser.AssignmentStatement:
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}

		existingVal, ok := i.env.Get(stmt.Name)
		if !ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("assignment to undefined variable: %v", stmt.Name))
		}

		if _, isConst := existingVal.(ConstValue); isConst {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("cannot reassign to const: %s", stmt.Name))
		}

		i.env.Set(stmt.Name, val)
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

		if val.Type() != expectedType {
			return NilValue{}, NewRuntimeError(stmt, fmt.Sprintf("field '%s' expects %v but got %v", stmt.Field.Value, expectedType, val.Type()))
		}

		structVal.Fields[stmt.Field.Value] = val
		return SignalNone{}, nil

	case *parser.FuncStatement:
		i.env.SetFunc(stmt.Name, &Func{Params: stmt.Params, Body: stmt.Body})
		return SignalNone{}, nil

	case *parser.ReturnStatement:
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}

		return SignalReturn{Value: val}, nil

	case *parser.ExpressionStatement:
		_, err := i.EvalExpression(stmt.Expression)
		if err != nil {
			return SignalNone{}, err
		}
		return SignalNone{}, nil

	case *parser.StructStatement:
		fields := make(map[string]ValueType)

		for _, field := range stmt.Fields {
			vt, err := i.resolveTypeFromName(stmt, field.Type.Value)
			if err != nil {
				return SignalNone{}, err
			}
			fields[field.Name.Value] = vt
		}

		i.typeEnv[stmt.Name.Value] = &StructType{
			Name:   stmt.Name.Value,
			Fields: fields,
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

		if isTruthy(cond) {
			if stmt.Consequence != nil {
				return i.EvalStatements(stmt.Consequence)
			}
		} else {
			if stmt.Alternative != nil {
				return i.EvalStatements(stmt.Alternative)
			}
		}
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

			for _, s := range c.Body {
				sig, err := i.EvalStatement(s)
				if err != nil {
					return SignalNone{}, err
				}

				if _, ok := sig.(SignalNone); !ok {
					return sig, nil
				}
			}

			return SignalNone{}, nil
		}

		if stmt.Default != nil {
			for _, s := range stmt.Default.Body {
				sig, err := i.EvalStatement(s)
				if err != nil {
					return SignalNone{}, err
				}

				if _, ok := sig.(SignalNone); !ok {
					return sig, nil
				}
			}
		}

		return SignalNone{}, nil

	case *parser.ForStatement:
		i.EvalStatement(stmt.Init)
		for {
			cond, err := i.EvalExpression(stmt.Condition)
			if err != nil {
				return SignalNone{}, err
			}

			if !isTruthy(cond) {
				break
			}

			sig, err := i.EvalStatements(stmt.Body)
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
		return SignalNone{}, nil

	case *parser.WhileStatement:
		for {
			cond, err := i.EvalExpression(stmt.Condition)
			if err != nil {
				return SignalNone{}, err
			}

			if !isTruthy(cond) {
				break
			}

			sig, err := i.EvalStatements(stmt.Body)
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
			return NilValue{}, NewRuntimeError(expr, "unknown struct type "+expr.TypeName.Value)
		}

		structType := val
		if !ok {
			return NilValue{}, NewRuntimeError(expr, expr.TypeName.Value+" is not a struct type")
		}

		fields := make(map[string]Value)

		for name, e := range expr.Fields {
			expectedType, ok := structType.Fields[name]
			if !ok {
				return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("unknown field '%s' in struct %s", name, structType.Name))
			}

			v, err := i.EvalExpression(e)
			if err != nil {
				return NilValue{}, err
			}

			if v.Type() != structType.Fields[name] {
				return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("field '%s' expects %v but got %v", name, expectedType, v.Type()))
			}

			fields[name] = v
		}

		return &StructValue{
			TypeName: structType,
			Fields:   fields,
		}, nil

	case *parser.AnonymousStructLiteral:
		fields := make(map[string]Value)
		fieldTypes := make(map[string]ValueType)

		for name, e := range expr.Fields {
			v, err := i.EvalExpression(e)
			if err != nil {
				return NilValue{}, err
			}

			if expected, ok := fieldTypes[name]; ok {
				if v.Type() != expected {
					return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("field '%s' type %v should be %v", name, v.Type(), fieldTypes[name]))
				}
			}

			fields[name] = v
			fieldTypes[name] = v.Type()
		}

		return &StructValue{
			TypeName: &StructType{
				Name:   "<anon>",
				Fields: fieldTypes,
			},
			Fields: fields,
		}, nil

	case *parser.FuncCall:
		// built in?
		if b, ok := i.env.builtins[expr.Name]; ok {
			args := []Value{}

			for _, a := range expr.Args {
				v, err := i.EvalExpression(a)
				if err != nil {
					return NilValue{}, err
				}
				args = append(args, v)
			}

			if b.Arity >= 0 && len(args) != b.Arity {
				return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("expected %d args, got %d", b.Arity, len(args)))
			}

			return b.Fn(expr, args)
		}

		// user-defined
		fn, ok := i.env.GetFunc(expr.Name)
		if !ok {
			return NilValue{}, NewRuntimeError(e, fmt.Sprintf("unknown function: %s", expr.Name))
		}

		if len(fn.Params) != len(expr.Args) {
			return NilValue{}, NewRuntimeError(e, "wrong numbers of args")
		}

		// create new env for func call
		newEnv := NewEnvironment()

		// copy stores
		for k, v := range i.env.store {
			newEnv.store[k] = v
		}

		// copy funcs
		for k, v := range i.env.funcs {
			newEnv.funcs[k] = v
		}

		// set params
		for idx, param := range fn.Params {
			val, err := i.EvalExpression(expr.Args[idx])
			if err != nil {
				return NilValue{}, err
			}

			// enforce type if parameter has one 
			if param.Type != nil {
				
			}

			newEnv.Set(param.Value, val)
		}

		// execute body
		oldEnv := i.env
		i.env = newEnv

		sig, err := i.EvalStatements(fn.Body)
		if err != nil {
			return NilValue{}, err
		}

		i.env = oldEnv

		if ret, ok := sig.(SignalReturn); ok {
			return ret.Value, nil
		}

		return NilValue{}, nil

	case *parser.InfixExpression:
		if expr.Operator == "&&" {
			left, err := i.EvalExpression(expr.Left)
			if err != nil {
				return NilValue{}, err
			}

			if !isTruthy(left) {
				return BoolValue{V: false}, nil
			}

			right, err := i.EvalExpression(expr.Right)
			if err != nil {
				return NilValue{}, err
			}

			return BoolValue{V: isTruthy(right)}, nil
		}

		if expr.Operator == "||" {
			left, err := i.EvalExpression(expr.Left)
			if err != nil {
				return NilValue{}, err
			}

			if isTruthy(left) {
				return BoolValue{V: true}, nil
			}

			right, err := i.EvalExpression(expr.Right)
			if err != nil {
				return NilValue{}, err
			}

			return BoolValue{V: isTruthy(right)}, nil
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

	if val.Type() != expectedType {
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

	// type mismatch check
	if left.Type() != right.Type() {
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("type mismatch: %s %s %s", left.Type(), op, right.Type()))
	}

	if left.Type() == INT && right.Type() == FLOAT {
		return evalFloatInfix(node, FloatValue{V: float64(left.(IntValue).V)}, op, right.(FloatValue))
	}

	if left.Type() == FLOAT && right.Type() == INT {
		return evalFloatInfix(node, left.(FloatValue), op, FloatValue{V: float64(right.(IntValue).V)})
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
		return BoolValue{V: !isTruthy(right)}, nil
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

func isTruthy(val Value) bool {
	switch v := val.(type) {
	case IntValue:
		return v.V != 0
	case FloatValue:
		return v.V != 0
	case BoolValue:
		return v.V
	case StringValue:
		return v.V != ""
	case NilValue:
		return false
	default:
		return false
	}
}
