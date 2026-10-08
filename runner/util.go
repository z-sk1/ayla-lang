package runner

import (
	"fmt"
	"os"
	"strings"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/parser"
)

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

	if err := interp.TypeCheck(program); err != nil {
		fmt.Println(err)
		return
	}

	_, err = interp.EvalStatements(program)
	if err != nil {
		fmt.Println(err)
	}
}

func normalizeGitHubURL(url string) string {
	if strings.Contains(url, "github.com") && !strings.Contains(url, "raw.githubusercontent.com") {
		url = strings.Replace(url, "github.com", "raw.githubusercontent.com", 1)
		url = strings.Replace(url, "/blob/", "/", 1)
	}
	return url
}

func readSourceFile(name string) (string, string, error) {
	candidates := []string{
		name,
		name + ".ayl",
		name + ".ayla",
	}

	for _, file := range candidates {
		data, err := os.ReadFile(file)
		if err == nil {
			return string(data), file, nil
		}
	}

	return "", "", fmt.Errorf("file not found: %s (.ayla or .ayl)", name)
}