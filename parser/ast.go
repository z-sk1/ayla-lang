package parser

import (
	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/token"
)

type Statement interface{}
type Expression interface{}

const (
	_ int = iota
	LOWEST
	OR          // ||
	AND         // &&
	EQUALS      // == !=
	LESSGREATER // < >
	SUM         // + -
	PRODUCT     // * /
	PREFIX      // !x -z
)

var precedences = map[token.TokenType]int{
	token.OR:  OR,
	token.AND: AND,

	token.EQ:     EQUALS,
	token.NOT_EQ: EQUALS,

	token.LT:  LESSGREATER,
	token.GT:  LESSGREATER,
	token.LTE: LESSGREATER,
	token.GTE: LESSGREATER,

	token.PLUS:  SUM,
	token.MINUS: SUM,

	token.ASTERISK: PRODUCT,
	token.SLASH:    PRODUCT,
}

type VarStatement struct {
	Name  string
	Value Expression
}

type ConstStatement struct {
	Name  string
	Value Expression
}

type AssignmentStatement struct {
	Name  string
	Value Expression
}

type PrintStatement struct {
	Value Expression
}

type ScanlnStatement struct {
	Name string
}

type IfStatement struct {
	Condition   Expression
	Consequence []Statement
	Alternative []Statement // optional else block
}

type FuncStatement struct {
	Name   string
	Params []string
	Body   []Statement
}

type FuncCall struct {
	Name string
	Args []Expression
}

type ForStatement struct {
	Init      Statement  // egg i = 0;
	Condition Expression // i < 5;
	Post      Statement  // i = i + 1
	Body      []Statement
}

type WhileStatement struct {
	Condition Expression // i < 5
	Body      []Statement
}

type BreakStatement struct{}
type ContinueStatement struct{}

type ReturnStatement struct {
	Value Expression
}

type ArrayLiteral struct {
    Elements []Expression
}

type IndexExpression struct {
    Left  Expression
    Index Expression
}

type IndexAssignmentStatement struct {
	Left Expression
	Index Expression
	Value Expression
}


type IntLiteral struct {
	Value int
}

type IntCastExpression struct {
	Value Expression
}

type StringLiteral struct {
	Value string
}

type StringCastExpression struct {
	Value Expression
}

type BoolLiteral struct {
	Value bool
}

type Identifier struct {
	Value string
}

type ExpressionStatement struct {
	Expression Expression
}

type InfixExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

type PrefixExpression struct {
	Operator string
	Right    Expression
}

type GroupedExpression struct {
	Expression Expression
}

type Parser struct {
	l       *lexer.Lexer
	curTok  token.Token
	peekTok token.Token
}
