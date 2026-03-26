package rand

import (
	"math/rand"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/registry"
)

func init() {
	registry.Register("rand", LoadRandModule)
}

func LoadRandModule(i *interpreter.Interpreter) (interpreter.ModuleValue, error) {
	env := interpreter.NewEnvironment(i.Env)

	env.Define("Int", &interpreter.BuiltinFunc{
		Name:  "Int",
		Arity: -1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			switch len(args) {
			case 0:
				n := rand.Intn(2)
				return interpreter.IntValue{V: n}, nil
			case 1:
				max, err := interpreter.ArgInt(node, args, 0, "rand.Int")
				if err != nil {
					return interpreter.NilValue{}, err
				}

				if max <= 0 {
					return interpreter.NilValue{}, interpreter.NewRuntimeError(node, "rand.Int: first argument must > 0")
				}

				n := rand.Intn(max) + 1
				return interpreter.IntValue{V: n}, nil
			case 2:
				min, err := interpreter.ArgInt(node, args, 0, "rand.Int")
				if err != nil {
					return interpreter.NilValue{}, err
				}

				max, err := interpreter.ArgInt(node, args, 1, "rand.Int")
				if err != nil {
					return interpreter.NilValue{}, err
				}

				if min > max {
					min, max = max, min
				}

				n := rand.Intn(max-min+1) + min
				return interpreter.IntValue{V: n}, nil
			}
			return interpreter.NilValue{}, interpreter.ExpectArgsRange(node, args, 0, 2, "rand.Int")
		},
	}, false)

	env.Define("Float", &interpreter.BuiltinFunc{
		Name:  "Float",
		Arity: -1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			switch len(args) {
			case 0:
				n := rand.Float64()
				return interpreter.FloatValue{V: n}, nil
			case 1:
				max, err := interpreter.ArgFloat(node, args, 0, "rand.Float")
				if err != nil {
					return interpreter.NilValue{}, err
				}

				n := rand.Float64() * max
				return interpreter.FloatValue{V: n}, nil
			case 2:
				min, err := interpreter.ArgFloat(node, args, 0, "rand.Float")
				if err != nil {
					return interpreter.NilValue{}, err
				}

				max, err := interpreter.ArgFloat(node, args, 1, "rand.Float")
				if err != nil {
					return interpreter.NilValue{}, err
				}

				if min > max {
					min, max = max, min
				}

				n := rand.Float64()*(max-min+1) + min
				return interpreter.FloatValue{V: n}, nil
			}
			return interpreter.NilValue{}, interpreter.ExpectArgsRange(node, args, 0, 2, "rand.Float")
		},
	}, false)

	env.Define("Choice", &interpreter.BuiltinFunc{
		Name:  "Choice",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			arr, err := interpreter.ArgArray(node, args, 0, "rand.Choice")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rand := rand.Intn(len(arr.Elements))
			return arr.Elements[rand], nil
		},
	}, false)

	module := interpreter.ModuleValue{
		Name: "rand",
		Env:  env,
	}

	return module, nil
}
