package parser

import (
	"fmt"
	"strconv"

	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/token"
)

func atoi(a string) int {
	val, _ := strconv.Atoi(a)
	return val
}

func atof(a string) float64 {
	val, _ := strconv.ParseFloat(a, 64)
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
	case token.CONST:
		return p.parseConstStatement()
	case token.FUNC:
		return p.parseFuncStatement()
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
	case token.BREAK:
		return p.parseBreakStatement()
	case token.CONTINUE:
		return p.parseContinueStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.IDENT:
		// arr[idx] = ???
		if p.peekTok.Type == token.LBRACKET {
			return p.parseIndexAssignment()
		}

		// reassignment
		if p.peekTok.Type == token.ASSIGN {
			return p.parseAssignStatement()
		}

		// function call
		expr := p.parseExpression(LOWEST)
		return &ExpressionStatement{NodeBase: NodeBase{Token: p.curTok}, Expression: expr}
	}
	return nil
}

func (p *Parser) parseVarStatement() *VarStatement {
	stmt := &VarStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

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
	stmt.NodeBase = NodeBase{Token: p.curTok}

	p.nextToken() // name
	stmt.Name = p.curTok.Literal

	p.nextToken() // '='
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseArrayLiteral() Expression {
	arr := &ArrayLiteral{}
	arr.NodeBase = NodeBase{Token: p.curTok}

	arr.Elements = []Expression{}

	// move to first element or ]
	p.nextToken()

	if p.curTok.Type == token.RBRACKET {
		return arr // empty array
	}

	arr.Elements = append(arr.Elements, p.parseExpression(LOWEST))

	for p.peekTok.Type == token.COMMA {
		p.nextToken() // ,
		p.nextToken() // next element
		arr.Elements = append(arr.Elements, p.parseExpression(LOWEST))
	}

	if p.peekTok.Type != token.RBRACKET {
		return nil
	}

	p.nextToken() // ]

	return arr
}

func (p *Parser) parseIndexAssignment() *IndexAssignmentStatement {
	stmt := &IndexAssignmentStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

	// left ident
	left := &Identifier{NodeBase: NodeBase{Token: p.curTok}, Value: p.curTok.Literal}

	p.nextToken() // [

	p.nextToken() // index
	idx := p.parseExpression(LOWEST)

	if p.peekTok.Type != token.RBRACKET {
		return nil
	}
	p.nextToken() // ]

	// expect '='
	if p.peekTok.Type != token.ASSIGN {
		return nil
	}
	p.nextToken() // =

	p.nextToken() // value
	val := p.parseExpression(LOWEST)

	stmt.Left = left
	stmt.Index = idx
	stmt.Value = val

	// optional semicolon
	if p.peekTok.Type == token.SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseConstStatement() *ConstStatement {
	stmt := &ConstStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

	// move to ident
	p.nextToken()
	if p.curTok.Type != token.IDENT {
		return nil
	}
	stmt.Name = p.curTok.Literal

	// expect '='
	p.nextToken()
	if p.curTok.Type != token.ASSIGN {
		return nil
	}

	// expression after '='
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	// optional semicolon
	if p.peekTok.Type == token.SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseAssignStatement() *AssignmentStatement {
	stmt := &AssignmentStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

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
	stmt.NodeBase = NodeBase{Token: p.curTok}
	stmt.Name = p.curTok.Literal

	// consume '='
	p.nextToken()
	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)
	return stmt
}

func (p *Parser) parsePrintStatement() *PrintStatement {
	stmt := &PrintStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

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
	stmt.NodeBase = NodeBase{Token: p.curTok}

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
	stmt.NodeBase = NodeBase{Token: p.curTok}

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

func (p *Parser) parseFuncStatement() *FuncStatement {
	stmt := &FuncStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

	// move to func name
	p.nextToken()
	if p.curTok.Type != token.IDENT {
		return nil
	}
	stmt.Name = p.curTok.Literal

	// expect '('
	p.nextToken()
	if p.curTok.Type != token.LPAREN {
		return nil
	}

	// parse params
	stmt.Params = []string{}
	p.nextToken()
	for p.curTok.Type != token.RPAREN {
		if p.curTok.Type == token.IDENT {
			stmt.Params = append(stmt.Params, p.curTok.Literal)
		}

		p.nextToken()
		if p.curTok.Type == token.COMMA {
			p.nextToken()
		}
	}

	// expect '{'
	if p.peekTok.Type != token.LBRACE {
		return nil
	}
	p.nextToken() // move to '{'
	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseFuncCall() Expression {
	call := &FuncCall{NodeBase: NodeBase{Token: p.curTok}, Name: p.curTok.Literal}

	// expect '('
	p.nextToken()
	if p.curTok.Type != token.LPAREN {
		return nil
	}

	// parse args
	call.Args = []Expression{}
	p.nextToken()
	for p.curTok.Type != token.RPAREN {
		call.Args = append(call.Args, p.parseExpression(LOWEST))
		p.nextToken()
		if p.curTok.Type == token.COMMA {
			p.nextToken()
		}
	}

	return call
}

func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

	// move past return
	p.nextToken()

	// if next tok is not semicolon, eof, or }, parse expr
	if p.curTok.Type != token.SEMICOLON && p.curTok.Type != token.EOF && p.curTok.Type != token.RBRACE {
		stmt.Value = p.parseExpression(LOWEST)
	} else {
		stmt.Value = nil // no value provided, so empty return
	}
	// optional semicolon
	if p.peekTok.Type == token.SEMICOLON {
		p.nextToken()
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

func (p *Parser) parseBreakStatement() *BreakStatement {
	stmt := &BreakStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

	// optional semicolon
	if p.peekTok.Type == token.SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseContinueStatement() *ContinueStatement {
	stmt := &ContinueStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

	// optional semicolon
	if p.peekTok.Type == token.SEMICOLON {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseForStatement() *ForStatement {
	stmt := &ForStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

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
	stmt.NodeBase = NodeBase{Token: p.curTok}

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

	for p.peekTok.Type == token.LBRACKET {
		p.nextToken() // [
		p.nextToken() // index expr

		index := p.parseExpression(LOWEST)

		if p.peekTok.Type != token.RBRACKET {
			return nil
		}
		p.nextToken() // ]

		left = &IndexExpression{
			NodeBase: NodeBase{Token: p.curTok},
			Left:     left,
			Index:    index,
		}
	}

	return left
}

func (p *Parser) parseInfixExpression(left Expression) Expression {
	expr := &InfixExpression{
		NodeBase: NodeBase{Token: p.curTok},
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
			NodeBase: NodeBase{Token: p.curTok},
			Operator: operator,
			Right:    right,
		}

	case token.INT:
		return &IntLiteral{NodeBase: NodeBase{Token: p.curTok}, Value: atoi(p.curTok.Literal)}

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

		return &IntCastExpression{NodeBase: NodeBase{Token: p.curTok}, Value: val}

	case token.FLOAT:
		return &FloatLiteral{NodeBase: NodeBase{Token: p.curTok}, Value: atof(p.curTok.Literal)}

	case token.FLOAT_TYPE:
		// expect '('
		p.nextToken()
		if p.curTok.Type != token.LPAREN {
			return nil
		}

		// parse inner expresion
		p.nextToken()
		val := p.parseExpression(LOWEST)

		// expect ')'
		if p.peekTok.Type != token.RPAREN {
			panic("expected ) after float(")
		}
		p.nextToken()

		return &FloatCastExpression{NodeBase: NodeBase{Token: p.curTok}, Value: val}

	case token.STRING:
		return &StringLiteral{NodeBase: NodeBase{Token: p.curTok}, Value: p.curTok.Literal}

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

		return &StringCastExpression{NodeBase: NodeBase{Token: p.curTok}, Value: val}

	case token.TRUE:
		return &BoolLiteral{NodeBase: NodeBase{Token: p.curTok}, Value: true}

	case token.FALSE:
		return &BoolLiteral{NodeBase: NodeBase{Token: p.curTok}, Value: false}

	case token.IDENT:
		if p.peekTok.Type == token.LPAREN {
			return p.parseFuncCall()
		}
		return &Identifier{NodeBase: NodeBase{Token: p.curTok}, Value: p.curTok.Literal}

	case token.LPAREN:
		p.nextToken()
		exp := p.parseExpression(LOWEST)

		if p.peekTok.Type != token.RPAREN {
			panic("expected ')'")
		}

		p.nextToken()
		return &GroupedExpression{NodeBase: NodeBase{Token: p.curTok}, Expression: exp}

	case token.LBRACKET:
		return p.parseArrayLiteral()

	default:
		return nil
	}
}
