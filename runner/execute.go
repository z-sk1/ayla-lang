package runner

import (
	"bytes"
	"errors"
	"strings"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/parser"
)

func Execute(source string) (string, error) {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		strErrs := []string{}
		var err error
		for _, e := range p.Errors() {
			strErrs = append(strErrs, e.Error())
			err = errors.New(strings.Join(strErrs, ", "))
		}
		return "", err
	}

	var buf bytes.Buffer

	interp := interpreter.New("<playground>")
	interp.Stdout = &buf

	if err := interp.RegisterForward(program); err != nil {
		return "", err
	}
	if err := interp.ResolveTypes(program); err != nil {
		return "", err
	}
	if err := interp.TypeCheck(program); err != nil {
		return "", err
	}

	_, err := interp.EvalStatements(program)
	if err != nil {
		return "", err
	}

	interp.Wg.Wait()
	return buf.String(), nil
}
