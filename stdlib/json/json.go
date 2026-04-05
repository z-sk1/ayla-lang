package json

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/registry"
)

func init() {
	registry.Register("json", Load)
}

func Load(i *interpreter.Interpreter) (interpreter.ModuleValue, error) {
	env := interpreter.NewEnvironment(i.Env)

	env.Define("Parse", &interpreter.BuiltinFunc{
		Name:  "Parse",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			str, err := interpreter.ArgString(node, args, 0, "json.Parse")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			var raw any

			if err := json.Unmarshal([]byte(str), &raw); err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.Error{Message: err.Error()},
					},
				}, nil
			}

			return interpreter.TupleValue{
				Values: []interpreter.Value{
					jsonToAyla(i, raw),
					interpreter.NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("ParseFile", &interpreter.BuiltinFunc{
		Name:  "ParseFile",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "json.ParseFile")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.Error{Message: err.Error()},
					},
				}, nil
			}

			var raw any

			if err := json.Unmarshal(data, &raw); err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.Error{Message: err.Error()},
					},
				}, nil
			}

			return interpreter.TupleValue{
				Values: []interpreter.Value{
					jsonToAyla(i, raw),
					interpreter.NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("Stringify", &interpreter.BuiltinFunc{
		Name:  "Stringify",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			raw, err := aylaToJSON(i, args[0], "json.Stringify")
			if err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.Error{Message: err.Error()},
					},
				}, nil
			}

			bytes, err := json.Marshal(raw)
			if err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.Error{Message: err.Error()},
					},
				}, nil
			}

			return interpreter.TupleValue{
				Values: []interpreter.Value{
					interpreter.StringValue{V: string(bytes)},
					interpreter.NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("StringifyPretty", &interpreter.BuiltinFunc{
		Name:  "StringifyPretty",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			raw, err := aylaToJSON(i, args[0], "json.StringifyPretty")
			if err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.Error{Message: err.Error()},
					},
				}, nil
			}

			indent, err := interpreter.ArgInt(node, args, 1, "json.StringifyPretty")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			bytes, err := json.MarshalIndent(raw, "", strings.Repeat(" ", indent))
			if err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.Error{Message: err.Error()},
					},
				}, nil
			}

			return interpreter.TupleValue{
				Values: []interpreter.Value{
					interpreter.StringValue{V: string(bytes)},
					interpreter.NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("IsValid", &interpreter.BuiltinFunc{
		Name:  "IsValid",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			str, err := interpreter.ArgString(node, args, 0, "json.IsValid")
			if err != nil {
				return interpreter.NilValue{}, err
			}
			return interpreter.BoolValue{V: json.Valid([]byte(str))}, nil
		},
	}, false)

	env.Define("Get", &interpreter.BuiltinFunc{
		Name:  "Get",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			key, err := interpreter.ArgString(node, args, 1, "json.Get")
			if err != nil {
				return interpreter.NilValue{}, err
			}
			m, ok := interpreter.UnwrapFully(args[0]).(interpreter.MapValue)
			if !ok {
				return interpreter.NilValue{}, interpreter.NewRuntimeError(node, "json.Get: expected a map")
			}
			val, ok := m.Entries["s:"+key]
			if !ok {
				return interpreter.NilValue{}, nil
			}
			return val, nil
		},
	}, false)

	env.Define("Has", &interpreter.BuiltinFunc{
		Name:  "Has",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			key, err := interpreter.ArgString(node, args, 1, "json.Has")
			if err != nil {
				return interpreter.NilValue{}, err
			}
			m, ok := interpreter.UnwrapFully(args[0]).(interpreter.MapValue)
			if !ok {
				return interpreter.BoolValue{V: false}, nil
			}
			_, ok = m.Entries["s:"+key]
			return interpreter.BoolValue{V: ok}, nil
		},
	}, false)

	env.Define("GetIndex", &interpreter.BuiltinFunc{
		Name:  "GetIndex",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			arr, err := interpreter.ArgArray(node, args, 0, "json.GetIndex", "")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			idx, err := interpreter.ArgInt(node, args, 1, "json.GetIndex")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			if idx < 0 || idx >= len(arr.Elements) {
				return interpreter.NilValue{}, interpreter.NewRuntimeError(node, fmt.Sprintf("json.GetIndex: index %d out of bounds", idx))
			}
			return arr.Elements[idx], nil
		},
	}, false)

	module := interpreter.ModuleValue{
		Name: "json",
		Env:  env,
	}

	return module, nil
}

func jsonToAyla(i *interpreter.Interpreter, v any) interpreter.Value {
	if v == nil {
		return interpreter.NilValue{}
	}
	switch val := v.(type) {
	case bool:
		return interpreter.BoolValue{V: val}
	case float64:
		if val == float64(int(val)) {
			return interpreter.IntValue{V: int(val)}
		}
		return interpreter.FloatValue{V: val}
	case string:
		return interpreter.StringValue{V: val}
	case []any:
		elements := make([]interpreter.Value, len(val))
		for idx, el := range val {
			elements[idx] = jsonToAyla(i, el)
		}
		return interpreter.ArrayValue{
			Elements: elements,
			ElemType: i.TypeEnv["thing"].TypeInfo,
		}
	case map[string]any:
		entries := map[string]interpreter.Value{}
		keys := map[string]interpreter.Value{}
		for k, v := range val {
			key := interpreter.StringValue{V: k}
			entries[interpreter.MapKey(key)] = jsonToAyla(i, v)
			keys[interpreter.MapKey(key)] = key
		}
		return interpreter.MapValue{
			Entries:   entries,
			Keys:      keys,
			KeyType:   i.TypeEnv["string"].TypeInfo,
			ValueType: i.TypeEnv["thing"].TypeInfo,
		}
	}
	return interpreter.NilValue{}
}

func aylaToJSON(i *interpreter.Interpreter, v interpreter.Value, name string) (any, error) {
	v = interpreter.UnwrapFully(v)
	switch val := v.(type) {
	case interpreter.NilValue:
		return nil, nil
	case interpreter.BoolValue:
		return val.V, nil
	case interpreter.IntValue:
		return val.V, nil
	case interpreter.FloatValue:
		return val.V, nil
	case interpreter.StringValue:
		return val.V, nil
	case interpreter.ArrayValue:
		result := make([]any, len(val.Elements))
		for idx, el := range val.Elements {
			converted, err := aylaToJSON(i, el, name)
			if err != nil {
				return nil, err
			}
			result[idx] = converted
		}
		return result, nil
	case interpreter.MapValue:
		result := map[string]any{}
		for k, v := range val.Entries {
			converted, err := aylaToJSON(i, v, name)
			if err != nil {
				return nil, err
			}
			// strip the "s:" prefix from map keys
			key := val.Keys[k]
			keyStr, ok := key.(interpreter.StringValue)
			if !ok {
				return nil, fmt.Errorf("%s: map keys must be strings", name)
			}
			result[keyStr.V] = converted
		}
		return result, nil
	case *interpreter.StructValue:
		result := map[string]any{}
		for name, field := range val.Fields {
			converted, err := aylaToJSON(i, field, name)
			if err != nil {
				return nil, err
			}
			result[name] = converted
		}
		return result, nil
	default:
		return nil, fmt.Errorf("%s: unsupported type %s", name, i.TypeInfoFromValue(v).Name)
	}
}
