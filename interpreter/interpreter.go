package interpreter

import (
	"fmt"
	"strconv"

	"github.com/z-sk1/ayla-lang/parser"
)

type Func struct {
	Params []string
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

type Binding interface{}

type Environment struct {
	store    map[string]Binding
	funcs    map[string]*Func
	builtins map[string]*BuiltinFunc
}

type Interpreter struct {
	env *Environment
}

func New() *Interpreter {
	env := NewEnvironment()
	registerBuiltins(env)
	return &Interpreter{env: env}
}

func registerBuiltins(env *Environment) {
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

	env.builtins["explode"] = &BuiltinFunc{
		Name:  "explode",
		Arity: -1,
		Fn: func(node *parser.FuncCall, args []Value) (Value, error) {
			for _, v := range args {
				fmt.Println(v.String())
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
}

func NewEnvironment() *Environment {
	return &Environment{
		store:    make(map[string]Binding),
		funcs:    make(map[string]*Func),
		builtins: make(map[string]*BuiltinFunc),
	}
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("Runtime error at %d:%d: %s", e.Line, e.Column, e.Message)
}

func NewRuntimeError(node parser.Node, msg string) RuntimeError {
	line, col := node.Pos()
	return RuntimeError{Message: msg, Line: line, Column: col}
}

func (e *Environment) Get(name string) (Binding, bool) {
	val, ok := e.store[name]
	return val, ok
}

func (e *Environment) Set(name string, val Binding) Binding {
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
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return NilValue{}, err
		}

		// variable must not exist
		if _, ok := i.env.Get(stmt.Name); ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("cant redeclare var: %s", stmt.Name))
		}

		i.env.Set(stmt.Name, val)
		return SignalNone{}, nil

	case *parser.ConstStatement:
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return NilValue{}, err
		}

		// check if variable already exist
		if _, ok := i.env.Get(stmt.Name); ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("cant redeclare const %s", stmt.Name))
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
			return SignalNone{}, NewRuntimeError(s, "assignment to undefined variable")
		}

		if _, isConst := existingVal.(ConstValue); isConst {
			return SignalNone{}, NewRuntimeError(s, "cannot reassign to const")
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
			return SignalNone{}, NewRuntimeError(s, "assignment to non-array")
		}

		index, err := i.EvalExpression(stmt.Index)
		if err != nil {
			return SignalNone{}, err
		}

		idxVal, ok := index.(IntValue)
		if !ok {
			return SignalNone{}, NewRuntimeError(s, "array index must be int")
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(arrVal.Elements) {
			return SignalNone{}, NewRuntimeError(s, "array index out of bounds")
		}

		newVal, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}

		arrVal.Elements[idx] = newVal
		return SignalNone{}, nil

	case *parser.FuncStatement:
		i.env.SetFunc(stmt.Name, &Func{Params: stmt.Params, Body: stmt.Body})
		return SignalNone{}, nil

	case *parser.ReturnStatement:
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return NilValue{}, err
		}

		return SignalReturn{Value: val}, nil

	case *parser.ExpressionStatement:
		i.EvalExpression(stmt.Expression)
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
			return NilValue{}, err
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

	case *parser.Identifier:
		binding, ok := i.env.Get(expr.Value)
		if !ok {
			return NilValue{}, NewRuntimeError(e, "undefined variable")
		}

		if c, isConst := binding.(ConstValue); isConst {
			return c.Value, nil
		}

		return binding.(Value), nil

	case *parser.IntCastExpression:
		v, err := i.EvalExpression(expr.Value)
		if err != nil {
			return NilValue{}, err
		}

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
				return NilValue{}, NewRuntimeError(e, "could not convert string to int")
			}
			return IntValue{V: n}, nil
		default:
			return NilValue{}, NewRuntimeError(e, "unsupported int() conversion")
		}

	case *parser.FloatCastExpression:
		v, err := i.EvalExpression(expr.Value)
		if err != nil {
			return NilValue{}, err
		}

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
				return NilValue{}, NewRuntimeError(e, "could not convert string to float")
			}
			return FloatValue{V: n}, nil
		default:
			return NilValue{}, NewRuntimeError(e, "unsupported float() conversion")
		}

	case *parser.StringCastExpression:
		v, err := i.EvalExpression(expr.Value)
		if err != nil {
			return NilValue{}, err
		}

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
			return NilValue{}, NewRuntimeError(e, "unsupported string() conversion")
		}

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

		arr, ok := left.(ArrayValue)
		if !ok {
			return NilValue{}, NewRuntimeError(e, "indexing non-array")
		}

		idxVal, ok := index.(IntValue)
		if !ok {
			return NilValue{}, NewRuntimeError(e, "array index must be int")
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(arr.Elements) {
			return NilValue{}, NewRuntimeError(e, "array index out of bounds")
		}

		return arr.Elements[idx], nil

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

			newEnv.Set(param, val)
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

	default:
		return NilValue{}, nil
	}
}

func evalInfix(node *parser.InfixExpression, left Value, op string, right Value) (Value, error) {
	// type mismatch check
	if left.Type() != right.Type() {
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("type mismatch: %s %s %s", left.Type(), op, right.Type()))
	}

	if left.Type() == INT && right.Type() == FLOAT {
		return evalFloatInfix(node, FloatValue{V: float64(left.(IntValue).V)}, op, right.(FloatValue))
	}

	if left.Type() == FLOAT && right.Type() == INT {
		return evalFloatInfix(node, left.(FloatValue), op, FloatValue{V: float64(left.(IntValue).V)})
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
