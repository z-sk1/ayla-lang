package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/token"
)

type Parser struct {
	NodeBase
	l       *lexer.Lexer
	curTok  token.Token // current
	peekTok token.Token // lookahead 1
	peekBuf []token.Token

	errors []error
	types  map[string]bool
}

type ParseError struct {
	Message string
	Line    int
	Column  int
	Token   token.Token
}

func (e ParseError) Error() string {
	if e.Token.Literal == "" {
		e.Token.Literal = "nothing"
	}

	return fmt.Sprintf("parse error at %d:%d: %s (got %s)", e.Line, e.Column, e.Message, e.Token.Literal)
}

func (p *Parser) Errors() []error {
	return p.errors
}

func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, &ParseError{Message: msg, Line: p.curTok.Line, Column: p.curTok.Column, Token: p.curTok})
}

func atoi(a string) int {
	val, _ := strconv.Atoi(a)
	return val
}

func atof(a string) float64 {
	val, _ := strconv.ParseFloat(a, 64)
	return val
}

func (p *Parser) parseIdentList() []*Identifier {
	idents := []*Identifier{}

	for {
		if p.curTok.Type != token.IDENT {
			return idents
		}

		ident := &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		}
		idents = append(idents, ident)

		if p.peekTok.Type != token.COMMA {
			break
		}

		p.nextToken() // ,
		p.nextToken() // next ident
	}

	return idents
}

func (p *Parser) isTypeToken(t token.TokenType) bool {
	switch t {
	case
		token.INT_TYPE,
		token.STRING_TYPE,
		token.BOOL_TYPE,
		token.FLOAT_TYPE,
		token.ARR_TYPE:
		return true
	default:
		return false
	}
}

func (p *Parser) isTypeName(name string) bool {
	_, ok := p.types[name]
	return ok
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:     l,
		types: make(map[string]bool),
	}

	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curTok = p.peekTok

	if len(p.peekBuf) > 0 {
		p.peekTok = p.peekBuf[0]
		p.peekBuf = p.peekBuf[1:]
	} else {
		p.peekTok = p.l.NextToken()
	}
}

func (p *Parser) peekN(n int) token.Token {
	if n == 0 {
		return p.peekTok
	}

	idx := n - 1
	for len(p.peekBuf) <= idx {
		p.peekBuf = append(p.peekBuf, p.l.NextToken())
	}
	return p.peekBuf[idx]
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

func (p *Parser) peekUntilAssign() token.TokenType {
	depth := 0
	for i := 0; ; i++ {
		tok := p.peekN(i)

		switch tok.Type {
		case token.LPAREN, token.LBRACKET:
			depth++
		case token.RPAREN, token.RBRACKET:
			depth--
		case token.WALRUS, token.ASSIGN:
			if depth == 0 {
				return tok.Type
			}
		case token.NEWLINE, token.EOF:
			return token.ILLEGAL
		}
	}
}

func (p *Parser) consumeTerminators() {
	for {
		switch p.curTok.Type {
		case token.NEWLINE, token.SEMICOLON:
			p.nextToken()
		default:
			return
		}
	}
}

func (p *Parser) ParseProgram() []Statement {
	var statements []Statement
	for p.curTok.Type != token.EOF {
		if p.curTok.Type == token.NEWLINE {
			p.nextToken()
		}

		stmt := p.parseStatement()
		if stmt != nil {
			statements = append(statements, stmt)
		}
		p.nextToken()

		p.consumeTerminators()
	}

	return statements
}

func (p *Parser) parseStatement() Statement {
	switch p.curTok.Type {
	case token.VAR:
		if p.peekTok.Type == token.IDENT && p.peekN(1).Type == token.COMMA {
			return p.parseMultiVarStatement()
		}

		return p.parseVarStatement()

	case token.CONST:
		if p.peekTok.Type == token.IDENT && p.peekN(1).Type == token.COMMA {
			return p.parseMultiConstStatement()
		}

		return p.parseConstStatement()
	case token.TYPE:
		return p.parseTypeStatement()
	case token.SWITCH:
		return p.parseSwitchStatement()
	case token.FUNC:
		return p.parseFuncStatement()
	case token.SPAWN:
		return p.parseSpawnStatement()
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
		// multi assignment: a, b = ...
		if p.peekTok.Type == token.COMMA {
			if p.peekUntilAssign() == token.WALRUS {
				return p.parseMultiVarStatementNoKeyword()
			} else if p.peekUntilAssign() == token.ASSIGN {
				return p.parseMultiAssignStatement()
			}
		}

		// arr[idx] = ...
		// "string"[idx]
		if p.peekTok.Type == token.LBRACKET {
			return p.parseIndexAssignment()
		}

		// reassignment
		if p.peekTok.Type == token.ASSIGN {
			return p.parseAssignStatement()
		}

		// single inferred
		if p.peekTok.Type == token.WALRUS {
			return p.parseVarStatementNoKeyword()
		}

		// member assignment
		if p.peekTok.Type == token.DOT {
			return p.parseMemberAssignment()
		}

		// function call
		expr := p.parseExpression(LOWEST)
		return &ExpressionStatement{NodeBase: NodeBase{Token: p.curTok}, Expression: expr}
	}
	return nil
}

func (p *Parser) parseVarStatement() *VarStatement {
	stmt := &VarStatement{
		NodeBase: NodeBase{Token: p.curTok}, // 'egg'
	}

	// egg -> name
	p.nextToken()
	if p.curTok.Type != token.IDENT {
		p.addError("expected identifier after 'egg'")
		return nil
	}

	stmt.Name = &Identifier{
		NodeBase: NodeBase{Token: p.curTok},
		Value:    p.curTok.Literal,
	}

	// optional type
	if p.isTypeToken(p.peekTok.Type) || p.peekTok.Type == token.IDENT {
		p.nextToken()
		stmt.Type = &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		}
	}

	// optional assignment
	if p.peekTok.Type == token.ASSIGN {
		p.nextToken()
		p.nextToken()
		stmt.Value = p.parseExpression(LOWEST)
	}

	return stmt
}

func (p *Parser) parseVarStatementNoKeyword() *VarStatementNoKeyword {
	stmt := &VarStatementNoKeyword{
		NodeBase: NodeBase{Token: p.curTok}, // ident
	}

	stmt.Name = &Identifier{
		NodeBase: NodeBase{Token: p.curTok},
		Value:    p.curTok.Literal,
	}

	p.nextToken() // :=
	p.nextToken() // expr
	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseMultiVarStatement() *MultiVarStatement {
	stmt := &MultiVarStatement{
		NodeBase: NodeBase{Token: p.curTok}, // egg
	}

	p.nextToken() // ident
	stmt.Names = p.parseIdentList()

	// optional type
	if p.isTypeToken(p.peekTok.Type) {
		p.nextToken()
		stmt.Type = &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		}
	}

	// optional assignment
	if p.peekTok.Type == token.ASSIGN {
		p.nextToken() // move to '='
		p.nextToken() // move to expr start

		values := p.parseTupleList()

		if len(values) == 1 {
			stmt.Value = values[0]
		} else {
			stmt.Value = &TupleLiteral{
				NodeBase: NodeBase{Token: p.curTok},
				Values:   values,
			}
		}
	}

	return stmt
}

func (p *Parser) parseMultiVarStatementNoKeyword() *MultiVarStatementNoKeyword {
	stmt := &MultiVarStatementNoKeyword{
		NodeBase: NodeBase{Token: p.curTok}, // idents
	}

	stmt.Names = p.parseIdentList()

	p.nextToken() // :=
	p.nextToken() // move to expr start

	values := p.parseTupleList()

	if len(values) == 1 {
		stmt.Value = values[0]
	} else {
		stmt.Value = &TupleLiteral{
			NodeBase: NodeBase{Token: p.curTok},
			Values:   values,
		}
	}

	return stmt
}

func (p *Parser) parseTupleList() []Expression {
	values := []Expression{}

	values = append(values, p.parseExpression(LOWEST))

	for p.peekTok.Type == token.COMMA {
		p.nextToken() // ,
		p.nextToken() // next expr
		values = append(values, p.parseExpression(LOWEST))
	}

	return values
}

func (p *Parser) parseConstStatement() *ConstStatement {
	stmt := &ConstStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

	// rock -> name
	p.nextToken()
	if p.curTok.Type != token.IDENT {
		p.addError("expected identifier after 'egg'")
		return nil
	}
	stmt.Name = &Identifier{
		NodeBase: NodeBase{Token: p.curTok},
		Value:    p.curTok.Literal,
	}

	// type
	if p.isTypeToken(p.peekTok.Type) {
		p.nextToken()
		stmt.Type = &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		}
	}

	// assignment
	if p.peekTok.Type == token.ASSIGN {
		p.nextToken()
		p.nextToken()
		stmt.Value = p.parseExpression(LOWEST)
	}

	return stmt
}

func (p *Parser) parseMultiConstStatement() *MultiConstStatement {
	stmt := &MultiConstStatement{
		NodeBase: NodeBase{Token: p.curTok},
	}

	p.nextToken() // ident
	stmt.Names = p.parseIdentList()

	// optional type
	if p.isTypeToken(p.peekTok.Type) {
		p.nextToken()
		stmt.Type = &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		}
	}

	// optional assignment
	if p.peekTok.Type == token.ASSIGN {
		p.nextToken() // move to '='
		p.nextToken() // move to expr start

		values := p.parseTupleList()

		if len(values) == 1 {
			stmt.Value = values[0]
		} else {
			stmt.Value = &TupleLiteral{
				NodeBase: NodeBase{Token: p.curTok},
				Values:   values,
			}
		}
	}

	return stmt
}

func (p *Parser) parseAssignStatement() *AssignmentStatement {
	stmt := &AssignmentStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

	// current token is IDENT
	stmt.Name = &Identifier{
		NodeBase: NodeBase{Token: p.curTok},
		Value:    p.curTok.Literal,
	}

	// move to =
	p.nextToken()
	if p.curTok.Type != token.ASSIGN {
		return nil
	}

	// expression after =
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)
	if stmt.Value == nil {
		return nil
	}

	return stmt
}

func (p *Parser) parseMultiAssignStatement() *MultiAssignmentStatement {
	stmt := &MultiAssignmentStatement{
		NodeBase: NodeBase{Token: p.curTok},
	}

	stmt.Names = p.parseIdentList()

	// expect '='
	if p.peekTok.Type != token.ASSIGN {
		p.addError("expected '=' after identifiers")
		return nil
	}

	p.nextToken() // move to '='
	p.nextToken() // move to expr

	values := p.parseTupleList()

	if len(values) == 1 {
		stmt.Value = values[0]
	} else {
		stmt.Value = &TupleLiteral{
			NodeBase: NodeBase{Token: p.curTok},
			Values:   values,
		}
	}

	return stmt
}

func (p *Parser) parseTypeStatement() *TypeStatement {
	stmt := &TypeStatement{
		NodeBase: NodeBase{Token: p.curTok},
	}

	p.nextToken()
	if p.curTok.Type != token.IDENT {
		p.addError("expected identifier after 'type'")
		return nil
	}
	stmt.Name = &Identifier{
		NodeBase: NodeBase{Token: p.curTok},
		Value:    p.curTok.Literal,
	}

	p.types[stmt.Name.Value] = true

	p.nextToken()

	if p.curTok.Type == token.ASSIGN {
		stmt.Alias = true
		p.nextToken()
	}

	stmt.Type = p.parseType()

	return stmt
}

func (p *Parser) parseType() TypeNode {
	switch p.curTok.Type {
	case token.INT_TYPE,
		token.STRING_TYPE,
		token.FLOAT_TYPE,
		token.BOOL_TYPE,
		token.ARR_TYPE,
		token.IDENT:
		return &IdentType{
			NodeBase: NodeBase{Token: p.curTok},
			Name:     p.curTok.Literal,
		}
	case token.STRUCT:
		if p.peekTok.Type != token.LBRACE {
			p.addError("expected '{' after identifier")
			return nil
		}
		p.nextToken()

		fields := []*StructField{}

		// move to first field or }
		p.nextToken()
		p.consumeTerminators()

		for p.curTok.Type != token.RBRACE {
			if p.curTok.Type != token.IDENT {
				p.addError("expected field name inside struct type")
				return nil
			}
			fieldName := &Identifier{
				NodeBase: NodeBase{Token: p.curTok},
				Value:    p.curTok.Literal,
			}
			// type identifier
			p.nextToken()
			if !(p.isTypeToken(p.curTok.Type)) && p.curTok.Type != token.IDENT {
				p.addError(fmt.Sprintf("expected type name after field name '%s'", fieldName.Value))
				return nil
			}
			fieldType := p.parseType()
			fields = append(fields, &StructField{Name: fieldName, Type: fieldType})
			p.nextToken()
			p.consumeTerminators()
		}

		return &StructType{
			NodeBase: NodeBase{Token: p.curTok},
			Fields:   fields,
		}
	default:
		p.addError("unknown type")
		return nil
	}
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
		p.addError("expected ']' to close array")
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
		p.addError("expected ']' to close index")
		return nil
	}
	p.nextToken() // ]

	// expect '='
	if p.peekTok.Type != token.ASSIGN {
		stmt.Left = left
		stmt.Index = idx
		stmt.Value = nil
		return stmt
	}
	p.nextToken() // =

	p.nextToken() // value
	val := p.parseExpression(LOWEST)

	stmt.Left = left
	stmt.Index = idx
	stmt.Value = val

	return stmt
}

func (p *Parser) parseStructLiteral(left Expression) Expression {
	ident, ok := left.(*Identifier)
	if !ok {
		p.addError("struct literals must start with identifiers")
		return nil
	}

	lit := &StructLiteral{
		NodeBase: NodeBase{Token: p.curTok},
		TypeName: ident,
		Fields:   make(map[string]Expression),
	}

	p.nextToken() // move to '{'
	p.nextToken() // move inside struct
	p.consumeTerminators()

	// empty struct {}
	if p.curTok.Type == token.RBRACE {
		return lit
	}

	for {
		if p.curTok.Type != token.IDENT {
			p.addError("expected field names in struct literal")
			return nil
		}

		fieldName := p.curTok.Literal

		if p.peekTok.Type != token.COLON {
			p.addError("expected ':' after field name")
			return nil
		}

		p.nextToken() // :
		p.nextToken() // value
		lit.Fields[fieldName] = p.parseExpression(LOWEST)
		p.nextToken()

		p.consumeTerminators()

		if p.curTok.Type == token.COMMA {
			p.nextToken()
			p.consumeTerminators()
			if p.curTok.Type == token.RBRACE {
				break
			}
			continue
		}

		if p.curTok.Type == token.RBRACE {
			break
		}

		p.addError("expected ',' or '}' after struct field")
		return nil
	}

	return lit
}

func (p *Parser) parseAnonymousStructLiteral() Expression {
	lit := &AnonymousStructLiteral{
		NodeBase: NodeBase{Token: p.curTok},
		Fields:   make(map[string]Expression),
	}

	p.nextToken() // move to '{'
	p.nextToken() // move inside
	p.consumeTerminators()

	if p.curTok.Type == token.RBRACE {
		return lit
	}

	for {
		if p.curTok.Type != token.IDENT {
			p.addError("expected field names in anonymous struct literal")
			return nil
		}

		fieldName := p.curTok.Literal

		if p.peekTok.Type != token.COLON {
			p.addError("expected ':' after field name")
			return nil
		}

		p.nextToken() // :
		p.nextToken() // value
		lit.Fields[fieldName] = p.parseExpression(LOWEST)
		p.nextToken()

		p.consumeTerminators()

		if p.curTok.Type == token.COMMA {
			p.nextToken()
			p.consumeTerminators()
			if p.curTok.Type == token.RBRACE {
				break
			}
			continue
		}

		if p.curTok.Type == token.RBRACE {
			break
		}

		p.addError("expected ',' or '}' after anonymous struct field")
		return nil
	}

	return lit
}

func (p *Parser) parseMemberAssignment() *MemberAssignmentStatement {
	// p
	obj := &Identifier{
		NodeBase: NodeBase{Token: p.curTok},
		Value:    p.curTok.Literal,
	}

	p.nextToken() // .
	p.nextToken() // field

	if p.curTok.Type != token.IDENT {
		p.addError("expected field name after '.'")
		return nil
	}

	field := &Identifier{
		NodeBase: NodeBase{Token: p.curTok},
		Value:    p.curTok.Literal,
	}

	if p.peekTok.Type != token.ASSIGN {
		p.addError("expected '=' after member")
		return nil
	}

	p.nextToken() // =
	p.nextToken() // value

	val := p.parseExpression(LOWEST)

	return &MemberAssignmentStatement{
		NodeBase: NodeBase{Token: p.curTok},
		Object:   obj,
		Field:    field,
		Value:    val,
	}
}

func (p *Parser) parseIfStatement() *IfStatement {
	stmt := &IfStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

	// move to condition
	p.nextToken()

	if p.curTok.Type == token.LBRACE {
		p.addError("missing condition in if")
		return nil
	}

	stmt.Condition = p.parseExpression(LOWEST)

	// expect '{'
	if p.peekTok.Type != token.LBRACE {
		p.addError("expected '{' after conditional")
		return nil
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

func (p *Parser) parseSpawnStatement() *SpawnStatement {
	stmt := &SpawnStatement{
		NodeBase: NodeBase{Token: p.curTok},
	}

	p.nextToken() // {
	if p.curTok.Type != token.LBRACE {
		p.addError("expected '{' after spawn")
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseSwitchStatement() *SwitchStatement {
	stmt := &SwitchStatement{
		NodeBase: NodeBase{Token: p.curTok},
	}

	// switch <expr>
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	if p.curTok.Type != token.LBRACE {
		p.nextToken()
	}

	if p.curTok.Type != token.LBRACE {
		p.addError("expected '{' after switch expression")
		return nil
	}

	p.nextToken()

	stmt.Cases = []*CaseClause{}

	for p.curTok.Type != token.RBRACE && p.curTok.Type != token.EOF {
		switch p.curTok.Type {

		case token.CASE:
			clause := p.parseCaseClause()
			if clause == nil {
				return nil
			}
			stmt.Cases = append(stmt.Cases, clause)
			p.nextToken()

		case token.DEFAULT:
			if stmt.Default != nil {
				p.addError("multiple default clauses in switch")
				return nil
			}
			stmt.Default = p.parseDefaultClause()
			p.nextToken()

		default:
			p.addError("expected 'case' or 'default' in switch statement")
			return nil
		}
	}

	return stmt
}

func (p *Parser) parseCaseClause() *CaseClause {
	clause := &CaseClause{
		NodeBase: NodeBase{Token: p.curTok},
	}

	// case <expr>
	p.nextToken()
	clause.Expr = p.parseExpression(LOWEST)

	if p.peekTok.Type != token.LBRACE {
		p.addError("expected '{' after case expression")
		return nil
	}
	p.nextToken() // {
	p.nextToken() // first stmt

	clause.Body = []Statement{}

	for p.curTok.Type != token.RBRACE {
		stmt := p.parseStatement()
		if stmt != nil {
			clause.Body = append(clause.Body, stmt)
		}
		p.nextToken()
	}

	return clause
}

func (p *Parser) parseDefaultClause() *DefaultClause {
	clause := &DefaultClause{
		NodeBase: NodeBase{Token: p.curTok},
	}

	// default {
	if p.peekTok.Type != token.LBRACE {
		p.addError("expected '{' after default")
		return nil
	}
	p.nextToken() // {
	p.nextToken() // first stmt

	clause.Body = []Statement{}

	for p.curTok.Type != token.RBRACE {
		stmt := p.parseStatement()
		if stmt != nil {
			clause.Body = append(clause.Body, stmt)
		}
		p.nextToken()
	}

	return clause
}

func (p *Parser) parseFuncStatement() *FuncStatement {
	stmt := &FuncStatement{
		NodeBase: NodeBase{Token: p.curTok},
	}

	// fun <name>
	p.nextToken()
	if p.curTok.Type != token.IDENT {
		p.addError("expected identifier after 'fun'")
		return nil
	}
	stmt.Name = &Identifier{
		NodeBase: NodeBase{Token: p.curTok},
		Value:    p.curTok.Literal,
	}

	// (
	p.nextToken()
	if p.curTok.Type != token.LPAREN {
		p.addError("expected '(' after function name")
		return nil
	}

	stmt.Params = []*ParametersClause{}
	p.nextToken()

	for p.curTok.Type != token.RPAREN {
		if p.curTok.Type != token.IDENT {
			p.addError("expected parameter name")
			return nil
		}

		paramName := &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		}
		var paramType *Identifier = nil

		// optional type
		if p.peekTok.Type == token.INT_TYPE ||
			p.peekTok.Type == token.FLOAT_TYPE ||
			p.peekTok.Type == token.STRING_TYPE ||
			p.peekTok.Type == token.BOOL_TYPE ||
			p.isTypeName(p.peekTok.Literal) {

			p.nextToken()
			paramType = &Identifier{
				NodeBase: NodeBase{Token: p.curTok},
				Value:    p.curTok.Literal,
			}
		}

		stmt.Params = append(stmt.Params, &ParametersClause{
			NodeBase: NodeBase{Token: p.curTok},
			Name:     paramName,
			Type:     paramType,
		})

		p.nextToken()
		if p.curTok.Type == token.COMMA {
			p.nextToken()
		}
	}

	// consume ')'
	p.nextToken()

	stmt.ReturnTypes = []*Identifier{}

	if p.curTok.Type == token.LPAREN {
		p.nextToken()

		for p.curTok.Type != token.RPAREN {
			if p.curTok.Type != token.INT_TYPE &&
				p.curTok.Type != token.FLOAT_TYPE &&
				p.curTok.Type != token.STRING_TYPE &&
				p.curTok.Type != token.BOOL_TYPE {

				p.addError("expected return type")
				return nil
			}

			stmt.ReturnTypes = append(stmt.ReturnTypes, &Identifier{
				NodeBase: NodeBase{Token: p.curTok},
				Value:    p.curTok.Literal,
			})

			p.nextToken()
			if p.curTok.Type == token.COMMA {
				p.nextToken()
			}
		}

		// consume ')'
		p.nextToken()
	}

	if p.curTok.Type != token.LBRACE {
		p.addError("expected '{' before function body")
		return nil
	}

	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseFuncCall() Expression {
	call := &FuncCall{
		NodeBase: NodeBase{Token: p.curTok},
		Name: &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		},
	}

	// expect '('
	p.nextToken()
	if p.curTok.Type != token.LPAREN {
		p.addError("expected '(' after function name")
		return nil
	}

	// parse args
	call.Args = p.parseExpressionList(token.RPAREN)

	return call
}

func (p *Parser) parseExpressionList(end token.TokenType) []Expression {
	list := []Expression{}

	if p.peekTok.Type == end {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTok.Type == token.COMMA {
		p.nextToken() // skip comma
		p.nextToken() // next expr
		list = append(list, p.parseExpression(LOWEST))
	}

	if p.peekTok.Type != end {
		p.addError(fmt.Sprintf("expected '%s'", string(end)))
		return nil
	}

	p.nextToken()
	return list
}

func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

	// move past 'return'
	p.nextToken()

	stmt.Values = []Expression{}

	// empty return: return;
	if p.curTok.Type == token.SEMICOLON ||
		p.curTok.Type == token.RBRACE ||
		p.curTok.Type == token.EOF {
		return stmt
	}

	// first expression
	stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))

	// additional expressions: , expr
	for p.peekTok.Type == token.COMMA {
		p.nextToken() // move to comma
		p.nextToken() // move to next expr
		stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))
	}

	return stmt
}

func (p *Parser) parseForInit() Statement {
	if p.curTok.Type == token.VAR {
		return p.parseVarStatement()
	}
	return p.parseAssignStatement()
}

func (p *Parser) parseForPost() Statement {
	return p.parseAssignStatement()
}

func (p *Parser) parseBreakStatement() *BreakStatement {
	stmt := &BreakStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

	return stmt
}

func (p *Parser) parseContinueStatement() *ContinueStatement {
	stmt := &ContinueStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

	return stmt
}

func (p *Parser) parseForStatement() *ForStatement {
	stmt := &ForStatement{
		NodeBase: NodeBase{Token: p.curTok}, // 'four'
	}

	p.nextToken() // first token of init
	stmt.Init = p.parseForInit()
	if stmt.Init == nil {
		p.addError("expected for init")
		return nil
	}

	if p.peekTok.Type != token.SEMICOLON {
		p.addError("expected ';' after for init")
		return nil
	}
	p.nextToken() // move to ';'
	p.nextToken() // move past ';'

	stmt.Condition = p.parseExpression(LOWEST)
	if stmt.Condition == nil {
		p.addError("expected for condition")
		return nil
	}

	if p.peekTok.Type != token.SEMICOLON {
		if p.curTok.Type != token.SEMICOLON {
			p.addError("expected ';' after for condition")
			return nil
		}
	}

	if p.curTok.Type != token.SEMICOLON {
		p.nextToken() // move to ';'
	}
	p.nextToken() // move past ';'

	stmt.Post = p.parseForPost()
	if stmt.Post == nil {
		p.addError("expected for post statement")
		return nil
	}

	if p.peekTok.Type != token.LBRACE {
		p.addError("expected '{' after for post")
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
		p.addError("expected condition after 'why'")
		return nil
	}

	// expect '{'
	if p.peekTok.Type != token.LBRACE {
		p.addError("expected '{' after condition")
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

func (p *Parser) parseIndexExpression(left Expression) Expression {
	exp := &IndexExpression{
		NodeBase: NodeBase{Token: p.curTok}, // '['
		Left:     left,
	}

	p.nextToken() // move to index expression
	exp.Index = p.parseExpression(LOWEST)

	if p.peekTok.Type != token.RBRACKET {
		p.addError("expected ']'")
		return nil
	}

	p.nextToken() // consume ']'

	return exp
}

func (p *Parser) parseExpression(precedence int) Expression {
	left := p.parsePrimary()

	for {
		if p.peekTok.Type == token.LBRACKET {
			p.nextToken() // '['
			left = p.parseIndexExpression(left)
			continue
		}

		if p.peekTok.Type == token.DOT {
			p.nextToken()
			p.nextToken()

			if p.curTok.Type != token.IDENT {
				p.addError("expected property name identifier after '.'")
				return nil
			}

			left = &MemberExpression{
				NodeBase: NodeBase{Token: p.curTok},
				Left:     left,
				Field: &Identifier{
					NodeBase: NodeBase{Token: p.curTok},
					Value:    p.curTok.Literal,
				},
			}
			continue
		}

		if p.peekTok.Type == token.SEMICOLON ||
			p.peekTok.Type == token.RPAREN ||
			precedence >= p.peekPrecedence() {
			break
		}

		p.nextToken()
		left = p.parseInfixExpression(left)
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

func (p *Parser) parseStringLiteral() Expression {
	raw := p.curTok.Literal

	if !strings.Contains(raw, "${") {
		return &StringLiteral{NodeBase: NodeBase{Token: p.curTok}, Value: raw}
	}

	parts := []Expression{}
	i := 0

	for i < len(raw) {
		if raw[i] == '$' && i+1 < len(raw) && raw[i+1] == '{' {
			i += 2 // skip ${
			start := i
			depth := 1

			for i < len(raw) && depth > 0 {
				switch raw[i] {
				case '{':
					depth++
				case '}':
					depth--
				}
				i++
			}

			exprSrc := raw[start : i-1]

			expr := p.parseExpressionFromString(exprSrc)
			parts = append(parts, expr)
		} else {
			start := i
			for i < len(raw) && !(raw[i] == '$' && i+1 < len(raw) && raw[i+1] == '{') {
				i++
			}

			parts = append(parts, &StringLiteral{Value: raw[start:i]})
		}
	}

	return &InterpolatedString{Parts: parts}
}

func (p *Parser) parseExpressionFromString(src string) Expression {
	l := lexer.New(src)
	subParser := New(l)
	return subParser.parseExpression(LOWEST)
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

	case token.MINUS:
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
		if p.peekTok.Type == token.LPAREN {
			return p.parseFuncCall()
		}
		return nil

	case token.FLOAT:
		return &FloatLiteral{NodeBase: NodeBase{Token: p.curTok}, Value: atof(p.curTok.Literal)}

	case token.FLOAT_TYPE:
		if p.peekTok.Type == token.LPAREN {
			return p.parseFuncCall()
		}
		return nil

	case token.STRING:
		return p.parseStringLiteral()

	case token.STRING_TYPE:
		if p.peekTok.Type == token.LPAREN {
			return p.parseFuncCall()
		}
		return nil

	case token.TRUE:
		return &BoolLiteral{NodeBase: NodeBase{Token: p.curTok}, Value: true}

	case token.FALSE:
		return &BoolLiteral{NodeBase: NodeBase{Token: p.curTok}, Value: false}

	case token.NIL:
		return &NilLiteral{NodeBase: NodeBase{Token: p.curTok}}

	case token.BOOL_TYPE:
		if p.peekTok.Type == token.LPAREN {
			return p.parseFuncCall()
		}
		return nil

	case token.STRUCT:
		return p.parseAnonymousStructLiteral()

	case token.IDENT:
		ident := &Identifier{NodeBase: NodeBase{Token: p.curTok}, Value: p.curTok.Literal}

		if p.peekTok.Type == token.LPAREN {
			return p.parseFuncCall()
		}

		if p.peekTok.Type == token.LBRACE && p.isTypeName(ident.Value) {
			return p.parseStructLiteral(ident)
		}

		return ident

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

	case token.LBRACE:
		return p.parseAnonymousStructLiteral()

	default:
		return nil
	}
}
