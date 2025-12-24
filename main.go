package main

import (
	"fmt"
	"os"

	"time"

	"strings"
	"math/rand"

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
