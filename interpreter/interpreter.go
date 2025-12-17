package interpreter

import (
	"fmt"

	"github.com/z-sk1/ayla-lang/parser"
)

type Value interface{}

type Environment struct {
	store map[string]Value
}

type Interpreter struct {
	env *Environment
}

func New() *Interpreter {
	return &Interpreter{env: NewEnvironment()}
}

func NewEnvironment() *Environment {
	return &Environment{store: make(map[string]Value)}
}

func (e *Environment) Get(name string) (Value, bool) {
	val, ok := e.store[name]
	return val, ok
}

func (e *Environment) Set(name string, val Value) Value {
	e.store[name] = val
	return val
}

func (i *Interpreter) EvalStatements(stmts []parser.Statement) {
	for _, s := range stmts {
		if s == nil {
			continue
		}
		i.EvalStatement(s)
	}
}

func (i *Interpreter) EvalStatement(s parser.Statement) {
	if s == nil {
		panic("EvalStatement got nil statement")
	}

	switch stmt := s.(type) {
	case *parser.VarStatement:
		val := i.EvalExpression(stmt.Value)
		i.env.Set(stmt.Name, val)
	case *parser.PrintStatement:
		val := i.EvalExpression(stmt.Value)
		fmt.Println(val)
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
				i.EvalStatements(stmt.Consequence)
			}
		} else {
			if stmt.Alternative != nil {
				i.EvalStatements(stmt.Alternative)
			}
		}
	}
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
		return val
	case *parser.InfixExpression:
		left := i.EvalExpression(expr.Left)
		right := i.EvalExpression(expr.Right)
		return evalInfix(left, expr.Operator, right)
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
		case "==":
			return l == r
		case "!=":
			return l != r
		}
	}

	panic("unsupported operand types")
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
