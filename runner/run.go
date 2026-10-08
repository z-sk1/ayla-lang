package runner

import (
	"fmt"
	"os"
	"time"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/token"
)

func Run() {
	debug := false
	timed := false
	filename := ""

	for _, arg := range os.Args[2:] {
		switch arg {
		case "--timed":
			timed = true
		case "--debug":
			debug = true
		default:
			filename = arg
		}
	}

	if filename == "" {
		fmt.Println("No input file provided")
		return
	}

	source, name, err := readSourceFile(filename)
	if err != nil {
		fmt.Println(err)
		return
	}

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

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			fmt.Printf("%s: %v\n", name, err)
		}
		return
	}

	var started time.Time

	if timed {
		started = time.Now()
	}

	interp := interpreter.New(name)

	if err := interp.RegisterForward(program); err != nil {
		fmt.Printf("\n%s: %v\n", name, err)
		return
	}

	if err := interp.ResolveTypes(program); err != nil {
		fmt.Printf("\n%s: %v\n", name, err)
		return
	}

	if err := interp.TypeCheck(program); err != nil {
		fmt.Printf("\n%s: %v\n", name, err)
		return
	}

	_, err = interp.EvalStatements(program)

	if err != nil {
		fmt.Printf("\n%s: %v\n", name, err)
		return
	}

	interp.Wg.Wait()

	var elapsed time.Duration

	if timed {
		elapsed = time.Since(started)
		fmt.Println(elapsed)
	}
}
