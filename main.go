package main

import (
	"bufio"
	"bytes"
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
	rand.Seed(time.Now().UnixNano())

	exe, err := os.Executable()
	if err == nil {
		data, err := os.ReadFile(exe)
		if err == nil {
			startMarker := []byte("\n__AYLA_SCRIPT_START__\n")
			endMarker := []byte("\n__AYLA_SCRIPT_END__\n")

			start := bytes.LastIndex(data, startMarker)
			end := bytes.LastIndex(data, endMarker)

			if start != -1 && end != -1 && end > start {
				start += len(startMarker)
				script := data[start:end]

				runEmbedded(string(script))
				return
			}
		}
	}

	cmds := []string{
		"run: ayla run [--debug] [--timed] <file>, runs the ayla script",
		"build: ayla build <file> [-o <output>], turns the ayla script into a standalone executable",
		"install: ayla run install <url>, installs an ayla module and makes it global",
		"--version: ayla --version, returns the current version",
		"--help: ayla --help, returns all the available commands",
	}

	if len(os.Args) == 1 {
		fmt.Println("Welcome to ayla-lang v1.5.0, do ayla --help to see all commands.")
		repl()
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

	case "build":
		if len(os.Args) < 3 {
			fmt.Println("usage ayla build <file>")
			return
		}

		build()

	case "--version":
		fmt.Println("ayla-lang v1.5.0")

	case "--help":
		fmt.Println(strings.Join(cmds, "\n"))

	default:
		fmt.Println("unknown command: " + os.Args[1] + ", use --help if you need to see the available commands")
	}
}

func repl() {
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

func runEmbedded(source string) {
	exe, err := os.Executable()
	if err != nil {
		fmt.Println(err)
		return
	}

	l := lexer.New(source)
	p := parser.New(l)

	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			fmt.Println(err)
		}
		return
	}

	interp := interpreter.New(exe)

	if err := interp.RegisterForward(program); err != nil {
		fmt.Println(err)
		return
	}

	if err := interp.ResolveTypes(program); err != nil {
		fmt.Println(err)
		return
	}

	_, err = interp.EvalStatements(program)
	if err != nil {
		fmt.Println(err)
	}
}

func build() {
	args := os.Args[2:]

	filename := ""
	output := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "-o":
			if i+1 >= len(args) {
				fmt.Println("Expected filename after -o")
				return
			}
			output = args[i+1]
			i++

		default:
			filename = arg
		}
	}

	if filename == "" {
		fmt.Println("No input file provided")
		return
	}

	if output == "" {
		base := filepath.Base(filename)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		output = name + ".exe"
	}

	src, err := readSourceFile(filename)
	if err != nil {
		fmt.Println(err)
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		fmt.Println(err)
		return
	}

	data, err := os.ReadFile(exePath)
	if err != nil {
		fmt.Println(err)
		return
	}

	startMarker := []byte("\n__AYLA_SCRIPT_START__\n")
	endMarker := []byte("\n__AYLA_SCRIPT_END__\n")

	start := bytes.LastIndex(data, startMarker)
	end := bytes.LastIndex(data, endMarker)

	if start != -1 && end != -1 && end > start {
		data = data[:start]
	}

	out, err := os.Create(output)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer out.Close()

	out.Write(data)
	out.Write(startMarker)
	out.Write([]byte(src))
	out.Write(endMarker)

	fmt.Println("built executable:", output)
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
