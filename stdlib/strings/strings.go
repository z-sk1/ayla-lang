package strings

import (
	"strings"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/registry"
)

func init() {
	registry.Register("strings", Load)
}

func Load(i *interpreter.Interpreter) (interpreter.ModuleValue, error) {
	env := interpreter.NewEnvironment(i.Env)

	env.Define("Upper", interpreter.WrapString1("strings.Upper", strings.ToUpper), false)
	env.Define("Lower", interpreter.WrapString1("strings.Lower", strings.ToLower), false)
	env.Define("Contains", &interpreter.BuiltinFunc{
		Name:  "Contains",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			s, err := interpreter.ArgString(node, args, 0, "strings.Contains")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sub, err := interpreter.ArgString(node, args, 1, "strings.Contains")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: strings.Contains(s, sub)}, nil
		}}, false)
	env.Define("ContainsAny", &interpreter.BuiltinFunc{
		Name:  "ContainsAny",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			s, err := interpreter.ArgString(node, args, 0, "strings.ContainsAny")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sub, err := interpreter.ArgString(node, args, 1, "strings.ContainsAny")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: strings.ContainsAny(s, sub)}, nil
		}}, false)
	env.Define("Join", interpreter.WrapSliceStringRString("strings.Join", strings.Join), false)
	env.Define("Split", interpreter.WrapString2RSlice("strings.Split", strings.Split), false)
	env.Define("SplitN", interpreter.WrapString2IntRSlice("strings.SplitN", strings.SplitN), false)
	env.Define("TrimSpace", interpreter.WrapString1("strings.TrimSpace", strings.TrimSpace), false)
	env.Define("TrimPrefix", interpreter.WrapString2("strings.TrimPrefix", strings.TrimPrefix), false)
	env.Define("TrimSuffix", interpreter.WrapString2("strings.TrimSuffix", strings.TrimSuffix), false)
	env.Define("HasPrefix", &interpreter.BuiltinFunc{
		Name:  "HasPrefix",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			s, err := interpreter.ArgString(node, args, 0, "strings.HasPrefix")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sub, err := interpreter.ArgString(node, args, 1, "strings.HasPrefix")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: strings.HasPrefix(s, sub)}, nil
		}}, false)
	env.Define("HasSuffix", &interpreter.BuiltinFunc{
		Name:  "HasSuffix",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			s, err := interpreter.ArgString(node, args, 0, "strings.HasSuffix")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sub, err := interpreter.ArgString(node, args, 1, "strings.HasSuffix")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: strings.HasSuffix(s, sub)}, nil
		}}, false)
	env.Define("Index", interpreter.WrapString2RInt("strings.Index", strings.Index), false)
	env.Define("LastIndex", interpreter.WrapString2RInt("strings.LastIndex", strings.LastIndex), false)
	env.Define("Count", interpreter.WrapString2RInt("strings.Count", strings.Count), false)
	env.Define("Fields", interpreter.WrapString1RSlice("strings.Fields", strings.Fields), false)
	env.Define("Replace", &interpreter.BuiltinFunc{
		Name:  "Replace",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			s, err := interpreter.ArgString(node, args, 0, "strings.Replace")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			old, err := interpreter.ArgString(node, args, 1, "strings.Replace")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			new, err := interpreter.ArgString(node, args, 2, "strings.Replace")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			n, err := interpreter.ArgInt(node, args, 3, "strings.Replace")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.StringValue{V: strings.Replace(s, old, new, n)}, nil
		}}, false)
	env.Define("ReplaceAll", &interpreter.BuiltinFunc{
		Name:  "Replace",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			s, err := interpreter.ArgString(node, args, 0, "strings.ReplaceAll")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			old, err := interpreter.ArgString(node, args, 1, "strings.ReplaceAll")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			new, err := interpreter.ArgString(node, args, 2, "strings.ReplaceAll")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.StringValue{V: strings.ReplaceAll(s, old, new)}, nil
		}}, false)
	env.Define("Cut", &interpreter.BuiltinFunc{
		Name:  "Cut",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			s, err := interpreter.ArgString(node, args, 0, "strings.Cut")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sep, err := interpreter.ArgString(node, args, 1, "strings.Cut")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			bef, aft, found := strings.Cut(s, sep)

			return interpreter.TupleValue{
				Values: []interpreter.Value{
					interpreter.StringValue{V: bef},
					interpreter.StringValue{V: aft},
					interpreter.BoolValue{V: found},
				},
			}, nil
		}}, false)
	env.Define("Repeat", &interpreter.BuiltinFunc{
		Name:  "Repeat",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			s, err := interpreter.ArgString(node, args, 0, "strings.Repeat")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			count, err := interpreter.ArgInt(node, args, 1, "strings.Repeat")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.StringValue{V: strings.Repeat(s, count)}, nil
		}}, false)

	module := interpreter.ModuleValue{
		Name: "strings",
		Env:  env,
	}

	return module, nil
}
