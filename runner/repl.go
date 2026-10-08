package runner

import (
	"bufio"
	"fmt"
	"os"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/parser"
)

func REPL() {
	scanner := bufio.NewScanner(os.Stdin)
	interp := interpreter.New("<repl>")

	for {
		fmt.Print("\n> ")

		if !scanner.Scan() {
			break
		}

		line := scanner.Text()

		if line == "exit" || line == "quit" {
			break
		}

		l := lexer.New(line)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) > 0 {
			for _, err := range p.Errors() {
				fmt.Println(err)
			}
			continue
		}

		val, err := interp.EvalProgram(program)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if val != nil {
			if _, isNil := val.(interpreter.NilValue); !isNil {
				fmt.Println(val.String())
			}
		}
	}
}
