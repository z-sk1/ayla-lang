package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"time"

	"math/rand"
	"strings"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/parser"
	_ "github.com/z-sk1/ayla-lang/stdlib"
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
		fmt.Println("Welcome to ayla-lang v1.4.0, do ayla --help to see all commands.")
		return
	}

	switch os.Args[1] {
	case "run":
		if len(os.Args) < 3 {
			fmt.Println("usage: ayla run [--debug] [--timed] <file>")
			return
		}

		run()
	case "install":
		if len(os.Args) < 3 {
			fmt.Println("usage: ayla install <url>")
			return
		}

		install()
	case "--version":
		fmt.Println("ayla-lang v1.4.0")
	case "--help":
		fmt.Println(strings.Join(cmds, "\n"))
	default:
		fmt.Println("unknown command: " + os.Args[1] + ", use --help if you need to see the available commands")
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
			fmt.Printf("%s: %v\n", filename, err)
		}
		return
	}

	var started time.Time

	if timed {
		started = time.Now()
	}

	interp := interpreter.New(filename)

	if err := interp.RegisterForward(program); err != nil {
		fmt.Printf("\n%s: %v\n", filename, err)
		return
	}

	if err := interp.ResolveTypes(program); err != nil {
		fmt.Printf("\n%s: %v\n", filename, err)
		return
	}

	_, err = interp.EvalStatements(program)

	if err != nil {
		fmt.Printf("\n%s: %v\n", filename, err)
		return
	}

	var elapsed time.Duration

	if timed {
		elapsed = time.Since(started)
		fmt.Println(elapsed)
	}
}

func normalizeGitHubURL(url string) string {
	if strings.Contains(url, "github.com") && !strings.Contains(url, "raw.githubusercontent.com") {
		url = strings.Replace(url, "github.com", "raw.githubusercontent.com", 1)
		url = strings.Replace(url, "/blob/", "/", 1)
	}
	return url
}

func install() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		return
	}

	libDir := filepath.Join(home, ".ayla", "lib")
	os.MkdirAll(libDir, 0755)

	url := normalizeGitHubURL(os.Args[2])

	fmt.Println("downloading:", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("failed to download module")
		return
	}

	fileName := filepath.Base(url)

	if !strings.HasSuffix(fileName, ".ayla") && !strings.HasSuffix(fileName, ".ayl") {
		fileName += ".ayla"
	}

	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	moduleDir := filepath.Join(libDir, name)

	os.MkdirAll(moduleDir, 0755)

	dest := filepath.Join(moduleDir, fileName)

	out, err := os.Create(dest)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("installed module:", fileName)
}
