package parser

import (
	"github.com/z-sk1/ayla-lang/token"
)

type Node interface {
	Pos() (int, int)
}

type TypeNode interface {
	Node
	typeNode()
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
	CALL        // ()
	INDEX       // []
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
	token.MOD:      PRODUCT,

	token.DOT:      MEMBER,
	token.LPAREN:   CALL,
	token.LBRACKET: INDEX,
}

type VarStatement struct {
	NodeBase
	Name  *Identifier
	Type  TypeNode // if no type defaults to nil, and then automatically chooses type
	Value Expression
}

type VarStatementNoKeyword struct {
	NodeBase
	Name  *Identifier
	Value Expression
}

type MultiVarStatement struct {
	NodeBase
	Names []*Identifier
	Type  TypeNode
	Value Expression
}

type MultiVarStatementNoKeyword struct {
	NodeBase
	Names []*Identifier
	Value Expression
}

type ConstStatement struct {
	NodeBase
	Name  *Identifier
	Type  TypeNode // if no type defaults to nil, and then automatically chooses type
	Value Expression
}

type MultiConstStatement struct {
	NodeBase
	Names []*Identifier
	Type  TypeNode
	Value Expression
}

type AssignmentStatement struct {
	NodeBase
	Name  *Identifier
	Value Expression
}

type MultiAssignmentStatement struct {
	NodeBase
	Names []*Identifier
	Value Expression
}

type TypeStatement struct {
	NodeBase
	Name  *Identifier
	Type  TypeNode
	Alias bool
}

type StructType struct {
	NodeBase
	Fields []*StructField
}

func (*StructType) typeNode() {}

type IdentType struct {
	NodeBase
	Name string
}

func (*IdentType) typeNode() {}

type ArrayType struct {
	NodeBase
	Elem TypeNode
}

func (*ArrayType) typeNode() {}

type SpawnStatement struct {
	NodeBase
	Body []Statement
}

type IfStatement struct {
	NodeBase
	Condition   Expression
	Consequence []Statement
	Alternative []Statement // optional else block
}

type ParametersClause struct {
	NodeBase
	Type TypeNode
	Name *Identifier
}

type FuncStatement struct {
	NodeBase
	Name        *Identifier
	Params      []*ParametersClause
	Body        []Statement
	ReturnTypes []*Identifier
}

type FuncCall struct {
	NodeBase
	Name *Identifier
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
	Type TypeNode
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

type TypeAssertExpression struct {
	NodeBase
	Expr Expression
	Type TypeNode
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
