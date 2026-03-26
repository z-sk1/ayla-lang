package time

import (
	"time"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/registry"
)

func init() {
	registry.Register("time", LoadTimeModule)
}

func LoadTimeModule(i *interpreter.Interpreter) (interpreter.ModuleValue, error) {
	env := interpreter.NewEnvironment(i.Env)

	env.Define("Sleep", &interpreter.BuiltinFunc{
		Name:  "Sleep",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			var seconds float64

			switch v := interpreter.UnwrapFully(args[0]).(type) {
			case interpreter.IntValue:
				seconds = float64(v.V)
			case interpreter.FloatValue:
				seconds = v.V
			default:
				return interpreter.NilValue{}, interpreter.NewRuntimeError(node, "time.Sleep: argument must be number")
			}

			time.Sleep(time.Duration(seconds * float64(time.Second)))
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("Now", &interpreter.BuiltinFunc{
		Name:  "Now",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return interpreter.IntValue{V: int(time.Now().Unix())}, nil
		},
	}, false)

	env.Define("Since", &interpreter.BuiltinFunc{
		Name:  "Since",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			start, err := interpreter.ArgInt(node, args, 0, "time.Since")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			now := time.Now().Unix()
			return interpreter.IntValue{V: int(now - int64(start))}, nil
		},
	}, false)

	env.Define("Format", &interpreter.BuiltinFunc{
		Name:  "Format",
		Arity: -1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			switch len(args) {
			case 1:
				ts, err := interpreter.ArgInt(node, args, 0, "time.Format")
				if err != nil {
					return interpreter.NilValue{}, err
				}

				t := time.Unix(int64(ts), 0)
				s := t.Format("2006-01-02 15:04:05")

				return interpreter.StringValue{V: s}, nil
			case 2:
				ts, err := interpreter.ArgInt(node, args, 0, "time.Format")
				if err != nil {
					return interpreter.NilValue{}, err
				}

				layout, err := interpreter.ArgString(node, args, 1, "time.Format")
				if err != nil {
					return interpreter.NilValue{}, err
				}

				t := time.Unix(int64(ts), 0)
				s := t.Format(layout)

				return interpreter.StringValue{V: s}, nil
			default:
				return interpreter.NilValue{}, interpreter.NewRuntimeError(node, "time.Format: invalid amount of args, 0-2 args allowed")
			}
		},
	}, false)

	module := interpreter.ModuleValue{
		Name: "time",
		Env:  env,
	}

	return module, nil
}
