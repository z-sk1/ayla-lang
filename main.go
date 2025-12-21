package main

import (
	"fmt"
	"log"
	"os"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/token"
)

func main() {
	input, err := os.ReadFile("array.txt")
	if err != nil {
		log.Fatal(err)
	}

	l := lexer.New(string(input))

	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		fmt.Println(tok)
	}

	l = lexer.New(string(input))
	p := parser.New(l)
	program := p.ParseProgram()
	fmt.Printf("AST: %#v\n", program)

	interp := interpreter.New()
	if sig, err := interp.EvalStatements(program); err != nil {
		fmt.Println("Runtime error:", err)
	} else {
		_ = sig
	}

	for {
		tok := l.NextToken()
		fmt.Printf("%+v\n", tok)
		if tok.Type == "EOF" {
			break
		}
	}
}
