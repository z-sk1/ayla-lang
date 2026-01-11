package parser

import (
	"github.com/z-sk1/ayla-lang/token"
)

type Node interface {
	Pos() (int, int)
}

type Statement interface {
	Node
}

type Expression interface {
	Node
}

type NodeBase struct {
	Token token.Token
}

func (n *NodeBase) Pos() (int, int) {
	return n.Token.Line, n.Token.Column
}

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
	MEMBER      // p.x
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

	token.DOT: MEMBER,
}

type VarStatement struct {
	NodeBase
	Name  string
	Type  *Identifier // if no type defaults to nil, and then automatically chooses type
	Value Expression
}

type MultiVarStatement struct {
	NodeBase
	Names []string
	Type *Identifier
	Value Expression 
}

type ConstStatement struct {
	NodeBase
	Name  string
	Type  *Identifier // if no type defaults to nil, and then automatically chooses type
	Value Expression
}

type AssignmentStatement struct {
	NodeBase
	Name  string
	Value Expression
}

type MultiAssignmentStatement struct {
	NodeBase
	Names []string
	Value Expression
}

type IfStatement struct {
	NodeBase
	Condition   Expression
	Consequence []Statement
	Alternative []Statement // optional else block
}

type ParametersClause struct {
	NodeBase
	Type  *Identifier
	Value string
}

type FuncStatement struct {
	NodeBase
	Name        string
	Params      []*ParametersClause
	Body        []Statement
	ReturnTypes []*Identifier
}

type FuncCall struct {
	NodeBase
	Name string
	Args []Expression
}

type ForStatement struct {
	NodeBase
	Init      Statement  // egg i = 0;
	Condition Expression // i < 5;
	Post      Statement  // i = i + 1
	Body      []Statement
}

type WhileStatement struct {
	NodeBase
	Condition Expression // i < 5
	Body      []Statement
}

type StructField struct {
	Name *Identifier
	Type *Identifier
}

type StructStatement struct {
	NodeBase
	Name   *Identifier
	Fields []*StructField
}

type SwitchStatement struct {
	NodeBase
	Value   Expression
	Cases   []*CaseClause
	Default *DefaultClause
}

type CaseClause struct {
	NodeBase
	Expr Expression
	Body []Statement
}

type DefaultClause struct {
	NodeBase
	Body []Statement
}

type BreakStatement struct {
	NodeBase
}

type ContinueStatement struct {
	NodeBase
}

type ReturnStatement struct {
	NodeBase
	Values []Expression
}

type ArrayLiteral struct {
	NodeBase
	Elements []Expression
}

type IndexExpression struct {
	NodeBase
	Left  Expression
	Index Expression
}

type IndexAssignmentStatement struct {
	NodeBase
	Left  Expression
	Index Expression
	Value Expression
}

type IntLiteral struct {
	NodeBase
	Value int
}

type FloatLiteral struct {
	NodeBase
	Value float64
}

type StringLiteral struct {
	NodeBase
	Value string
}

type InterpolatedString struct {
	NodeBase
	Parts []Expression
}

type BoolLiteral struct {
	NodeBase
	Value bool
}

type NilLiteral struct {
	NodeBase
}

type TupleLiteral struct {
	NodeBase
	Values []Expression
}

type StructLiteral struct {
	NodeBase
	TypeName *Identifier
	Fields   map[string]Expression
}

type AnonymousStructLiteral struct {
	NodeBase
	Fields map[string]Expression
}

type MemberExpression struct {
	NodeBase
	Left  Expression  // p
	Field *Identifier // x
}

type MemberAssignmentStatement struct {
	NodeBase
	Object Expression  // p
	Field  *Identifier // x
	Value  Expression
}

type Identifier struct {
	NodeBase
	Value string
}

type ExpressionStatement struct {
	NodeBase
	Expression Expression
}

type InfixExpression struct {
	NodeBase
	Left     Expression
	Operator string
	Right    Expression
}

type PrefixExpression struct {
	NodeBase
	Operator string
	Right    Expression
}

type GroupedExpression struct {
	NodeBase
	Expression Expression
}
