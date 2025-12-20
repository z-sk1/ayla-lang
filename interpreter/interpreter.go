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

func (i *Interpreter) EvalStatements(stmts []parser.Statement) ControlSignal {
	for _, s := range stmts {
		sig := i.EvalStatement(s)

		switch sig.(type) {
		case SignalReturn, SignalBreak, SignalContinue:
			return sig
		}
	}
	return SignalNone{}
}

func (i *Interpreter) EvalStatement(s parser.Statement) ControlSignal {
	if s == nil {
		panic("EvalStatement got nil statement")
	}

	switch stmt := s.(type) {
	case *parser.VarStatement:
		val := i.EvalExpression(stmt.Value)

		// variable must not exist
		if _, ok := i.env.Get(stmt.Name); ok {
			fmt.Printf("cant redeclare variable with egg, to reassign just do '%s = %v'\n", stmt.Name, stmt.Value)
			panic("\ndeclaration to already defined variable " + stmt.Name)
		}

		i.env.Set(stmt.Name, val)
		return SignalNone{}

	case *parser.ConstStatement:
		val := i.EvalExpression(stmt.Value)

		// check if variable already exist
		if _, ok := i.env.Get(stmt.Name); ok {
			panic("cant redeclare const " + stmt.Name)
		}

		// store const val
		i.env.Set(stmt.Name, ConstValue{Value: val})
		return SignalNone{}

	case *parser.AssignmentStatement:
		val := i.EvalExpression(stmt.Value)

		// variable must already exist
		if _, ok := i.env.Get(stmt.Name); !ok {
			fmt.Println("declare variable with egg first, cant reassign undeclared var")
			panic("\nassignment to undefined variable: " + stmt.Name)
		}

		// check for const
		if existingVal, ok := i.env.Get(stmt.Name); ok {
			if _, isConst := existingVal.(ConstValue); isConst {
				panic("cannot reassign to const: " + stmt.Name)
			}
		}

		i.env.Set(stmt.Name, val)
		return SignalNone{}

	case *parser.FuncStatement:
		i.env.SetFunc(stmt.Name, &Func{Params: stmt.Params, Body: stmt.Body})
		return SignalNone{}

	case *parser.ReturnStatement:
		val := i.EvalExpression(stmt.Value)
		return SignalReturn{Value: val}

	case *parser.ExpressionStatement:
		i.EvalExpression(stmt.Expression)

	case *parser.PrintStatement:
		val := i.EvalExpression(stmt.Value)
		fmt.Println(val)
		return SignalNone{}

	case *parser.ScanlnStatement:
		var input string
		fmt.Scanln(&input)
		i.env.Set(stmt.Name, input)
		return SignalNone{}

	case *parser.IfStatement:
		if stmt.Condition == nil {
			panic("if statement missing condition")
		}
		if stmt.Consequence == nil {
			panic("if statement missing consequence")
		}
		cond := i.EvalExpression(stmt.Condition)

		if isTruthy(cond) {
			if stmt.Consequence != nil {
				return i.EvalStatements(stmt.Consequence)
			}
		} else {
			if stmt.Alternative != nil {
				return i.EvalStatements(stmt.Alternative)
			}
		}
		return SignalNone{}

	case *parser.ForStatement:
		i.EvalStatement(stmt.Init)
		for isTruthy(i.EvalExpression(stmt.Condition)) {
			sig := i.EvalStatements(stmt.Body)

			switch sig.(type) {
			case SignalBreak:
				return SignalNone{}
			case SignalContinue:
				i.EvalStatement(stmt.Post)
				continue
			case SignalReturn:
				return sig
			}
			i.EvalStatement(stmt.Post)
		}
		return SignalNone{}
	case *parser.WhileStatement:
		for isTruthy(i.EvalExpression(stmt.Condition)) {
			sig := i.EvalStatements(stmt.Body)

			switch sig.(type) {
			case SignalBreak:
				return SignalNone{}
			case SignalContinue:
				continue
			case SignalReturn:
				return sig
			}
		}
		return SignalNone{}

	case *parser.BreakStatement:
		return SignalBreak{}
	case *parser.ContinueStatement:
		return SignalContinue{}
	}

	return SignalNone{}
}

func (i *Interpreter) EvalExpression(e parser.Expression) interface{} {
	switch expr := e.(type) {
	case *parser.IntLiteral:
		return expr.Value

	case *parser.StringLiteral:
		return expr.Value

	case *parser.BoolLiteral:
		return expr.Value

	case *parser.Identifier:
		val, ok := i.env.Get(expr.Value)
		if !ok {
			panic("undefined variable: " + expr.Value)
		}

		// unwrap const from ConstValue{}
		if constVal, isConst := val.(ConstValue); isConst {
			return constVal.Value
		}

		return val

	case *parser.IntCastExpression:
		v := i.EvalExpression(expr.Value)

		switch x := v.(type) {
		case int:
			return x
		case bool:
			if x {
				return 1
			}
			return 0
		case string:
			n, err := strconv.Atoi(x)
			if err != nil {
				panic("could not convert string to int")
			}
			return n
		default:
			panic("unsupported int() conversion")
		}

	case *parser.StringCastExpression:
		v := i.EvalExpression(expr.Value)

		switch x := v.(type) {
		case string:
			return x
		case bool:
			if x {
				return "true"
			}
			return "false"
		case int:
			n := strconv.Itoa(x)
			return n
		default:
			panic("unsupport string() conversion")
		}

	case *parser.FuncCall:
		fn, ok := i.env.GetFunc(expr.Name)
		if !ok {
			panic("unknown function: " + expr.Name)
		}

		if len(fn.Params) != len(expr.Args) {
			panic("wrong numbers of args")
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
			newEnv.Set(param, i.EvalExpression(expr.Args[idx]))
		}

		// execute body
		oldEnv := i.env
		i.env = newEnv

		sig := i.EvalStatements(fn.Body)
		i.env = oldEnv

		if ret, ok := sig.(SignalReturn); ok {
			return ret.Value
		}

		return nil

	case *parser.InfixExpression:
		if expr.Operator == "&&" {
			left := i.EvalExpression(expr.Left)
			if !isTruthy(left) {
				return false
			}
			right := i.EvalExpression(expr.Right)
			return isTruthy(right)
		}

		if expr.Operator == "||" {
			left := i.EvalExpression(expr.Left)
			if isTruthy(left) {
				return true
			}
			right := i.EvalExpression(expr.Right)
			return isTruthy(right)
		}

		left := i.EvalExpression(expr.Left)
		right := i.EvalExpression(expr.Right)
		return evalInfix(left, expr.Operator, right)

	case *parser.PrefixExpression:
		right := i.EvalExpression(expr.Right)
		return evalPrefix(expr.Operator, right)

	case *parser.GroupedExpression:
		return i.EvalExpression(expr.Expression)

	default:
		return nil
	}
}

func evalInfix(left interface{}, operator string, right interface{}) interface{} {
	switch l := left.(type) {

	case int:
		r, ok := right.(int)
		if !ok {
			panic("type mismatch: int compared to non-int")
		}

		switch operator {
		case "+":
			return l + r
		case "-":
			return l - r
		case "*":
			return l * r
		case "/":
			return l / r
		case "==":
			return l == r
		case "!=":
			return l != r
		case ">":
			return l > r
		case "<":
			return l < r
		case ">=":
			return l >= r
		case "<=":
			return l <= r
		}

	case string:
		r, ok := right.(string)
		if !ok {
			panic("type mismatch: string compared to non-string")
		}

		switch operator {
		case "+":
			return l + r
		case "==":
			return l == r
		case "!=":
			return l != r
		}

	case bool:
		r, ok := right.(bool)
		if !ok {
			panic("type mismatch: bool compared to non-bool")
		}

		switch operator {
		case "&&":
			return l && r
		case "||":
			return l || r
		case "==":
			return l == r
		case "!=":
			return l != r
		}
	}

	panic("unsupported operand types")
}

func evalPrefix(operator string, right interface{}) interface{} {
	switch operator {
	case "!":
		return !isTruthy(right)
	default:
		panic("unknown prefix operator: " + operator)
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
