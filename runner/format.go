package runner

import (
	"fmt"
	"os"

	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/parser"
)

func Format(path string) error {
	src, name, err := readSourceFile(path)
	if err != nil {
		return err
	}

	l := lexer.New(string(src))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, e := range p.Errors() {
			fmt.Println(e)
		}
		return fmt.Errorf("parse failed")
	}

	out := parser.FormatProgram(program)

	return os.WriteFile(name, []byte(out), 0644)
}
