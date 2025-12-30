package main

import (
	"fmt"
	"os"

	"time"

	"math/rand"
	"strings"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/token"
)

func main() {
	rand.Seed(time.Now().Unix())

	cmds := []string{
		"run: ayla run [--debug] [--timed] <file>, runs the ayla script",
		"--version: ayla --version, returns the current version",
		"--help: ayla --help, returns all the available commands",
	}

	if len(os.Args) == 1 {
		fmt.Println("Welcome to ayla-lang v1.0, do ayla --help to see all commands.")
		return
	}

	switch os.Args[1] {
	case "run":
		if len(os.Args) < 3 {
			fmt.Println("usage: ayla run [--debug] [--timed] <file>")
			return
		}

		run()
	case "--version":
		fmt.Println("ayla-lang v1.1")
	case "--help":
		fmt.Println(strings.Join(cmds, "\n"))
	default:
		fmt.Println("unknown command: " + os.Args[1] + ", use --help if you need to see the available commands")
	}
}

func run() {
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

	source, err := readSourceFile(filename)
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
			fmt.Println(err)
		}
		return
	}

	var started time.Time

	if timed {
		started = time.Now()
	}

	interp := interpreter.New()
	if sig, err := interp.EvalStatements(program); err != nil {
		fmt.Println(err)
	} else {
		_ = sig
	}

	var elapsed time.Duration

	if timed {
		elapsed = time.Since(started)
		fmt.Println(elapsed)
	}
}

func readSourceFile(name string) (string, error) {
	candidates := []string{
		name,
		name + ".ayl",
		name + ".ayla",
	}

	for _, file := range candidates {
		data, err := os.ReadFile(file)
		if err == nil {
			return string(data), nil
		}
	}

	return "", fmt.Errorf("file not found: %s (.ayla or .ayl)", name)
}
