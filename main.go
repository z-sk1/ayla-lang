package main

import (
	"fmt"
	"os"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/token"
)

func main() {
	debug := false

	if len(os.Args) < 3 || os.Args[1] != "run" {
		fmt.Println("usage: ayla run <file>")
		return
	}

	filename := os.Args[2]

	if os.Args[2] == "--debug" {
		debug = true
		filename = os.Args[3]
	}

	if len(filename) < 5 || filename[len(filename)-5:] != ".ayla" {
		filename += ".ayla"
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
		return
	}

	source := string(content)

	if debug {
		l := lexer.New(string(source))

		for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
			fmt.Println(tok)
		}
	}

	l := lexer.New(source)
	p := parser.New(l)

	program := p.ParseProgram()
	if debug {
		fmt.Printf("AST: %#v\n", program)
	}

	interp := interpreter.New()
	if sig, err := interp.EvalStatements(program); err != nil {
		fmt.Println("Runtime error:", err)
	} else {
		_ = sig
	}
}
