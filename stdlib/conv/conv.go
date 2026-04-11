package conv

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/registry"
)

func init() {
	registry.Register("conv", Load)
}

func Load(i *interpreter.Interpreter) (interpreter.ModuleValue, error) {
	env := interpreter.NewEnvironment(i.Env)
	typeEnv := make(map[string]interpreter.TypeValue)

	env.Define("Int", &interpreter.BuiltinFunc{
		Name:  "Int",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v := interpreter.UnwrapFully(args[0])

			ti := interpreter.UnwrapAlias(i.TypeInfoFromValue(v))

			switch ti.Kind {
			case interpreter.TypeInt:
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						v,
						interpreter.NilValue{},
					},
				}, nil
			case interpreter.TypeFloat:
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.IntValue{V: int(v.(interpreter.FloatValue).V)},
						interpreter.NilValue{},
					},
				}, nil
			case interpreter.TypeString:
				n, err := strconv.Atoi(v.(interpreter.StringValue).V)
				if err != nil {
					return interpreter.TupleValue{
						Values: []interpreter.Value{
							interpreter.IntValue{V: n},
							interpreter.Error{Message: fmt.Sprintf("conv.Int: %s", err.Error())},
						},
					}, nil
				}

				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.IntValue{V: n},
						interpreter.NilValue{},
					},
				}, nil
			case interpreter.TypeBool:
				if v.(interpreter.BoolValue).V {
					return interpreter.TupleValue{
						Values: []interpreter.Value{
							interpreter.IntValue{V: 1},
							interpreter.NilValue{},
						},
					}, nil
				}

				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.IntValue{V: 0},
						interpreter.NilValue{},
					},
				}, nil
			default:
				return interpreter.NilValue{}, interpreter.NewRuntimeError(node, fmt.Sprintf("conv.Int: cannot convert type '%s' to int", i.TypeInfoFromValue(args[0]).Name))
			}
		},
	}, false)

	env.Define("Float", &interpreter.BuiltinFunc{
		Name:  "Float",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v := interpreter.UnwrapFully(args[0])

			ti := interpreter.UnwrapAlias(i.TypeInfoFromValue(v))

			switch ti.Kind {
			case interpreter.TypeFloat:
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						v,
						interpreter.NilValue{},
					},
				}, nil
			case interpreter.TypeInt:
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.FloatValue{V: float64(v.(interpreter.IntValue).V)},
						interpreter.NilValue{},
					},
				}, nil
			case interpreter.TypeString:
				n, err := strconv.ParseFloat(v.(interpreter.StringValue).V, 64)
				if err != nil {
					return interpreter.TupleValue{
						Values: []interpreter.Value{
							interpreter.NilValue{},
							interpreter.Error{Message: fmt.Sprintf("conv.Float: %s", err.Error())},
						},
					}, nil
				}

				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.FloatValue{V: n},
						interpreter.NilValue{},
					},
				}, nil
			case interpreter.TypeBool:
				if v.(interpreter.BoolValue).V {
					return interpreter.TupleValue{
						Values: []interpreter.Value{
							interpreter.FloatValue{V: 1},
							interpreter.NilValue{},
						},
					}, nil
				}

				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.FloatValue{V: 0},
						interpreter.NilValue{},
					},
				}, nil
			default:
				return interpreter.NilValue{}, interpreter.NewRuntimeError(node, fmt.Sprintf("conv.Float: cannot convert type '%s' to float", i.TypeInfoFromValue(args[0]).Name))
			}
		},
	}, false)

	env.Define("String", &interpreter.BuiltinFunc{
		Name:  "String",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v := interpreter.UnwrapFully(args[0])

			ti := interpreter.UnwrapAlias(i.TypeInfoFromValue(v))

			switch ti.Kind {
			case interpreter.TypeString:
				return v, nil
			case interpreter.TypeInt:
				s := strconv.Itoa(v.(interpreter.IntValue).V)

				return interpreter.StringValue{V: s}, nil
			case interpreter.TypeFloat:
				s := strconv.FormatFloat(v.(interpreter.FloatValue).V, 'g', -1, 64)
				return interpreter.StringValue{V: s}, nil
			case interpreter.TypeBool:
				if v.(interpreter.BoolValue).V {
					return interpreter.StringValue{V: "yes"}, nil
				}

				return interpreter.StringValue{V: "no"}, nil
			default:
				return interpreter.NilValue{}, interpreter.NewRuntimeError(node, fmt.Sprintf("conv.String: cannot convert type '%s' to string", i.TypeInfoFromValue(args[0]).Name))
			}
		},
	}, false)

	env.Define("Bool", &interpreter.BuiltinFunc{
		Name:  "Bool",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v := interpreter.UnwrapFully(args[0])

			ti := interpreter.UnwrapAlias(i.TypeInfoFromValue(v))

			switch ti.Kind {
			case interpreter.TypeBool:
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						v,
						interpreter.NilValue{},
					},
				}, nil
			case interpreter.TypeInt:
				if v.(interpreter.IntValue).V != 0 {
					return interpreter.TupleValue{
						Values: []interpreter.Value{
							interpreter.BoolValue{V: true},
							interpreter.NilValue{},
						},
					}, nil
				}

				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.BoolValue{V: false},
						interpreter.NilValue{},
					},
				}, nil
			case interpreter.TypeFloat:
				if v.(interpreter.FloatValue).V != 0 {
					return interpreter.TupleValue{
						Values: []interpreter.Value{
							interpreter.BoolValue{V: true},
							interpreter.NilValue{},
						},
					}, nil
				}

				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.BoolValue{V: false},
						interpreter.NilValue{},
					},
				}, nil
			case interpreter.TypeString:
				s := strings.ToLower(v.(interpreter.StringValue).V)

				if s == "yes" {
					return interpreter.TupleValue{
						Values: []interpreter.Value{
							interpreter.BoolValue{V: true},
							interpreter.NilValue{},
						},
					}, nil
				}
				if s == "no" {
					return interpreter.TupleValue{
						Values: []interpreter.Value{
							interpreter.BoolValue{V: false},
							interpreter.NilValue{},
						},
					}, nil
				}
				if s == "true" {
					return interpreter.TupleValue{
						Values: []interpreter.Value{
							interpreter.BoolValue{V: true},
							interpreter.NilValue{},
						},
					}, nil
				}
				if s == "false" {
					return interpreter.TupleValue{
						Values: []interpreter.Value{
							interpreter.BoolValue{V: false},
							interpreter.NilValue{},
						},
					}, nil
				}

				b, err := strconv.ParseBool(s)
				if err != nil {
					return interpreter.TupleValue{
						Values: []interpreter.Value{
							interpreter.NilValue{},
							interpreter.Error{Message: "conv.Bool: invalid boolean string"},
						},
					}, nil
				}

				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.BoolValue{V: b},
						interpreter.NilValue{},
					},
				}, nil
			default:
				return interpreter.NilValue{}, interpreter.NewRuntimeError(node, fmt.Sprintf("conv.Bool: cannot convert type '%s' to bool", i.TypeInfoFromValue(args[0]).Name))
			}
		},
	}, false)

	mod := interpreter.ModuleValue{
		Name:    "conv",
		Env:     env,
		TypeEnv: typeEnv,
	}

	return mod, nil
}
