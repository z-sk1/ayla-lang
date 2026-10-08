package runner

import (
	"fmt"
	"os"
	"strings"
)

func Main() {
	cmds := []string{
		"run: ayla run [--debug] [--timed] <file>",
		"build: ayla build [-o name] <file>",
		"fmt: ayla fmt <file>",
		"install: ayla install <url>",
		"--version: ayla --version",
		"--help: ayla --help",
	}

	if len(os.Args) == 1 {
		fmt.Println("Welcome to ayla-lang v1.5.2")
		REPL()
		return
	}

	switch os.Args[1] {
	case "run":
		Run()
	case "build":
		Build()
	case "fmt":
		if len(os.Args) < 3 {
			fmt.Println("usage: ayla fmt <file>")
			return
		}
		if err := Format(os.Args[2]); err != nil {
			fmt.Println(err)
		}
	case "install":
		Install()
	case "--version":
		fmt.Println("ayla-lang v1.5.2")
	case "--help":
		fmt.Println(strings.Join(cmds, "\n"))
	default:
		fmt.Println("unknown command:", os.Args[1])
	}
}
