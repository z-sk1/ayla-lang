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
	case token.SCANLN:
		return p.parseScanlnStatement()
	case token.IF:
		return p.parseIfStatement()
	case token.FOR:
		return p.parseForStatement()
	case token.WHILE:
		return p.parseWhileStatement()
	case token.IDENT:
		if p.peekTok.Type == token.ASSIGN {
			return p.parseAssignStatement()
		}
	}
	return nil
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

func (p *Parser) parseVarStatementNoSemicolon() *VarStatement {
	stmt := &VarStatement{}
	p.nextToken() // name
	stmt.Name = p.curTok.Literal

	p.nextToken() // '='
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseAssignStatement() *AssignmentStatement {
	stmt := &AssignmentStatement{}

	// current token is IDENT
	stmt.Name = p.curTok.Literal

	// move to =
	p.nextToken()
	if p.curTok.Type != token.ASSIGN {
		return nil
	}

	// expression after =
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	// optional semicolon
	if p.peekTok.Type == token.SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseAssignmentNoSemicolon() *AssignmentStatement {
	stmt := &AssignmentStatement{}
	stmt.Name = p.curTok.Literal

	// consume '='
	p.nextToken()
	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)
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

func (p *Parser) parseScanlnStatement() *ScanlnStatement {
	stmt := &ScanlnStatement{}

	// expect '('
	p.nextToken()
	if p.curTok.Type != token.LPAREN {
		return nil
	}

	// expect the ident and store it
	p.nextToken()
	if p.curTok.Type != token.IDENT {
		return nil
	}
	stmt.Name = p.curTok.Literal

	// expect ')'
	p.nextToken()
	if p.curTok.Type != token.RPAREN {
		return nil
	}

	// optional semicolon
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

func (p *Parser) parseForInit() Statement {
	if p.curTok.Type == token.VAR {
		return p.parseVarStatementNoSemicolon()
	}
	return p.parseAssignmentNoSemicolon()
}

func (p *Parser) parseForPost() Statement {
	return p.parseAssignmentNoSemicolon()
}

func (p *Parser) parseForStatement() *ForStatement {
	stmt := &ForStatement{}

	// init
	p.nextToken() // move to VAR or IDENT
	stmt.Init = p.parseForInit()
	if stmt.Init == nil {
		return nil
	}

	// expect ';'
	if p.peekTok.Type != token.SEMICOLON {
		return nil
	}
	p.nextToken() // consume ';'

	// condition
	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)
	if stmt.Condition == nil {
		return nil
	}

	// expect ';'
	if p.peekTok.Type != token.SEMICOLON {
		return nil
	}
	p.nextToken() // consume ';'

	// post
	p.nextToken()
	stmt.Post = p.parseForPost()
	if stmt.Post == nil {
		return nil
	}

	// expect '{'
	if p.peekTok.Type != token.LBRACE {
		return nil
	}
	p.nextToken() // move to '{'

	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseWhileStatement() *WhileStatement {
	stmt := &WhileStatement{}

	// move to condition
	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)
	if stmt.Condition == nil {
		return nil
	}

	// expect '{'
	if p.peekTok.Type != token.LBRACE {
		return nil
	}
	p.nextToken() // move to '{'

	stmt.Body = p.parseBlockStatement()
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

	case token.INT_TYPE:
		// expect '('
		p.nextToken()
		if p.curTok.Type != token.LPAREN {
			return nil
		}

		// parse inner expression
		p.nextToken()
		val := p.parseExpression(LOWEST)

		// expect ')'
		if p.peekTok.Type != token.RPAREN {
			panic("expected ) after int(")
		}
		p.nextToken()

		return &IntCastExpression{Value: val}

	case token.STRING:
		return &StringLiteral{Value: p.curTok.Literal}

	case token.STRING_TYPE:
		// expect '('
		p.nextToken()
		if p.curTok.Type != token.LPAREN {
			return nil
		}

		// parse inner expression
		p.nextToken()
		val := p.parseExpression(LOWEST)

		// expect ')'
		if p.peekTok.Type != token.RPAREN {
			panic("expected ) after string(")
		}
		p.nextToken()

		return &StringCastExpression{Value: val}

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
