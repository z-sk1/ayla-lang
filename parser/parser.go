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
}

type ModuleMeta struct {
	Types map[string]struct{}
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

	return fmt.Sprintf("syntax error at %d:%d: %s (got %s)", e.Line, e.Column, e.Message, e.Token.Literal)
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

func (p *Parser) parseIdentList() []Expression {
	idents := []Expression{}

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

func (p *Parser) parseIdentPtrList() []*Identifier {
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

func (p *Parser) parseEnumVariants() []*Identifier {
	var variants []*Identifier

	for {
		p.consumeTerminators()

		if p.curTok.Type == token.RBRACE {
			return variants
		}

		if p.curTok.Type != token.IDENT {
			p.addError("expected identifier in enum body")
			return nil
		}

		variants = append(variants, &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		})

		p.nextToken()
	}
}

func (p *Parser) isTypeToken(t token.TokenType) bool {
	switch t {
	case
		token.INT_TYPE,
		token.STRING_TYPE,
		token.BOOL_TYPE,
		token.FLOAT_TYPE,
		token.ANY_TYPE,
		token.ERROR_TYPE,
		token.LBRACKET,
		token.FUNC,
		token.IDENT,
		token.STRUCT,
		token.INTERFACE,
		token.MAP:
		return true
	default:
		return false
	}
}

func (p *Parser) isAssignToken(t token.TokenType) bool {
	switch t {
	case
		token.ASSIGN,
		token.PLUS_ASSIGN,
		token.SUB_ASSIGN,
		token.MUL_ASSIGN,
		token.SLASH_ASSIGN,
		token.MOD_ASSIGN,
		token.AND_ASSIGN,
		token.OR_ASSIGN,
		token.XOR_ASSIGN,
		token.SHL_ASSIGN,
		token.SHR_ASSIGN:
		return true
	default:
		return false
	}
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l: l,
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
		case token.NEWLINE:
			p.nextToken()
		default:
			return
		}
	}
}

func (p *Parser) isType() bool {
	return p.isTypeToken(p.peekTok.Type) ||
		(p.peekTok.Type == token.IDENT && p.peekN(1).Type == token.DOT) ||
		(p.peekTok.Type == token.MUL && p.isPointerType())
}

func (p *Parser) isPointerType() bool {
	i := 1

	for p.peekN(i).Type == token.MUL {
		i++
	}

	tok := p.peekN(i)

	return p.isTypeToken(tok.Type) ||
		(tok.Type == token.IDENT && p.peekN(i+1).Type == token.DOT)
}

func (p *Parser) isPointerTypeM1() bool {
	i := 0

	for p.peekN(i).Type == token.MUL {
		i++
	}

	tok := p.peekN(i)

	return p.isTypeToken(tok.Type) ||
		(tok.Type == token.IDENT && p.peekN(i+1).Type == token.DOT)
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
		if p.peekTok.Type == token.LPAREN {
			return p.parseVarStatementBlock()
		}

		if p.peekTok.Type == token.IDENT && p.peekN(1).Type == token.COMMA {
			return p.parseMultiVarStatement()
		}

		return p.parseVarStatement()

	case token.CONST:
		if p.peekTok.Type == token.LPAREN {
			return p.parseConstStatementBlock()
		}

		if p.peekTok.Type == token.IDENT && p.peekN(1).Type == token.COMMA {
			return p.parseMultiConstStatement()
		}

		return p.parseConstStatement()
	case token.IMPORT:
		return p.parseImportStatement()
	case token.ENUM:
		return p.parseEnumStatement()
	case token.TYPE:
		return p.parseTypeStatement()
	case token.SWITCH:
		return p.parseSwitchStatement()
	case token.FUNC:
		if p.peekTok.Type == token.LPAREN {

			// find closing ')'
			i := 1
			depth := 1
			for depth > 0 {
				tok := p.peekN(i)
				if tok.Type == token.LPAREN {
					depth++
				}
				if tok.Type == token.RPAREN {
					depth--
				}
				i++
			}

			afterParen := p.peekN(i)

			if afterParen.Type == token.IDENT {
				return p.parseMethodStatement()
			}

			expr := p.parseExpression(LOWEST)
			return &ExpressionStatement{
				NodeBase:   NodeBase{Token: p.curTok},
				Expression: expr,
			}
		}
		return p.parseFuncStatement()
	case token.SPAWN:
		return p.parseSpawnStatement()
	case token.IF:
		return p.parseIfStatement()
	case token.WITH:
		return p.parseWithStatement()
	case token.FOR:
		return p.parseFor()
	case token.WHILE:
		return p.parseWhileStatement()
	case token.BREAK:
		return p.parseBreakStatement()
	case token.CONTINUE:
		return p.parseContinueStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.DEFER:
		return p.parseDeferStatement()
	case token.IDENT, token.MUL:
		if p.peekUntilAssign() == token.WALRUS {
			if p.peekTok.Type == token.COMMA {
				return p.parseMultiVarStatementNoKeyword()
			}

			return p.parseVarStatementNoKeyword()
		}

		return p.parseAssignOrExprStatement()
	}

	if p.curTok.Type != token.NEWLINE {
		switch p.curTok.Type {
		case token.RBRACE,
			token.SEMICOLON,
			token.COMMA,
			token.RBRACKET,
			token.RPAREN:
			p.addError(fmt.Sprintf("unexpected '%s'", p.curTok.Literal))
			return nil
		}
	}

	return nil
}

func (p *Parser) parseVarStatementBlock() *VarStatementBlock {
	stmt := &VarStatementBlock{
		NodeBase: NodeBase{Token: p.curTok}, // egg
	}

	if p.peekTok.Type != token.LPAREN {
		p.addError("expected '(' in egg block")
		return nil
	}

	p.nextToken() // (
	p.nextToken() // first token inside block

	for p.curTok.Type != token.RPAREN && p.curTok.Type != token.EOF {
		p.consumeTerminators()

		if p.curTok.Type == token.RPAREN {
			break
		}

		decl := p.parseVarBlockDecl()
		if decl != nil {
			stmt.Decls = append(stmt.Decls, decl)
		}

		p.consumeTerminators()
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseVarBlockDecl() Statement {
	id := p.curTok
	var stmt Statement = &VarStatement{
		NodeBase: NodeBase{Token: id},
	}

	if p.curTok.Type != token.IDENT {
		p.addError("expected identifier in egg block")
		return nil
	}

	if p.peekTok.Type == token.COMMA {
		stmt = &MultiVarStatement{
			NodeBase: NodeBase{Token: id},
		}
	}

	switch stmt := stmt.(type) {
	case *VarStatement:
		stmt.Name = &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		}

		// optional lifetime
		if p.peekTok.Type == token.LT {
			p.nextToken() // move to '<'
			p.nextToken() // move to first token of lifetime expression

			stmt.Lifetime = p.parseExpressionUntil(token.GT)

			if p.peekTok.Type != token.GT {
				p.addError("expected '>' after lifetime expression")
				return nil
			}

			p.nextToken() // move to '>'
		}

		if p.isType() {
			p.nextToken()
			stmt.Type = p.parseType()
		}

		if p.peekTok.Type == token.ASSIGN {
			p.nextToken() // =
			p.nextToken() // expression start

			stmt.Value = p.parseExpression(LOWEST)
		}

		return stmt

	case *MultiVarStatement:
		stmt.Names = p.parseIdentPtrList()

		// optional lifetime
		if p.peekTok.Type == token.LT {
			p.nextToken() // move to '<'
			p.nextToken() // move to first token of lifetime expression

			stmt.Lifetime = p.parseExpressionUntil(token.GT)

			if p.peekTok.Type != token.GT {
				p.addError("expected '>' after lifetime expression")
				return nil
			}

			p.nextToken() // move to '>'
		}

		if p.isType() {
			p.nextToken()
			stmt.Type = p.parseType()
		}

		// optional assignment
		if p.peekTok.Type == token.ASSIGN {
			p.nextToken() // move to '='
			p.nextToken() // move to expr start

			values := p.parseTupleList()

			stmt.Values = values
		}

		return stmt
	}

	return nil
}

func (p *Parser) parseVarStatement() *VarStatement {
	stmt := &VarStatement{
		NodeBase: NodeBase{Token: p.curTok}, // 'egg'
	}

	// egg -> name
	p.nextToken()
	stmt.Name = &Identifier{
		NodeBase: NodeBase{Token: p.curTok},
		Value:    p.curTok.Literal,
	}

	// optional lifetime
	if p.peekTok.Type == token.LT {
		p.nextToken() // move to '<'
		p.nextToken() // move to first token of lifetime expression

		stmt.Lifetime = p.parseExpressionUntil(token.GT)

		if p.peekTok.Type != token.GT {
			p.addError("expected '>' after lifetime expression")
			return nil
		}

		p.nextToken() // move to '>'
	}

	// optional type
	if p.isType() {
		p.nextToken() // move to type start
		stmt.Type = p.parseType()
	}

	// optional assignment
	if p.peekTok.Type == token.ASSIGN {
		p.nextToken() // move to '='
		p.nextToken() // move to expression
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

	// optional lifetime
	if p.peekTok.Type == token.LT {
		p.nextToken() // move to '<'
		p.nextToken() // move to first token of lifetime expression

		stmt.Lifetime = p.parseExpressionUntil(token.GT)

		if p.peekTok.Type != token.GT {
			p.addError("expected '>' after lifetime expression")
			return nil
		}

		p.nextToken() // move to '>'
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
	stmt.Names = p.parseIdentPtrList()

	// optional lifetime
	if p.peekTok.Type == token.LT {
		p.nextToken() // move to '<'
		p.nextToken() // move to first token of lifetime expression

		stmt.Lifetime = p.parseExpressionUntil(token.GT)

		if p.peekTok.Type != token.GT {
			p.addError("expected '>' after lifetime expression")
			return nil
		}

		p.nextToken() // move to '>'
	}

	// optional type
	if p.isType() {
		p.nextToken()
		stmt.Type = p.parseType()
	}

	// optional assignment
	if p.peekTok.Type == token.ASSIGN {
		p.nextToken() // move to '='
		p.nextToken() // move to expr start

		values := p.parseTupleList()

		stmt.Values = values
	}

	return stmt
}

func (p *Parser) parseMultiVarStatementNoKeyword() *MultiVarStatementNoKeyword {
	stmt := &MultiVarStatementNoKeyword{
		NodeBase: NodeBase{Token: p.curTok}, // idents
	}

	stmt.Names = p.parseIdentPtrList()

	// optional lifetime
	if p.peekTok.Type == token.LT {
		p.nextToken() // move to '<'
		p.nextToken() // move to first token of lifetime expression

		stmt.Lifetime = p.parseExpressionUntil(token.GT)

		if p.peekTok.Type != token.GT {
			p.addError("expected '>' after lifetime expression")
			return nil
		}

		p.nextToken() // move to '>'
	}

	p.nextToken() // :=
	p.nextToken() // move to expr start

	values := p.parseTupleList()

	stmt.Values = values

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

func (p *Parser) parseConstStatementBlock() *ConstStatementBlock {
	stmt := &ConstStatementBlock{
		NodeBase: NodeBase{Token: p.curTok}, // const
	}

	if p.peekTok.Type != token.LPAREN {
		p.addError("expected '(' in const block")
		return nil
	}

	p.nextToken() // (
	p.nextToken() // first token inside block

	for p.curTok.Type != token.RPAREN && p.curTok.Type != token.EOF {
		p.consumeTerminators()

		if p.curTok.Type == token.RPAREN {
			break
		}

		decl := p.parseConstBlockDecl()
		if decl != nil {
			stmt.Decls = append(stmt.Decls, decl)
		}

		p.consumeTerminators()
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseConstBlockDecl() Statement {
	id := p.curTok
	var stmt Statement = &ConstStatement{
		NodeBase: NodeBase{Token: id},
	}

	if p.curTok.Type != token.IDENT {
		p.addError("expected identifier in rock block")
		return nil
	}

	if p.peekTok.Type == token.COMMA {
		stmt = &MultiConstStatement{
			NodeBase: NodeBase{Token: id},
		}
	}

	switch stmt := stmt.(type) {

	case *ConstStatement:
		stmt.Name = &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		}

		// optional lifetime
		if p.peekTok.Type == token.LT {
			p.nextToken() // move to '<'
			p.nextToken() // move to first token of lifetime expression

			stmt.Lifetime = p.parseExpressionUntil(token.GT)

			if p.peekTok.Type != token.GT {
				p.addError("expected '>' after lifetime expression")
				return nil
			}

			p.nextToken() // move to '>'
		}

		if p.isType() {
			p.nextToken()
			stmt.Type = p.parseType()
		}

		// assignment
		if p.peekTok.Type == token.ASSIGN {
			p.nextToken()
			p.nextToken()
			stmt.Value = p.parseExpression(LOWEST)
		}

		return stmt
	case *MultiConstStatement:
		stmt.Names = p.parseIdentPtrList()

		// optional lifetime
		if p.peekTok.Type == token.LT {
			p.nextToken() // move to '<'
			p.nextToken() // move to first token of lifetime expression

			stmt.Lifetime = p.parseExpressionUntil(token.GT)

			if p.peekTok.Type != token.GT {
				p.addError("expected '>' after lifetime expression")
				return nil
			}

			p.nextToken() // move to '>'
		}

		if p.isType() {
			p.nextToken()
			stmt.Type = p.parseType()
		}

		// optional assignment
		if p.peekTok.Type == token.ASSIGN {
			p.nextToken() // move to '='
			p.nextToken() // move to expr start

			values := p.parseTupleList()

			stmt.Values = values
		}

		return stmt
	}

	return nil
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

	// optional lifetime
	if p.peekTok.Type == token.LT {
		p.nextToken() // move to '<'
		p.nextToken() // move to first token of lifetime expression

		stmt.Lifetime = p.parseExpressionUntil(token.GT)

		if p.peekTok.Type != token.GT {
			p.addError("expected '>' after lifetime expression")
			return nil
		}

		p.nextToken() // move to '>'
	}

	// type
	if p.isType() {
		p.nextToken()
		stmt.Type = p.parseType()
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
	stmt.Names = p.parseIdentPtrList()

	// optional lifetime
	if p.peekTok.Type == token.LT {
		p.nextToken() // move to '<'
		p.nextToken() // move to first token of lifetime expression

		stmt.Lifetime = p.parseExpressionUntil(token.GT)

		if p.peekTok.Type != token.GT {
			p.addError("expected '>' after lifetime expression")
			return nil
		}

		p.nextToken() // move to '>'
	}

	// optional type
	if p.isType() {
		p.nextToken()
		stmt.Type = p.parseType()
	}

	// optional assignment
	if p.peekTok.Type == token.ASSIGN {
		p.nextToken() // move to '='
		p.nextToken() // move to expr start

		values := p.parseTupleList()

		stmt.Values = values
	}

	return stmt
}

func (p *Parser) parseImportStatement() *ImportStatement {
	stmt := &ImportStatement{
		NodeBase: NodeBase{Token: p.curTok},
	}

	p.nextToken()
	stmt.Name = p.curTok.Literal

	return stmt
}

func (p *Parser) parseAssignOrExprStatement() Statement {

	exprs := p.parseExpressionList()

	if p.isAssignToken(p.peekTok.Type) {
		op := p.peekTok.Type
		p.nextToken() // =
		p.nextToken()

		values := p.parseExpressionList()

		return &AssignmentStatement{
			NodeBase: NodeBase{Token: p.curTok},
			Targets:  exprs,
			Op:       op,
			Values:   values,
		}
	}

	// otherwise it's just an expression statement
	if len(exprs) == 1 {
		return &ExpressionStatement{
			NodeBase:   NodeBase{Token: p.curTok},
			Expression: exprs[0],
		}
	}

	p.addError("invalid expression list")
	return nil
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

	p.nextToken()

	if p.curTok.Type == token.ASSIGN {
		stmt.Alias = true
		p.nextToken()
	}

	stmt.Type = p.parseType()

	return stmt
}

func (p *Parser) exprToType(expr Expression) TypeNode {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {

	case TypeNode:
		return e

	case *Identifier:
		return &IdentType{
			Name: e,
		}

	case *MemberExpression:
		mod, ok := e.Left.(*Identifier)
		if !ok {
			p.addError("qualified type must be module.Type")
			return nil
		}

		return &QualifiedType{
			Module: mod,
			Name:   e.Field,
		}

	default:
		p.addError("invalid type expression")
		return nil
	}
}

func (p *Parser) parseType() TypeNode {
	var base TypeNode

	switch p.curTok.Type {
	case token.MUL:
		p.nextToken()
		base := p.parseType()

		return &PointerType{
			NodeBase: NodeBase{Token: p.curTok},
			Base:     base,
		}

	case token.INT_TYPE,
		token.FLOAT_TYPE,
		token.BOOL_TYPE,
		token.STRING_TYPE,
		token.ANY_TYPE,
		token.ERROR_TYPE:

		base = &IdentType{
			NodeBase: NodeBase{Token: p.curTok},
			Name: &Identifier{
				NodeBase: NodeBase{Token: p.curTok},
				Value:    p.curTok.Literal,
			},
		}

	case token.IDENT:
		ident := &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		}

		if p.peekTok.Type == token.DOT {
			p.nextToken()
			p.nextToken()

			if p.curTok.Type != token.IDENT {
				p.addError("expected identifier after '.'")
				return nil
			}

			base = &QualifiedType{
				Module: ident,
				Name: &Identifier{
					NodeBase: NodeBase{Token: p.curTok},
					Value:    p.curTok.Literal,
				},
			}

		} else {
			base = &IdentType{
				NodeBase: NodeBase{Token: p.curTok},
				Name:     ident,
			}
		}

	case token.FUNC:
		base = p.parseFuncType()

	case token.STRUCT:
		base = p.parseStructType()

	case token.INTERFACE:
		base = p.parseInterfaceType()

	case token.LBRACKET:
		base = p.parseArrayType()

	case token.MAP:
		base = p.parseMapType()

	default:
		return p.exprToType(p.parseExpression(LOWEST))
	}

	for p.peekTok.Type == token.LBRACKET {
		if p.peekN(2).Type == token.DUODOT {
			base = p.parseRangeType(base)
		} else {
			break
		}
	}

	return base
}

func (p *Parser) parseTypeList(end token.TokenType) []TypeNode {
	list := []TypeNode{}
	p.nextToken()
	if p.curTok.Type == end {
		return list
	}
	if p.curTok.Type == token.IDENT && p.isTypeToken(p.peekTok.Type) {
		p.nextToken()
	}
	list = append(list, p.parseType())
	for p.peekTok.Type == token.COMMA {
		p.nextToken()
		p.nextToken()
		if p.curTok.Type == token.IDENT && p.isTypeToken(p.peekTok.Type) {
			p.nextToken()
		}
		list = append(list, p.parseType())
	}
	if p.peekTok.Type != end {
		p.addError(fmt.Sprintf("expected '%s'", end))
		return nil
	}
	p.nextToken()
	return list
}

func (p *Parser) parseRangeType(base TypeNode) TypeNode {
	p.nextToken() // move to '['
	p.nextToken() // first token of min

	min := p.parseExpression(LOWEST)

	if p.curTok.Type != token.DUODOT {
		if p.peekTok.Type == token.DUODOT {
			p.nextToken()
		} else {
			p.addError("expected '..' in range type")
			return nil
		}
	}

	p.nextToken() // move to max start

	max := p.parseExpression(LOWEST)

	if p.peekTok.Type != token.RBRACKET {
		p.addError("expected ']' after range type")
		return nil
	}

	p.nextToken() // consume ']'

	return &RangeType{
		Base: base,
		Min:  min,
		Max:  max,
	}
}

func (p *Parser) parseArrayType() TypeNode {
	at := &ArrayType{
		NodeBase: NodeBase{Token: p.curTok}, // '['
	}

	p.nextToken()

	if p.curTok.Type == token.RBRACKET {
		p.nextToken()
		at.Elem = p.parseType()
		return at
	}

	at.Size = p.parseExpression(LOWEST)

	if p.peekTok.Type != token.RBRACKET {
		p.addError("expected ']'")
		return nil
	}

	p.nextToken()
	p.nextToken()

	at.Elem = p.parseType()

	return at
}

func (p *Parser) parseInterfaceType() TypeNode {
	tok := p.curTok

	if p.peekTok.Type != token.LBRACE {
		p.addError("expected '{' after interface")
		return nil
	}

	p.nextToken() // {
	p.nextToken() // method

	methods := []*FuncType{}
	p.consumeTerminators()

	for p.curTok.Type != token.RBRACE {

		if p.curTok.Type != token.IDENT {
			p.addError("expected method name inside interface type")
			return nil
		}

		methodName := &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		}

		p.nextToken() // (
		if p.curTok.Type != token.LPAREN {
			p.addError("expected '(' after method name")
			return nil
		}

		methodParams := p.parseTypeList(token.RPAREN)
		var methodReturns []TypeNode = nil

		if p.peekTok.Type == token.LPAREN {
			p.nextToken() // (
			methodReturns = p.parseTypeList(token.RPAREN)
		}

		methods = append(methods, &FuncType{
			NodeBase: NodeBase{Token: p.curTok},
			Name:     methodName,
			Params:   methodParams,
			Returns:  methodReturns,
		})

		p.nextToken()
		p.consumeTerminators()
	}

	return &InterfaceType{
		NodeBase: NodeBase{Token: tok},
		Methods:  methods,
	}
}

func (p *Parser) parseStructType() TypeNode {
	tok := p.curTok

	if p.peekTok.Type != token.LBRACE {
		p.addError("expected '{' after struct")
		return nil
	}

	p.nextToken() // {
	p.nextToken() // first field or }

	fields := []*StructField{}
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

		p.nextToken() // move to type

		fieldType := p.parseType()

		fields = append(fields, &StructField{
			Name: fieldName,
			Type: fieldType,
		})

		p.nextToken() // move past type

		p.consumeTerminators()
	}

	return &StructType{
		NodeBase: NodeBase{Token: tok},
		Fields:   fields,
	}
}

func (p *Parser) parseMapType() TypeNode {
	if p.peekTok.Type != token.LBRACKET {
		p.addError("expected '[' in key type")
		return nil
	}
	p.nextToken() // [
	p.nextToken() // key type (eg: string)

	key := p.parseType()
	if key == nil {
		return nil
	}

	if p.peekTok.Type != token.RBRACKET {
		p.addError("expected ']' after key type")
		return nil
	}
	p.nextToken() // ]
	p.nextToken() // value type (eg: string)

	val := p.parseType()
	if val == nil {
		return nil
	}

	return &MapType{
		NodeBase: NodeBase{Token: p.curTok},
		Key:      key,
		Value:    val,
	}
}

func (p *Parser) parseFuncType() TypeNode {
	typ := &FuncType{
		NodeBase: NodeBase{Token: p.curTok},
		Params:   make([]TypeNode, 0),
		Returns:  make([]TypeNode, 0),
	}

	if p.peekTok.Type != token.LPAREN {
		p.addError("expected '(' after fun")
		return nil
	}

	p.nextToken()
	typ.Params = p.parseTypeList(token.RPAREN)

	if p.peekTok.Type == token.LPAREN {
		p.nextToken()
		typ.Returns = p.parseTypeList(token.RPAREN)
	}

	return typ
}

func (p *Parser) parseCompositeLiteral(typ TypeNode) Expression {

	lit := &CompositeLiteral{
		NodeBase: NodeBase{Token: p.curTok}, // '{'
		Type:     typ,
		Elements: []Expression{},
		Fields:   make(map[string]Expression),
		Pairs:    []MapPair{},
	}

	p.nextToken()

	p.consumeTerminators()

	if p.curTok.Type == token.RBRACE {
		return lit
	}

	for p.curTok.Type != token.RBRACE {

		if p.curTok.Type == token.IDENT && p.peekTok.Type == token.COLON {
			fieldName := p.curTok.Literal
			p.nextToken() // :
			p.nextToken() // value
			lit.Fields[fieldName] = p.parseExpression(LOWEST)
		} else {
			first := p.parseExpression(LOWEST)

			if p.peekTok.Type == token.COLON {
				p.nextToken() // :
				p.nextToken() // value
				value := p.parseExpression(LOWEST)

				lit.Pairs = append(lit.Pairs, MapPair{
					Key:   first,
					Value: value,
				})
			} else {
				lit.Elements = append(lit.Elements, first)
			}
		}

		p.consumeTerminators()

		if p.peekTok.Type == token.COMMA {
			p.nextToken() // move to comma
			p.nextToken() // move to next element
			p.consumeTerminators()
			continue
		}

		if p.peekTok.Type == token.RBRACE {
			p.nextToken() // move to '}'
			break
		}

		p.addError("expected ',' or '}' in composite literal")
		return nil
	}

	return lit
}

func (p *Parser) parseEnumStatement() *EnumStatement {
	stmt := &EnumStatement{
		NodeBase: NodeBase{Token: p.curTok}, // enum
	}

	// enum Color
	if p.peekTok.Type != token.IDENT {
		p.addError("expected identifier after 'enum'")
		return nil
	}
	p.nextToken()

	stmt.Name = &Identifier{
		NodeBase: NodeBase{Token: p.curTok},
		Value:    p.curTok.Literal,
	}

	// {
	if p.peekTok.Type != token.LBRACE {
		p.addError("expected '{' after enum identifier")
		return nil
	}
	p.nextToken() // curTok == '{'

	p.nextToken()

	stmt.Variants = p.parseEnumVariants()

	if p.curTok.Type != token.RBRACE {
		p.addError("expected '}' after variants")
		return nil
	}

	p.nextToken() // consume }

	return stmt
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

func (p *Parser) parseDeferStatement() *DeferStatement {
	stmt := &DeferStatement{
		NodeBase: NodeBase{Token: p.curTok},
	}

	p.nextToken()
	if p.curTok.Type != token.IDENT {
		p.addError("expected function identifier after defer")
		return nil
	}

	stmt.Call = p.parseFuncCall().(*FuncCall)

	return stmt
}

func (p *Parser) parseSwitchStatement() *SwitchStatement {
	stmt := &SwitchStatement{
		NodeBase: NodeBase{Token: p.curTok},
	}

	p.nextToken()

	if p.curTok.Type == token.LBRACE {
		stmt.Value = nil
	} else {
		stmt.Value = p.parseExpression(LOWEST)

		if p.peekTok.Type != token.LBRACE {
			p.addError("expected '{' after switch expression")
			return nil
		}

		p.nextToken() // {
		p.nextToken() // first token inside
	}

	stmt.Cases = []*CaseClause{}

	for p.curTok.Type != token.EOF {

		p.consumeTerminators()

		switch p.curTok.Type {

		case token.CASE:
			clause := p.parseCaseClause()
			stmt.Cases = append(stmt.Cases, clause)

		case token.DEFAULT:
			stmt.Default = p.parseDefaultClause()

		case token.RBRACE:
			return stmt

		default:
			p.addError("expected 'when' or 'otherwise'")
			return nil
		}

		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseCaseClause() *CaseClause {
	clause := &CaseClause{
		NodeBase: NodeBase{Token: p.curTok},
	}

	// consume `when`
	p.nextToken()

	clause.Exprs = []Expression{}
	clause.Exprs = append(clause.Exprs, p.parseExpression(LOWEST))

	for p.peekTok.Type == token.COMMA {
		p.nextToken() // ,
		p.nextToken() // next expression
		clause.Exprs = append(clause.Exprs, p.parseExpression(LOWEST))
	}

	if p.peekTok.Type != token.LBRACE {
		p.addError("expected '{' after case expression")
		return nil
	}

	p.nextToken() // {
	p.nextToken() // first stmt

	clause.Body = []Statement{}

	for p.curTok.Type != token.RBRACE && p.curTok.Type != token.EOF {
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

func (p *Parser) parseMethodStatement() *MethodStatement {
	stmt := &MethodStatement{
		NodeBase: NodeBase{Token: p.curTok},
		Receiver: &Receiver{},
	}

	// fun (
	p.nextToken()
	if p.curTok.Type != token.LPAREN {
		p.addError("expected '(' after 'fun'")
		return nil
	}

	// receiver name
	p.nextToken()
	if p.curTok.Type != token.IDENT {
		p.addError("expected identifier after '('")
		return nil
	}

	stmt.Receiver.Name = &Identifier{
		NodeBase: NodeBase{Token: p.curTok},
		Value:    p.curTok.Literal,
	}

	// receiver type
	p.nextToken()
	stmt.Receiver.Type = p.parseType()
	if stmt.Receiver.Type == nil {
		p.addError("expected type after identifier")
		return nil
	}

	// )
	p.nextToken()
	if p.curTok.Type != token.RPAREN {
		p.addError("expected ')' after type")
		return nil
	}

	// name
	p.nextToken() // ident
	if p.curTok.Type != token.IDENT {
		p.addError("expected method name after receiver")
		return nil
	}
	stmt.Name = &Identifier{
		NodeBase: NodeBase{Token: p.curTok},
		Value:    p.curTok.Literal,
	}

	p.nextToken() // (
	if p.curTok.Type != token.LPAREN {
		p.addError("expected '(' after method name")
		return nil
	}
	stmt.Params = p.parseFuncParams()

	p.nextToken() // move past ')'

	if p.curTok.Type == token.LPAREN {
		stmt.ReturnTypes = p.parseFuncReturnTypes()
	} else {
		stmt.ReturnTypes = nil
	}

	p.consumeTerminators()

	if p.curTok.Type != token.LBRACE {
		p.addError("expected '{' before method body")
		return nil
	}

	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) parseFuncParams() []*Param {
	params := []*Param{}
	seenVariadic := false

	if p.peekTok.Type == token.RPAREN {
		p.nextToken() // consume ')'
		return params
	}

	p.nextToken() // move to first IDENT

	for {
		if p.curTok.Type != token.IDENT {
			p.addError("expected parameter name")
			return nil
		}

		paramName := &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		}

		p.nextToken()

		variadic := false
		if p.curTok.Type == token.ELLIPSIS {
			variadic = true

			if seenVariadic {
				p.addError("only one variadic parameter allowed")
				return nil
			}

			seenVariadic = true
			p.nextToken()
		}

		if !(p.isTypeToken(p.curTok.Type) || (p.curTok.Type == token.IDENT && p.peekTok.Type == token.DOT) || (p.curTok.Type == token.MUL && p.isPointerTypeM1())) {
			p.addError("expected type after parameter name")
			return nil
		}

		paramType := p.parseType()

		if variadic {
			paramType = &ArrayType{
				NodeBase: NodeBase{Token: paramName.Token},
				Elem:     paramType,
			}
		}

		params = append(params, &Param{
			NodeBase: NodeBase{Token: paramName.Token},
			Name:     paramName,
			Type:     paramType,
			Variadic: variadic,
		})

		if variadic && p.peekTok.Type == token.COMMA {
			p.addError("variadic parameter must be last")
			return nil
		}

		if p.peekTok.Type == token.COMMA {
			p.nextToken() // consume comma
			p.nextToken() // move to next IDENT
			continue
		}

		if p.peekTok.Type != token.RPAREN {
			p.addError("expected ',' or ')'")
			return nil
		}

		p.nextToken() // consume ')'
		break
	}

	return params
}

func (p *Parser) parseFuncReturnTypes() []TypeNode {
	returnTypes := []TypeNode{}

	if p.curTok.Type == token.LPAREN {
		p.nextToken()

		for p.curTok.Type != token.RPAREN {
			if !(p.isTypeToken(p.curTok.Type) || (p.curTok.Type == token.IDENT && p.peekTok.Type == token.DOT) || (p.curTok.Type == token.MUL && p.isPointerTypeM1())) {
				p.addError("expected return type")
				return nil
			}

			typ := p.parseType()
			returnTypes = append(returnTypes, typ)

			p.nextToken()
			if p.curTok.Type == token.COMMA {
				p.nextToken()
			}
		}

		p.nextToken()
		return returnTypes
	}

	return nil
}

func (p *Parser) parseFuncLiteral() *FuncLiteral {
	lit := &FuncLiteral{
		NodeBase: NodeBase{Token: p.curTok},
	}

	// fun <params>
	p.nextToken()
	if p.curTok.Type != token.LPAREN {
		p.addError("expected '(' after 'fun'")
		return nil
	}
	lit.Params = p.parseFuncParams()

	if p.peekTok.Type == token.LPAREN {
		p.nextToken()
		lit.ReturnTypes = p.parseFuncReturnTypes()
	} else {
		lit.ReturnTypes = nil
		p.nextToken()
	}

	p.consumeTerminators()
	if p.curTok.Type != token.LBRACE {
		p.addError("expected '{' before function body")
		return nil
	}

	lit.Body = p.parseBlockStatement()
	return lit
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

	// expect '('
	p.nextToken()
	if p.curTok.Type != token.LPAREN {
		p.addError("expected '(' after function name")
		return nil
	}

	stmt.Params = p.parseFuncParams()
	if stmt.Params == nil {
		return nil
	}

	if p.peekTok.Type == token.LPAREN {
		p.nextToken()
		stmt.ReturnTypes = p.parseFuncReturnTypes()
	} else {
		stmt.ReturnTypes = nil
		p.nextToken()
	}

	p.consumeTerminators()

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
		Callee: &Identifier{
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
	call.Args = p.parseArgList(token.RPAREN)

	return call
}

func (p *Parser) parseReturnStatement() *ReturnStatement {
	stmt := &ReturnStatement{}
	stmt.NodeBase = NodeBase{Token: p.curTok}

	p.nextToken()

	stmt.Values = []Expression{}

	if p.curTok.Type == token.SEMICOLON ||
		p.curTok.Type == token.RBRACE ||
		p.curTok.Type == token.EOF ||
		p.curTok.Type == token.NEWLINE {
		return stmt
	}

	stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))

	for p.peekTok.Type == token.COMMA {
		p.nextToken() // move to comma
		p.nextToken() // move to next expr
		stmt.Values = append(stmt.Values, p.parseExpression(LOWEST))
	}

	return stmt
}

func (p *Parser) parseForInit() Statement {
	if p.curTok.Type == token.IDENT && p.peekTok.Type == token.WALRUS {
		return p.parseForVarNoKeyword()
	}
	return p.parseAssignOrExprStatement()
}

func (p *Parser) parseForVarNoKeyword() *VarStatementNoKeyword {
	stmt := &VarStatementNoKeyword{
		NodeBase: NodeBase{Token: p.curTok}, // ident
	}

	stmt.Name = &Identifier{
		NodeBase: NodeBase{Token: p.curTok},
		Value:    p.curTok.Literal,
	}

	// optional lifetime
	if p.peekTok.Type == token.LT {
		p.nextToken() // move to '<'
		p.nextToken() // move to first token of lifetime expression

		stmt.Lifetime = p.parseExpressionUntil(token.GT)

		if p.peekTok.Type != token.GT {
			p.addError("expected '>' after lifetime expression")
			return nil
		}

		p.nextToken() // move to '>'
	}

	p.nextToken() // :=
	p.nextToken() // expr
	stmt.Value = p.parseExpressionUntil(token.SEMICOLON)

	return stmt
}

func (p *Parser) parseForPost() Statement {
	return p.parseAssignOrExprStatement()
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

func (p *Parser) parseFor() Statement {
	p.nextToken() // move past 'for'

	if p.curTok.Type == token.VAR {
		p.addError("unexpected 'egg', use := instead")
		return nil
	}

	// for range m {}
	if p.curTok.Type == token.RANGE {
		return p.parseForRangeStatement([]*Identifier{})
	}

	idents := []*Identifier{}

	if p.curTok.Type == token.IDENT {
		idents = append(idents, &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		})

		for p.peekTok.Type == token.COMMA {
			p.nextToken() // ,
			p.nextToken() // ident

			if p.curTok.Type != token.IDENT {
				p.addError("expected identifier in for range")
				return nil
			}

			idents = append(idents, &Identifier{
				NodeBase: NodeBase{Token: p.curTok},
				Value:    p.curTok.Literal,
			})
		}
	}

	if p.peekTok.Type == token.WALRUS && p.peekN(1).Type == token.RANGE {
		p.nextToken() // :=
		p.nextToken() // range

		if len(idents) > 2 {
			p.addError("for range allows at most 2 variables")
			return nil
		}

		return p.parseForRangeStatement(idents)
	}

	return p.parseForStatement()
}

func (p *Parser) parseForStatement() *ForStatement {
	stmt := &ForStatement{
		NodeBase: NodeBase{Token: p.curTok},
	}

	stmt.Init = p.parseForInit()

	if p.peekTok.Type != token.SEMICOLON {
		p.addError("expected ';'")
		return nil
	}

	p.nextToken() // ;
	p.nextToken() // condition
	stmt.Condition = p.parseExpression(LOWEST)

	if p.peekTok.Type != token.SEMICOLON {
		p.addError("expected ';'")
		return nil
	}

	p.nextToken() // ;
	p.nextToken() // post
	stmt.Post = p.parseForPost()

	if p.peekTok.Type != token.LBRACE {
		p.addError("expected '{'")
		return nil
	}

	p.nextToken() // {
	stmt.Body = p.parseBlockStatement()

	return stmt
}

func (p *Parser) parseForRangeStatement(idents []*Identifier) *ForRangeStatement {
	stmt := &ForRangeStatement{
		NodeBase: NodeBase{Token: p.curTok}, // range
	}

	if len(idents) > 2 {
		p.addError("for range allows at most 2 variables")
		return nil
	}

	if len(idents) >= 1 {
		stmt.Key = idents[0]
	}
	if len(idents) == 2 {
		stmt.Value = idents[1]
	}

	p.nextToken() // move to expr
	stmt.Expr = p.parseExpression(LOWEST)

	if p.peekTok.Type != token.LBRACE {
		p.addError("expected '{' after range expression")
		return nil
	}

	p.nextToken() // {
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

func (p *Parser) parseWithStatement() *WithStatement {
	stmt := &WithStatement{
		NodeBase: NodeBase{Token: p.curTok}, // with
	}

	p.nextToken()
	stmt.Expr = p.parseExpression(LOWEST)
	if stmt.Expr == nil {
		return nil
	}

	p.nextToken()
	if p.curTok.Type != token.LBRACE {
		p.addError("expected '{' after expression")
		return nil
	}

	p.nextToken() // move past {

	stmt.Body = p.parseBlockStatement()

	if p.curTok.Type != token.RBRACE {
		p.addError("expected '}' after statements")
		return nil
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

func (p *Parser) parseIndexExpression(left Expression) Expression {
	tok := p.curTok // '['

	p.nextToken() // move after '['

	var start Expression
	var end Expression

	if p.curTok.Type == token.COLON {
		p.nextToken() // move to end expression or ']'

		if p.curTok.Type != token.RBRACKET {
			end = p.parseExpression(LOWEST)
		}
	} else {
		start = p.parseExpression(LOWEST)

		if p.peekTok.Type == token.COLON {
			p.nextToken() // consume ':'
			p.nextToken() // move to end expression or ']'

			if p.curTok.Type != token.RBRACKET {
				end = p.parseExpression(LOWEST)
			}
		} else {
			if p.peekTok.Type != token.RBRACKET {
				p.addError("expected ']'")
				return nil
			}
			p.nextToken() // consume ']'

			return &IndexExpression{
				NodeBase: NodeBase{Token: tok},
				Left:     left,
				Index:    start,
			}
		}
	}

	if p.peekTok.Type != token.RBRACKET {
		p.addError("expected ']' after slice expression")
		return nil
	}
	p.nextToken() // consume ']'

	return &SliceExpression{
		NodeBase: NodeBase{Token: tok},
		Left:     left,
		Start:    start,
		End:      end,
	}
}

func (p *Parser) parseDotExpression(left Expression) Expression {

	if p.peekTok.Type == token.LPAREN {
		p.nextToken() // consume '('
		p.nextToken() // move to start of type

		typ := p.parseType()
		if typ == nil {
			p.addError("expected type in type assertion")
			return nil
		}

		if p.peekTok.Type != token.RPAREN {
			p.addError("expected ')' after type assertion")
			return nil
		}

		p.nextToken() // consume ')'

		return &TypeAssertExpression{
			NodeBase: NodeBase{Token: p.curTok},
			Expr:     left,
			Type:     typ,
		}
	}

	p.nextToken() // move to identifier

	if p.curTok.Type != token.IDENT {
		p.addError("expected property name identifier after '.'")
		return nil
	}

	member := &MemberExpression{
		NodeBase: NodeBase{Token: p.curTok},
		Left:     left,
		Field: &Identifier{
			NodeBase: NodeBase{Token: p.curTok},
			Value:    p.curTok.Literal,
		},
	}

	if p.peekTok.Type == token.LBRACE && !p.peekTok.HadWhitespaceBefore {
		p.nextToken()
		return p.parseCompositeLiteral(p.exprToType(member))
	}

	if p.peekTok.Type == token.LPAREN {
		p.nextToken() // consume '('
		return p.parseCallExpression(member)
	}

	return member
}

func (p *Parser) parseExpressionList() []Expression {
	list := []Expression{}

	list = append(list, p.parseExpression(LOWEST))

	for p.peekTok.Type == token.COMMA {
		p.nextToken() // ,
		p.nextToken() // next expression
		list = append(list, p.parseExpression(LOWEST))
	}

	return list
}

func (p *Parser) parseArgList(end token.TokenType) []Expression {
	list := []Expression{}

	p.nextToken() // move past '('
	p.consumeTerminators()

	if p.curTok.Type == end {
		return list
	}

	for {
		expr := p.parseExpression(LOWEST)

		list = append(list, expr)

		p.consumeTerminators()

		if p.peekTok.Type == token.COMMA {
			p.nextToken() // consume comma
			p.nextToken() // move to next expression
			p.consumeTerminators()

			// allow trailing comma
			if p.curTok.Type == end {
				break
			}

			continue
		}

		if p.peekTok.Type == end {
			p.nextToken() // move to ')'
			break
		}

		p.addError(fmt.Sprintf("expected ',' or '%s'", end))
		return nil
	}

	return list
}

func (p *Parser) parseCallExpression(callee Expression) Expression {
	args := p.parseArgList(token.RPAREN)

	return &FuncCall{
		NodeBase: NodeBase{Token: p.curTok},
		Callee:   callee,
		Args:     args,
	}
}

func (p *Parser) parseExpressionUntil(stop token.TokenType) Expression {
	expr := p.parseExpression(LOWEST)

	if p.peekTok.Type == stop {
		return expr
	}

	return expr
}

func (p *Parser) parseExpression(precedence int) Expression {
	left := p.parsePrimary()
	for precedence < p.peekPrecedence() {
		switch p.peekTok.Type {
		case token.LPAREN:
			p.nextToken()
			left = p.parseCallExpression(left)

		case token.LBRACKET:
			p.nextToken()
			left = p.parseIndexExpression(left)

		case token.DOT:
			p.nextToken()
			left = p.parseDotExpression(left)

		case token.ELLIPSIS, token.INC, token.DEC:
			p.nextToken()
			left = &PostfixExpression{
				NodeBase: NodeBase{Token: p.curTok},
				Left:     left,
				Operator: p.curTok.Literal,
			}
			
		default:
			p.nextToken()
			left = p.parseInfixExpression(left)
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
		tok := p.curTok
		p.nextToken()

		right := p.parseExpression(PREFIX)
		if right == nil {
			return nil
		}

		return &PrefixExpression{
			NodeBase: NodeBase{Token: tok},
			Operator: operator,
			Right:    right,
		}

	case token.SUB:
		operator := p.curTok.Literal
		tok := p.curTok
		p.nextToken()

		right := p.parseExpression(PREFIX)
		if right == nil {
			return nil
		}

		return &PrefixExpression{
			NodeBase: NodeBase{Token: tok},
			Operator: operator,
			Right:    right,
		}

	case token.AND:
		operator := p.curTok.Literal
		tok := p.curTok
		p.nextToken()

		right := p.parseExpression(PREFIX)
		if right == nil {
			return nil
		}

		return &PrefixExpression{
			NodeBase: NodeBase{Token: tok},
			Operator: operator,
			Right:    right,
		}

	case token.MUL:
		operator := p.curTok.Literal
		tok := p.curTok
		p.nextToken()

		right := p.parseExpression(PREFIX)
		if right == nil {
			return nil
		}

		return &PrefixExpression{
			NodeBase: NodeBase{Token: tok},
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

	case token.FUNC:
		return p.parseFuncLiteral()

	case token.BOOL_TYPE:
		if p.peekTok.Type == token.LPAREN {
			return p.parseFuncCall()
		}
		return nil

	case token.ERROR_TYPE:
		if p.peekTok.Type == token.LPAREN {
			return p.parseFuncCall()
		}
		return nil

	case token.STRUCT:
		typ := p.parseType()

		if p.peekTok.Type == token.LBRACE {
			p.nextToken()
			return p.parseCompositeLiteral(typ)
		}

		return typ

	case token.IDENT:
		ident := &Identifier{NodeBase: NodeBase{Token: p.curTok}, Value: p.curTok.Literal}

		if p.peekTok.Type == token.LBRACE && !p.peekTok.HadWhitespaceBefore {
			typ := &IdentType{
				NodeBase: NodeBase{Token: p.curTok},
				Name:     ident,
			}
			p.nextToken()
			return p.parseCompositeLiteral(typ)
		}

		return ident

	case token.LBRACKET:
		typ := p.parseType()

		if p.peekTok.Type != token.LBRACE {
			return typ
		}

		p.nextToken()
		return p.parseCompositeLiteral(typ)

	case token.MAP:
		typ := p.parseType()

		if p.peekTok.Type != token.LBRACE {
			p.addError("expected '{'")
			return nil
		}

		p.nextToken()
		return p.parseCompositeLiteral(typ)

	case token.LPAREN:
		p.nextToken()
		exp := p.parseExpression(LOWEST)

		if p.peekTok.Type != token.RPAREN {
			p.addError("expected ')'")
			return nil
		}

		p.nextToken()
		return &GroupedExpression{NodeBase: NodeBase{Token: p.curTok}, Expression: exp}

	default:
		return nil
	}
}
