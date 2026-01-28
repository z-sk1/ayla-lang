package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	f, err := os.Create("ayla-lsp.log")
	if err == nil {
		log.SetOutput(f)
	}

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

	case "initialized":
		return

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
	rootScope := BuildSymbols(program)

	ident := findIdentAt(program, params.Position)
	if ident == nil {
		s.sendResponse(req.ID, nil)
		return
	}

	sym := rootScope.Resolve(ident.Value)
	if sym == nil {
		s.sendResponse(req.ID, nil)
		return
	}

	if sym.Type == nil && sym.Value != nil {
		inferred := inferExprType(rootScope, sym.Value)
		if inferred != nil {
			sym.Type = inferred
		}
	}

	hoverText := hoverFromSymbol(sym)

	hover := HoverResult{
		Contents: map[string]interface{}{
			"kind":  "markdown",
			"value": hoverText,
		},
	}

	s.sendResponse(req.ID, hover)
}

func hoverFromSymbol(sym *Symbol) string {
	typeStr := "unknown"
	if sym.Type != nil {
		typeStr = sym.Type.Value
	}

	switch sym.Kind {
	case SymVar:
		return fmt.Sprintf("```ayla\negg %s %s\n```", sym.Name, typeStr)
	case SymConst:
		return fmt.Sprintf("```ayla\nrock %s %s\n```", sym.Name, typeStr)
	case SymFunc:
		return fmt.Sprintf("```ayla\nfun %s (...)\n```", sym.Name)
	case SymParam:
		return fmt.Sprintf("```ayla\nparam %s %s\n```", sym.Name, typeStr)
	case SymStructField:
		return fmt.Sprintf("```ayla\nfield %s %s\n```", sym.Name, typeStr)
	case SymType:
		return fmt.Sprintf("```ayla\ntype %s\n```", sym.Name)
	case SymUserType:
		return fmt.Sprintf("```ayla\ntype %s %s\n```", sym.Name, typeStr)
	}
	return sym.Name
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
	rootScope := BuildSymbols(program)

	ident := findIdentAt(program, params.Position)
	if ident == nil {
		s.sendResponse(req.ID, nil)
		return
	}

	sym := rootScope.Resolve(ident.Value)
	if sym == nil {
		return
	}

	line, col := sym.Ident.Pos()

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

func walkForIdent(n parser.Node, pos Position) *parser.Identifier {
	if n == nil {
		return nil
	}

	switch n := n.(type) {

	case *parser.Identifier:
		if posInsideTok(n.NodeBase.Token, pos) {
			return n
		}

	case *parser.ExpressionStatement:
		return walkForIdent(n.Expression, pos)

	case *parser.TypeStatement:
		switch t := n.Type.(type) {

		case *parser.IdentType:
			typeName := t.Name
			return &parser.Identifier{
				NodeBase: n.NodeBase,
				Value:    typeName,
			}

		case *parser.StructType:
			var fieldNames string
			for _, field := range t.Fields {
				if field != nil && field.Name != nil {
					fieldNames += field.Name.Value + "\n"
				}
			}
			typeName := fmt.Sprintf("struct {\n%s}", fieldNames)
			return &parser.Identifier{
				NodeBase: n.NodeBase,
				Value:    typeName,
			}

		default:
			typeName := "unknown"
			return &parser.Identifier{
				NodeBase: n.NodeBase,
				Value:    typeName,
			}
		}

	case *parser.VarStatement:
		if res := walkForIdent(n.Name, pos); res != nil {
			return res
		}

		if n.Type != nil {
			if res := walkForIdent(n.Type, pos); res != nil {
				return res
			}
		}

		if n.Value != nil {
			return walkForIdent(n.Value, pos)
		}

	case *parser.ConstStatement:
		if res := walkForIdent(n.Name, pos); res != nil {
			return res
		}

		if n.Type != nil {
			if res := walkForIdent(n.Type, pos); res != nil {
				return res
			}
		}

		if n.Value != nil {
			return walkForIdent(n.Value, pos)
		}

	case *parser.MultiVarStatement:
		for _, name := range n.Names {
			if res := walkForIdent(name, pos); res != nil {
				return res
			}
		}

		if n.Type != nil {
			if res := walkForIdent(n.Type, pos); res != nil {
				return res
			}
		}

		if n.Value != nil {
			return walkForIdent(n.Value, pos)
		}

	case *parser.MultiConstStatement:
		for _, name := range n.Names {
			if res := walkForIdent(name, pos); res != nil {
				return res
			}
		}

		if n.Type != nil {
			if res := walkForIdent(n.Type, pos); res != nil {
				return res
			}
		}

		if n.Value != nil {
			return walkForIdent(n.Value, pos)
		}

	case *parser.AssignmentStatement:
		if res := walkForIdent(n.Name, pos); res != nil {
			return res
		}

		if n.Value != nil {
			return walkForIdent(n.Value, pos)
		}

	case *parser.MultiAssignmentStatement:
		for _, name := range n.Names {
			if res := walkForIdent(name, pos); res != nil {
				return res
			}
		}

		if n.Value != nil {
			return walkForIdent(n.Value, pos)
		}

	case *parser.InfixExpression:
		if res := walkForIdent(n.Left, pos); res != nil {
			return res
		}
		return walkForIdent(n.Right, pos)

	case *parser.IndexAssignmentStatement:
		if res := walkForIdent(n.Left, pos); res != nil {
			return res
		}
		if res := walkForIdent(n.Index, pos); res != nil {
			return res
		}
		return walkForIdent(n.Value, pos)

	case *parser.IndexExpression:
		if res := walkForIdent(n.Left, pos); res != nil {
			return res
		}
		return walkForIdent(n.Index, pos)

	case *parser.PrefixExpression:
		return walkForIdent(n.Right, pos)

	case *parser.MemberExpression:
		if res := walkForIdent(n.Left, pos); res != nil {
			return res
		}
		if res := walkForIdent(n.Field, pos); res != nil {
			return res
		}

	case *parser.FuncStatement:
		if res := walkForIdent(n.Name, pos); res != nil {
			return res
		}

		for _, param := range n.Params {
			if res := walkForIdent(param, pos); res != nil {
				return res
			}
		}

		for _, stmt := range n.Body {
			if res := walkForIdent(stmt, pos); res != nil {
				return res
			}
		}

	case *parser.SpawnStatement:
		for _, stmt := range n.Body {
			if res := walkForIdent(stmt, pos); res != nil {
				return res
			}
		}

	case *parser.FuncCall:
		if res := walkForIdent(n.Name, pos); res != nil {
			return res
		}

		for _, arg := range n.Args {
			if res := walkForIdent(arg, pos); res != nil {
				return res
			}
		}

	case *parser.StructLiteral:
		if res := walkForIdent(n.TypeName, pos); res != nil {
			return res
		}

		for _, field := range n.Fields {
			if res := walkForIdent(field, pos); res != nil {
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

func posInsideTok(tok token.Token, pos Position) bool {
	// convert token position to 0-based
	line := tok.Line - 1
	startCol := tok.Column - 1
	endCol := startCol
	startCol = startCol - len(tok.Literal)

	if pos.Line != line {
		return false
	}

	return pos.Character >= startCol && pos.Character < endCol
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

func inferExprType(scope *Scope, expr parser.Expression) *parser.Identifier {
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
		left := inferExprType(scope, e.Left)
		right := inferExprType(scope, e.Right)

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
		return inferExprType(scope, e.Right)

	case *parser.Identifier:
		sym := scope.Resolve(e.Value)
		if sym == nil {
			return nil
		}

		return sym.Type
	}

	return nil
}
