package interpreter

import (
	"fmt"
	"strconv"

	"github.com/z-sk1/ayla-lang/parser"
)

type Value interface{}

type ControlSignal interface{}

type SignalNone struct{}
type SignalBreak struct{}
type SignalContinue struct{}

type SignalReturn struct {
	Value interface{}
}

type ConstValue struct {
	Value interface{}
}

type Func struct {
	Params []string
	Body   []parser.Statement
}

type RuntimeError struct {
	Message string
	Line    int
	Column  int
}

type Environment struct {
	store map[string]Value
	funcs map[string]*Func
}

type Interpreter struct {
	env *Environment
}

func New() *Interpreter {
	return &Interpreter{env: NewEnvironment()}
}

func NewEnvironment() *Environment {
	return &Environment{
		store: make(map[string]Value),
		funcs: make(map[string]*Func),
	}
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("Runtime error at %d:%d: %s", e.Line, e.Column, e.Message)
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
		return nil, RuntimeError{Message: "EvalStatement got nil statement"}
	}

	switch stmt := s.(type) {
	case *parser.VarStatement:
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return nil, err
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
			return nil, err
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
			return nil, err
		}

		// variable must already exist
		if _, ok := i.env.Get(stmt.Name); !ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("assignment to undefined variable: %s", stmt.Name))
		}

		// check for const
		if existingVal, ok := i.env.Get(stmt.Name); ok {
			if _, isConst := existingVal.(ConstValue); isConst {
				return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("cannot reassign to const: %s", stmt.Name))
			}
		}

		i.env.Set(stmt.Name, val)
		return SignalNone{}, nil

	case *parser.IndexAssignmentStatement:
		leftVal, err := i.EvalExpression(stmt.Left)
		if err != nil {
			return nil, err
		}

		arrVal, ok := leftVal.([]interface{})
		if !ok {
			return SignalNone{}, NewRuntimeError(s, "assignment to non-array")
		}

		idxVal, err := i.EvalExpression(stmt.Index)
		if err != nil {
			return nil, err
		}

		idx, ok := idxVal.(int)
		if !ok {
			return SignalNone{}, NewRuntimeError(s, "array index must be int")
		}

		if idx < 0 || idx >= len(arrVal) {
			return SignalNone{}, NewRuntimeError(s, "array index out of bounds")
		}

		newVal, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return nil, err
		}

		arrVal[idx] = newVal
		return SignalNone{}, nil

	case *parser.FuncStatement:
		i.env.SetFunc(stmt.Name, &Func{Params: stmt.Params, Body: stmt.Body})
		return SignalNone{}, nil

	case *parser.ReturnStatement:
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return nil, err
		}

		return SignalReturn{Value: val}, nil

	case *parser.ExpressionStatement:
		i.EvalExpression(stmt.Expression)
		return SignalNone{}, nil

	case *parser.PrintStatement:
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return nil, err
		}

		fmt.Println(val)
		return SignalNone{}, nil

	case *parser.ScanlnStatement:
		var input string
		fmt.Scanln(&input)
		i.env.Set(stmt.Name, input)
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
			return nil, err
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

func (i *Interpreter) EvalExpression(e parser.Expression) (interface{}, error) {
	switch expr := e.(type) {
	case *parser.IntLiteral:
		return expr.Value, nil

	case *parser.StringLiteral:
		return expr.Value, nil

	case *parser.BoolLiteral:
		return expr.Value, nil

	case *parser.Identifier:
		val, ok := i.env.Get(expr.Value)
		if !ok {
			return nil, NewRuntimeError(e, fmt.Sprintf("undefined variable: %s", expr.Value))
		}

		// unwrap const from ConstValue{}
		if constVal, isConst := val.(ConstValue); isConst {
			return constVal.Value, nil
		}

		return val, nil

	case *parser.IntCastExpression:
		v, err := i.EvalExpression(expr.Value)
		if err != nil {
			return nil, err
		}

		switch x := v.(type) {
		case int:
			return x, nil
		case float64:
			return int(x), nil
		case bool:
			if x {
				return 1, nil
			}
			return 0, nil
		case string:
			n, err := strconv.Atoi(x)
			if err != nil {
				return nil, NewRuntimeError(e, "could not convert string to int")
			}
			return n, nil
		default:
			return nil, NewRuntimeError(e, "unsupported int() conversion")
		}

	case *parser.FloatCastExpression:
		v, err := i.EvalExpression(expr.Value)
		if err != nil {
			return nil, err
		}

		switch x := v.(type) {
		case float64:
			return x, nil
		case int:
			return float64(x), nil
		case bool:
			if x {
				return 1.0, nil
			}
			return 0.0, nil
		case string:
			n, err := strconv.ParseFloat(x, 64)
			if err != nil {
				return nil, NewRuntimeError(e, "could not convert string to float")
			}
			return n, nil
		default:
			return nil, NewRuntimeError(e, "unsupported float() conversion")
		}

	case *parser.StringCastExpression:
		v, err := i.EvalExpression(expr.Value)
		if err != nil {
			return nil, err
		}

		switch x := v.(type) {
		case string:
			return x, nil
		case bool:
			if x {
				return "true", nil
			}
			return "false", nil
		case int:
			n := strconv.Itoa(x)
			return n, nil
		case float64:
			n := strconv.FormatFloat(x, 'f', -1, 64)
			return n, nil
		default:
			return nil, NewRuntimeError(e, "unsupported string() conversion")
		}

	case *parser.ArrayLiteral:
		elements := []interface{}{}
		for _, el := range expr.Elements {
			val, err := i.EvalExpression(el)
			if err != nil {
				return nil, err
			}

			elements = append(elements, val)
		}
		return elements, nil

	case *parser.IndexExpression:
		left, err := i.EvalExpression(expr.Left)
		if err != nil {
			return nil, err
		}

		index, err := i.EvalExpression(expr.Index)
		if err != nil {
			return nil, err
		}

		arr, ok := left.([]interface{})
		if !ok {
			return nil, NewRuntimeError(e, "indexing non-array")
		}

		idx, ok := index.(int)
		if !ok {
			return nil, NewRuntimeError(e, "array index must be int")
		}

		if idx < 0 || idx >= len(arr) {
			return nil, NewRuntimeError(e, "array index out of bounds")
		}

		return arr[idx], nil

	case *parser.FuncCall:
		fn, ok := i.env.GetFunc(expr.Name)
		if !ok {
			return nil, NewRuntimeError(e, fmt.Sprintf("unknown function: %s", expr.Name))
		}

		if len(fn.Params) != len(expr.Args) {
			return nil, NewRuntimeError(e, "wrong numbers of args")
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
				return nil, err
			}

			newEnv.Set(param, val)
		}

		// execute body
		oldEnv := i.env
		i.env = newEnv

		sig, err := i.EvalStatements(fn.Body)
		if err != nil {
			return nil, err
		}

		i.env = oldEnv

		if ret, ok := sig.(SignalReturn); ok {
			return ret.Value, nil
		}

		return nil, nil

	case *parser.InfixExpression:
		if expr.Operator == "&&" {
			left, err := i.EvalExpression(expr.Left)
			if err != nil {
				return nil, err
			}

			if !isTruthy(left) {
				return false, nil
			}

			right, err := i.EvalExpression(expr.Right)
			if err != nil {
				return nil, err
			}

			return isTruthy(right), nil
		}

		if expr.Operator == "||" {
			left, err := i.EvalExpression(expr.Left)
			if err != nil {
				return nil, err
			}

			if isTruthy(left) {
				return true, nil
			}

			right, err := i.EvalExpression(expr.Right)
			if err != nil {
				return nil, err
			}

			return isTruthy(right), nil
		}

		left, err := i.EvalExpression(expr.Left)
		if err != nil {
			return nil, err
		}

		right, err := i.EvalExpression(expr.Right)
		if err != nil {
			return nil, err
		}

		return evalInfix(expr, left, expr.Operator, right)

	case *parser.PrefixExpression:
		right, err := i.EvalExpression(expr.Right)
		if err != nil {
			return nil, err
		}

		return evalPrefix(expr, expr.Operator, right)

	case *parser.GroupedExpression:
		return i.EvalExpression(expr.Expression)

	default:
		return nil, nil
	}
}

func evalInfix(node *parser.InfixExpression, left interface{}, operator string, right interface{}) (interface{}, error) {
	switch l := left.(type) {

	case int:
		r, ok := right.(int)
		if !ok {
			return nil, NewRuntimeError(node, "type mismatch: int compared to non-int")
		}

		switch operator {
		case "+":
			return l + r, nil
		case "-":
			return l - r, nil
		case "*":
			return l * r, nil
		case "/":
			if r == 0 {
				return nil, NewRuntimeError(node, "undefined: cant divide by zero")
			}

			return l / r, nil
		case "==":
			return l == r, nil
		case "!=":
			return l != r, nil
		case ">":
			return l > r, nil
		case "<":
			return l < r, nil
		case ">=":
			return l >= r, nil
		case "<=":
			return l <= r, nil
		}

	case float64:
		r, ok := right.(float64)
		if !ok {
			if ri, isInt := right.(int); isInt {
				r = float64(ri)
			} else {
				return nil, NewRuntimeError(node, "type mismatch: float compared to non-float")
			}
		}

		switch operator {
		case "+":
			return l + r, nil
		case "-":
			return l - r, nil
		case "*":
			return l * r, nil
		case "/":
			if r == 0 {
				return nil, NewRuntimeError(node, "undefined: cant divide by zero")
			}

			return l / r, nil
		case "==":
			return l == r, nil
		case "!=":
			return l != r, nil
		case ">":
			return l > r, nil
		case "<":
			return l < r, nil
		case ">=":
			return l >= r, nil
		case "<=":
			return l <= r, nil
		}

	case string:
		r, ok := right.(string)
		if !ok {
			return nil, NewRuntimeError(node, "type mismatch: string compared to non-string")
		}

		switch operator {
		case "+":
			return l + r, nil
		case "==":
			return l == r, nil
		case "!=":
			return l != r, nil
		}

	case bool:
		r, ok := right.(bool)
		if !ok {
			return nil, NewRuntimeError(node, "type mismatch: bool compared to non-bool")
		}

		switch operator {
		case "&&":
			return l && r, nil
		case "||":
			return l || r, nil
		case "==":
			return l == r, nil
		case "!=":
			return l != r, nil
		}
	}

	return nil, NewRuntimeError(node, "unsupported operand types")
}

func evalPrefix(node *parser.PrefixExpression, operator string, right interface{}) (interface{}, error) {
	switch operator {
	case "!":
		return !isTruthy(right), nil
	default:
		return nil, NewRuntimeError(node, fmt.Sprintf("unknown prefix operator: %s", operator))
	}
}

func isTruthy(val interface{}) bool {
	switch v := val.(type) {
	case int:
		return v != 0
	case bool:
		return v
	case string:
		return v != ""
	default:
		return false
	}
}
