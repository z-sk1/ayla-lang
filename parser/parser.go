package parser

import (
	"fmt"
	"strconv"

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

type PrintStatement struct {
	Value Expression
}

type IfStatement struct {
	Condition   Expression
	Consequence []Statement
	Alternative []Statement // optional else block
}

type IntLiteral struct {
	Value int
}

type StringLiteral struct {
	Value string
}

type BoolLiteral struct {
	Value bool
}

type Identifier struct {
	Value string
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

func atoi(a string) int {
	val, _ := strconv.Atoi(a)
	return val
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curTok = p.peekTok
	p.peekTok = p.l.NextToken()
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekTok.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curTok.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) ParseProgram() []Statement {
	var statements []Statement
	for p.curTok.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			statements = append(statements, stmt)
		}
		p.nextToken()
	}

	return statements
}

func (p *Parser) parseStatement() Statement {
	switch p.curTok.Type {
	case token.VAR:
		return p.parseVarStatement()
	case token.PRINT:
		return p.parsePrintStatement()
	case token.IF:
		return p.parseIfStatement()
	default:
		return nil
	}
}

func (p *Parser) parseVarStatement() *VarStatement {
	stmt := &VarStatement{}

	// Expect next token to be identifier
	p.nextToken()
	if p.curTok.Type != token.IDENT {
		return nil
	}
	stmt.Name = p.curTok.Literal

	// Expect '='
	p.nextToken()
	if p.curTok.Type != token.ASSIGN {
		return nil
	}

	// Expression after '='
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	// Optional semicolon
	if p.peekTok.Type == token.SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parsePrintStatement() *PrintStatement {
	stmt := &PrintStatement{}

	// Expect '('
	p.nextToken()
	if p.curTok.Type != token.LPAREN {
		return nil
	}

	// Parse the expression inside parentheses
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	// Expect ')'
	p.nextToken()
	if p.curTok.Type != token.RPAREN {
		return nil
	}

	// Optional semicolon
	if p.peekTok.Type == token.SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseIfStatement() *IfStatement {
	stmt := &IfStatement{}

	// move to condition
	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	// expect '{'
	if p.peekTok.Type != token.LBRACE {
		fmt.Println("Error: missing '{' in if")
		return stmt
	}

	p.nextToken() // move to '{'

	stmt.Consequence = p.parseBlockStatement()

	// else and else if
	if p.peekTok.Type == token.ELSE {
		p.nextToken() // ELSE

		// else if
		if p.peekTok.Type == token.IF {
			p.nextToken()
			stmt.Alternative = []Statement{
				p.parseIfStatement(),
			}
			return stmt
		}

		// else
		if p.peekTok.Type != token.LBRACE {
			return stmt
		}

		p.nextToken() // '{'
		stmt.Alternative = p.parseBlockStatement()
	}

	return stmt
}

func (p *Parser) parseBlockStatement() []Statement {
	statements := []Statement{}

	p.nextToken() // move past '{'

	for p.curTok.Type != token.RBRACE && p.curTok.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			statements = append(statements, stmt)
		}
		p.nextToken()
	}

	return statements
}

func (p *Parser) parseExpression(precedence int) Expression {
	left := p.parsePrimary()

	for p.peekTok.Type != token.SEMICOLON && precedence < p.peekPrecedence() {
		p.nextToken() // move to operator

		left = p.parseInfixExpression(left)
	}

	return left
}

func (p *Parser) parseInfixExpression(left Expression) Expression {
	expr := &InfixExpression{
		Left:     left,
		Operator: p.curTok.Literal,
	}

	prec := p.curPrecedence()
	p.nextToken()

	expr.Right = p.parseExpression(prec)
	return expr
}

func (p *Parser) parsePrimary() Expression {
	switch p.curTok.Type {
	case token.BANG:
		operator := p.curTok.Literal
		p.nextToken()
		right := p.parsePrimary()
		if right == nil {
			return nil
		}
		return &PrefixExpression{
			Operator: operator,
			Right:    right,
		}

	case token.INT:
		return &IntLiteral{Value: atoi(p.curTok.Literal)}

	case token.STRING:
		return &StringLiteral{Value: p.curTok.Literal}

	case token.TRUE:
		return &BoolLiteral{Value: true}

	case token.FALSE:
		return &BoolLiteral{Value: false}

	case token.IDENT:
		return &Identifier{Value: p.curTok.Literal}

	case token.LPAREN:
		p.nextToken()
		exp := p.parseExpression(LOWEST)

		if p.peekTok.Type != token.RPAREN {
			panic("expected ')'")
		}

		p.nextToken()
		return &GroupedExpression{Expression: exp}

	default:
		return nil
	}
}
