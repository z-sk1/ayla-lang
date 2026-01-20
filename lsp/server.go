package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/token"
)

type Server struct {
	in  *bufio.Reader
	out *bufio.Writer

	documents map[string]string
}

type Request struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      *int        `json:"id"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error,omitempty"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"` // 1 = Error
	Message  string `json:"message"`
}

type DidOpenParams struct {
	TextDocument struct {
		URI  string `json:"uri"`
		Text string `json:"text"`
	} `json:"textDocument"`
}

type DidChangeParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	ContentChanges []struct {
		Text string `json:"text"`
	} `json:"contentChanges"`
}

type DefinitionParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Position Position `json:"position"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type HoverParams struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Position Position `json:"position"`
}

type HoverResult struct {
	Contents interface{} `json:"contents"`
}

func main() {
	server := NewServer()
	server.Run()
}

func NewServer() *Server {
	return &Server{
		in:        bufio.NewReader(os.Stdin),
		out:       bufio.NewWriter(os.Stdout),
		documents: make(map[string]string),
	}
}

func (s *Server) Run() {
	for {
		msg, err := readMessage(s.in)
		if err != nil {
			return
		}

		s.handleMessage(msg)
	}
}

func (s *Server) handleMessage(req *Request) {
	fmt.Fprintf(os.Stderr, "METHOD: %s\n", req.Method)

	switch req.Method {
	case "initialize":
		s.handleIntialize(req)

	case "textDocument/didOpen":
		s.handleDidOpen(req)

	case "textDocument/didChange":
		s.handleDidChange(req)

	case "textDocument/definition":
		s.handleDefinition(req)

	case "textDocument/hover":
		s.handleHover(req)

	case "shutdown":
		s.sendResponse(req.ID, nil)

	case "exit":
		os.Exit(0)
	}
}

func (s *Server) handleIntialize(req *Request) {
	result := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"textDocumentSync":   1,
			"definitionProvider": true,
			"hoverProvider":      true,
		},
	}

	s.sendResponse(req.ID, result)
}

func (s *Server) handleDidOpen(req *Request) {
	var params DidOpenParams
	json.Unmarshal(req.Params, &params)

	uri := params.TextDocument.URI
	text := params.TextDocument.Text

	s.documents[uri] = text

	// run diagnostics
	s.publishDiagnostics(uri, text)
}

func (s *Server) handleDidChange(req *Request) {
	var params DidChangeParams
	json.Unmarshal(req.Params, &params)

	uri := params.TextDocument.URI
	text := params.ContentChanges[0].Text

	s.documents[uri] = text
	s.publishDiagnostics(uri, text)
}

func (s *Server) handleHover(req *Request) {
	var params HoverParams
	json.Unmarshal(req.Params, &params)

	text := s.documents[params.TextDocument.URI]
	if text == "" {
		s.sendResponse(req.ID, nil)
		return
	}

	l := lexer.New(text)
	p := parser.New(l)
	program := p.ParseProgram()

	ident := findIdentAt(program, params.Position)
	if ident == nil {
		s.sendResponse(req.ID, nil)
		return
	}

	decl := findDeclaration(program, ident.Value)

	if decl == nil {
		s.sendResponse(req.ID, nil)
		return
	}

	var (
		typeIdent *parser.Identifier
		keyword   string
	)

	switch d := decl.(type) {

	case *parser.VarStatement:
		typeIdent = d.Type
		keyword = "egg"
		if typeIdent == nil && d.Value != nil {
			typeIdent = inferExprType(program, d.Value)
		}

	case *parser.ConstStatement:
		typeIdent = d.Type
		keyword = "rock"
		if typeIdent == nil && d.Value != nil {
			typeIdent = inferExprType(program, d.Value)
		}

	case *parser.MultiVarStatement:
		typeIdent = d.Type
		keyword = "egg"

		tuple, ok := d.Value.(*parser.TupleLiteral)
		if !ok {
			break
		}

		for i, name := range d.Names {
			if name == ident.Value && i < len(tuple.Values) {
				typeIdent = inferExprType(program, tuple.Values[i])
				break
			}
		}

	case *parser.MultiConstStatement:
		typeIdent = d.Type
		keyword = "rock"

		tuple, ok := d.Value.(*parser.TupleLiteral)
		if !ok {
			break
		}

		for i, name := range d.Names {
			if name == ident.Value && i < len(tuple.Values) {
				typeIdent = inferExprType(program, tuple.Values[i])
				break
			}
		}
	}

	typeStr := "unknown"
	if typeIdent != nil {
		typeStr = typeIdent.Value
	}

	hoverText := fmt.Sprintf(
		"```ayla\n%s %s %s\n```",
		keyword,
		ident.Value,
		typeStr,
	)

	hover := HoverResult{
		Contents: map[string]interface{}{
			"kind":  "markdown",
			"value": hoverText,
		},
	}

	s.sendResponse(req.ID, hover)
}

func (s *Server) handleDefinition(req *Request) {
	var params DefinitionParams
	json.Unmarshal(req.Params, &params)

	text := s.documents[params.TextDocument.URI]
	if text == "" {
		s.sendResponse(req.ID, nil)
		return
	}

	l := lexer.New(text)
	p := parser.New(l)
	program := p.ParseProgram()

	ident := findIdentAt(program, params.Position)
	if ident == nil {
		s.sendResponse(req.ID, nil)
		return
	}

	decl := findDeclaration(program, ident.Value)
	if decl == nil {
		s.sendResponse(req.ID, nil)
		return
	}

	line, col := decl.Pos()

	line--
	col--

	loc := Location{
		URI: params.TextDocument.URI,
		Range: Range{
			Start: Position{
				Line:      line,
				Character: col,
			},
			End: Position{
				Line:      line,
				Character: col + len(ident.Value),
			},
		},
	}

	s.sendResponse(req.ID, loc)
}

func findIdentAt(statements []parser.Statement, pos Position) *parser.Identifier {
	for _, stmt := range statements {
		ident := walkForIdent(stmt, pos)
		if ident != nil {
			return ident
		}
	}
	return nil
}

func findDeclaration(program []parser.Statement, name string) parser.Statement {
	for i := len(program) - 1; i >= 0; i-- {
		if d := findDeclarationInStmt(program[i], name); d != nil {
			return d
		}
	}

	return nil
}

func findDeclarationInStmt(stmt parser.Statement, name string) parser.Statement {
	switch s := stmt.(type) {
	case *parser.VarStatement:
		if s.Name == name {
			return s
		}

	case *parser.ConstStatement:
		if s.Name == name {
			return s
		}

	case *parser.MultiVarStatement:
		for _, n := range s.Names {
			if n == name {
				return s
			}
		}

	case *parser.MultiConstStatement:
		for _, n := range s.Names {
			if n == name {
				return s
			}
		}

	case *parser.FuncStatement:
		for i := len(s.Body) - 1; i >= 0; i-- {
			if d := findDeclarationInStmt(s.Body[i], name); d != nil {
				return d
			}
		}

	case *parser.ForStatement:
		if s.Init != nil {
			if d := findDeclarationInStmt(s.Init, name); d != nil {
				return d
			}
		}
		for i := len(s.Body) - 1; i >= 0; i-- {
			if d := findDeclarationInStmt(s.Body[i], name); d != nil {
				return d
			}
		}

	case *parser.WhileStatement:
		for i := len(s.Body) - 1; i >= 0; i-- {
			if d := findDeclarationInStmt(s.Body[i], name); d != nil {
				return d
			}
		}

	case *parser.IfStatement:
		for i := len(s.Consequence) - 1; i >= 0; i-- {
			if d := findDeclarationInStmt(s.Consequence[i], name); d != nil {
				return d
			}
		}
		for i := len(s.Alternative) - 1; i >= 0; i-- {
			if d := findDeclarationInStmt(s.Alternative[i], name); d != nil {
				return d
			}
		}
	}

	return nil
}

func walkForIdent(n parser.Node, pos Position) *parser.Identifier {
	switch n := n.(type) {

	case *parser.Identifier:
		if posInsideIdent(n, pos) {
			return n
		}

	case *parser.ExpressionStatement:
		return walkForIdent(n.Expression, pos)

	case *parser.VarStatement:
		// Check identifier being declared
		if posInsideIdent(&parser.Identifier{
			NodeBase: n.NodeBase,
			Value:    n.Name,
		}, pos) {
			return &parser.Identifier{
				NodeBase: n.NodeBase,
				Value:    n.Name,
			}
		}

		if n.Value != nil {
			return walkForIdent(n.Value, pos)
		}

	case *parser.ConstStatement:
		// check ident being declared
		if posInsideIdent(&parser.Identifier{
			NodeBase: n.NodeBase,
			Value:    n.Name,
		}, pos) {
			return &parser.Identifier{
				NodeBase: n.NodeBase,
				Value:    n.Name,
			}
		}

		if n.Value != nil {
			return walkForIdent(n.Value, pos)
		}

	case *parser.MultiVarStatement:
		for i, tok := range n.NameTokens {
			if posInsideTok(tok, pos) {
				return &parser.Identifier{
					Value: n.Names[i],
				}
			}
		}

		if n.Value != nil {
			return walkForIdent(n.Value, pos)
		}

	case *parser.MultiConstStatement:
		for i, tok := range n.NameTokens {
			if posInsideTok(tok, pos) {
				return &parser.Identifier{
					Value: n.Names[i],
				}
			}
		}

		if n.Value != nil {
			return walkForIdent(n.Value, pos)
		}

	case *parser.AssignmentStatement:
		if posInsideIdent(&parser.Identifier{
			NodeBase: n.NodeBase,
			Value:    n.Name,
		}, pos) {
			return &parser.Identifier{
				NodeBase: n.NodeBase,
				Value:    n.Name,
			}
		}

		return walkForIdent(n.Value, pos)

	case *parser.InfixExpression:
		if res := walkForIdent(n.Left, pos); res != nil {
			return res
		}
		return walkForIdent(n.Right, pos)

	case *parser.PrefixExpression:
		return walkForIdent(n.Right, pos)

	case *parser.FuncStatement:
		for _, stmt := range n.Body {
			if res := walkForIdent(stmt, pos); res != nil {
				return res
			}
		}

	case *parser.FuncCall:
		for _, arg := range n.Args {
			if res := walkForIdent(arg, pos); res != nil {
				return res
			}
		}

	case *parser.IfStatement:
		if res := walkForIdent(n.Condition, pos); res != nil {
			return res
		}
		for _, stmt := range n.Consequence {
			if res := walkForIdent(stmt, pos); res != nil {
				return res
			}
		}
		for _, stmt := range n.Alternative {
			if res := walkForIdent(stmt, pos); res != nil {
				return res
			}
		}

	case *parser.ForStatement:
		if n.Init != nil {
			if res := walkForIdent(n.Init, pos); res != nil {
				return res
			}
		}
		if n.Condition != nil {
			if res := walkForIdent(n.Condition, pos); res != nil {
				return res
			}
		}
		if n.Post != nil {
			if res := walkForIdent(n.Post, pos); res != nil {
				return res
			}
		}
		for _, stmt := range n.Body {
			if res := walkForIdent(stmt, pos); res != nil {
				return res
			}
		}

	case *parser.WhileStatement:
		if res := walkForIdent(n.Condition, pos); res != nil {
			return res
		}
		for _, stmt := range n.Body {
			if res := walkForIdent(stmt, pos); res != nil {
				return res
			}
		}
	}

	return nil
}

func posInsideIdent(ident *parser.Identifier, pos Position) bool {
	line1, col1 := ident.Pos()

	// token → LSP (0-based)
	line := line1 - 1
	start := col1 - 1
	end := start + len(ident.Value)

	if pos.Line != line {
		return false
	}

	// tolerate VS Code click jitter
	return pos.Character >= start-len(ident.Value) && pos.Character <= end
}

func posInsideTok(tok token.Token, pos Position) bool {
	// convert token position to 0-based
	line := tok.Line - 1
	startCol := tok.Column - 1
	endCol := startCol + len(tok.Literal)

	if pos.Line != line {
		return false
	}

	return pos.Character >= startCol-len(tok.Literal) && pos.Character < endCol
}

func tokenRange(pe *parser.ParseError) Range {
	startCol := pe.Column - 1
	length := len(pe.Token.Literal)
	if length == 0 {
		length = 1
	}

	return Range{
		Start: Position{
			Line:      pe.Line - 1,
			Character: startCol,
		},
		End: Position{
			Line:      pe.Line - 1,
			Character: startCol + length,
		},
	}
}

func (s *Server) publishDiagnostics(uri string, text string) {
	l := lexer.New(text)
	p := parser.New(l)
	p.ParseProgram()

	diagnostics := []Diagnostic{}

	for _, err := range p.Errors() {
		pe, ok := err.(*parser.ParseError)
		if !ok {
			continue
		}

		diagnostics = append(diagnostics, Diagnostic{
			Range:    tokenRange(pe),
			Severity: 1, // Error
			Message:  pe.Error(),
		})
	}

	params := map[string]interface{}{
		"uri":         uri,
		"diagnostics": diagnostics,
	}

	s.sendNotification("textDocument/publishDiagnostics", params)
}

func (s *Server) sendNotification(method string, params interface{}) {
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}

	data, _ := json.Marshal(msg)
	writeMessage(s.out, data)
}

func (s *Server) sendResponse(id *int, result interface{}) {
	if id == nil {
		return
	}

	resp := Response{
		Jsonrpc: "2.0",
		ID:      id,
		Result:  result,
	}

	data, _ := json.Marshal(resp)
	writeMessage(s.out, data)
}

func readMessage(r *bufio.Reader) (*Request, error) {
	// read headers
	var contentLength int
	for {
		line, _ := r.ReadString('\n')
		if line == "\r\n" {
			break
		}
		fmt.Sscanf(line, "Content-Length: %d\r\n", &contentLength)
	}

	body := make([]byte, contentLength)

	_, err := io.ReadFull(r, body)
	if err != nil {
		return nil, err
	}

	var req Request
	json.Unmarshal(body, &req)
	return &req, nil
}

func writeMessage(w *bufio.Writer, data []byte) {
	fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(data))
	w.Write(data)
	w.Flush()
}

func inferExprType(program []parser.Statement, expr parser.Expression) *parser.Identifier {
	switch e := expr.(type) {

	case *parser.IntLiteral:
		return &parser.Identifier{Value: "int"}

	case *parser.FloatLiteral:
		return &parser.Identifier{Value: "float"}

	case *parser.StringLiteral:
		return &parser.Identifier{Value: "string"}

	case *parser.BoolLiteral:
		return &parser.Identifier{Value: "bool"}

	case *parser.ArrayLiteral:
		return &parser.Identifier{Value: "arr"}

	case *parser.AnonymousStructLiteral:
		return &parser.Identifier{Value: "struct"}

	case *parser.StructLiteral:
		return &parser.Identifier{Value: e.TypeName.Value}

	case *parser.InfixExpression:
		left := inferExprType(program, e.Left)
		right := inferExprType(program, e.Right)

		if left == nil || right == nil {
			return nil
		}

		// very simple rules for now
		if left.Value == right.Value {
			return left
		}

		// int + float => float
		if (left.Value == "int" && right.Value == "float") ||
			(left.Value == "float" && right.Value == "int") {
			return &parser.Identifier{Value: "float"}
		}

	case *parser.PrefixExpression:
		return inferExprType(program, e.Right)

	case *parser.Identifier:
		if d := findDeclaration(program, e.Value); d != nil {
			if d, ok := d.(*parser.VarStatement); ok {
				if d.Type != nil {
					return d.Type
				}
				if d.Value != nil {
					return inferExprType(program, d.Value)
				}
			}

			if d, ok := d.(*parser.ConstStatement); ok {
				if d.Type != nil {
					return d.Type
				}
				if d.Value != nil {
					return inferExprType(program, d.Value)
				}
			}
		}
	}

	return nil
}
