package interpreter

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/z-sk1/ayla-lang/parser"
)

type NativeLoader func(i *Interpreter) (ModuleValue, error)

func expectArgsRange(node parser.Node, args []Value, startRange, endRange int, name string) error {
	return NewRuntimeError(node, fmt.Sprintf("%s: expected %d-%d arguments, got %d", name, startRange, endRange, len(args)))
}

func argInt(node parser.Node, args []Value, i int, name string) (int, error) {
	v := unwrapNamed(args[i])
	v = unwrapUntyped(v)
	iv, ok := v.(IntValue)
	if !ok {
		return 0, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be an int", name, i+1))
	}
	return iv.V, nil
}

func argFloat(node parser.Node, args []Value, i int, name string) (float64, error) {
	v, ok := toFloat(unwrapUntyped(unwrapNamed(args[i])))
	if !ok {
		return 0, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be an float", name, i+1))
	}
	return v, nil
}

func argString(node parser.Node, args []Value, i int, name string) (string, error) {
	v := unwrapNamed(args[i])
	v = unwrapUntyped(v)
	iv, ok := v.(StringValue)
	if !ok {
		return "", NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a string", name, i+1))
	}
	return iv.V, nil
}

func argBool(node parser.Node, args []Value, i int, name string) (bool, error) {
	v := unwrapNamed(args[i])
	v = unwrapUntyped(v)
	iv, ok := v.(BoolValue)
	if !ok {
		return false, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a boolean", name, i+1))
	}
	return iv.V, nil
}

func argStruct(node parser.Node, args []Value, i int, name, sname string) (*StructValue, error) {
	v := unwrapNamed(args[i])
	sv, ok := v.(*StructValue)
	if !ok {
		return nil, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a %s", name, i+1, sname))
	}
	return sv, nil
}

func argType(node parser.Node, args []Value, i int, name string) (TypeValue, error) {
	v := unwrapNamed(args[i])
	tv, ok := v.(TypeValue)
	if !ok {
		return TypeValue{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a type signature", name, i+1))
	}
	return tv, nil
}

func argPointer(node parser.Node, args []Value, i int, name string) (*PointerValue, error) {
	v := unwrapNamed(args[i])
	pv, ok := v.(*PointerValue)
	if !ok {
		return &PointerValue{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a pointer", name, i+1))
	}
	return pv, nil
}

func argArray(node parser.Node, args []Value, i int, name string) (ArrayValue, error) {
	v := unwrapNamed(args[i])
	av, ok := v.(ArrayValue)
	if !ok {
		return ArrayValue{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be an array or slice", name, i+1))
	}
	return av, nil
}

func argColor(node parser.Node, typeEnv map[string]TypeValue, args []Value, i int, name string) (rl.Color, error) {
	colTI := typeEnv["Color"].TypeInfo

	sv, err := argStruct(node, args, i, name, "gfx.Color")
	if err != nil {
		return rl.Color{}, err
	}

	if !typesAssignable(sv.TypeName, colTI) {
		return rl.Color{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be gfx.Color", name, i+1))
	}

	return colorFromValue(sv)
}

func argVector2(node parser.Node, i *Interpreter, typeEnv map[string]TypeValue, args []Value, idx int, name string) (rl.Vector2, error) {
	vecTI := typeEnv["Vector2"].TypeInfo

	v := unwrapNamed(args[idx])

	vecVal, ok := v.(*StructValue)
	if !ok {
		return rl.Vector2{}, NewRuntimeError(node, name+": argument must be gfx.Vector2")
	}

	if !typesAssignable(i.typeInfoFromValue(vecVal), vecTI) {
		return rl.Vector2{}, NewRuntimeError(node, name+": argument must be gfx.Vector2")
	}

	x, _ := toFloat(unwrapUntyped(vecVal.Fields["X"]))
	y, _ := toFloat(unwrapUntyped(vecVal.Fields["Y"]))

	return rl.Vector2{
		X: float32(x),
		Y: float32(y),
	}, nil
}

func colorFromValue(v Value) (rl.Color, error) {
	colVal := v.(*StructValue)

	r := colVal.Fields["R"].(IntValue).V
	g := colVal.Fields["G"].(IntValue).V
	b := colVal.Fields["B"].(IntValue).V
	a := colVal.Fields["A"].(IntValue).V

	return rl.Color{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
		A: uint8(a),
	}, nil
}

func unwrapVector2(v Value) (float32, float32) {
	x, ok := toFloat(v.(*StructValue).Fields["X"])
	if !ok {
		return 0, 0
	}

	y, ok := toFloat(v.(*StructValue).Fields["Y"])
	if !ok {
		return 0, 0
	}

	return float32(x), float32(y)
}

func wrapFloat1(name string, fn func(float64) float64) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			f, err := argFloat(node, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			return FloatValue{V: fn(f)}, nil
		},
	}
}

func wrapFloat2(name string, fn func(float64, float64) float64) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			f1, err := argFloat(node, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			f2, err := argFloat(node, args, 1, name)
			if err != nil {
				return NilValue{}, err
			}

			return FloatValue{V: fn(f1, f2)}, nil
		},
	}
}

func LoadMathModule(i *Interpreter) (ModuleValue, error) {
	env := NewEnvironment(i.Env)

	// functions
	env.Define("Abs", wrapFloat1("Abs", math.Abs), false)
	env.Define("Sin", wrapFloat1("Sin", math.Sin), false)
	env.Define("Asin", wrapFloat1("Asin", math.Asin), false)
	env.Define("Cos", wrapFloat1("Cos", math.Cos), false)
	env.Define("Acos", wrapFloat1("Acos", math.Acos), false)
	env.Define("Tan", wrapFloat1("Tan", math.Tan), false)
	env.Define("Atan", wrapFloat1("Atan", math.Atan), false)
	env.Define("Sqrt", wrapFloat1("Sqrt", math.Sqrt), false)
	env.Define("Log", wrapFloat1("Log", math.Log), false)
	env.Define("Exp", wrapFloat1("Exp", math.Exp), false)
	env.Define("Floor", wrapFloat1("Floor", math.Floor), false)
	env.Define("Ceil", wrapFloat1("Ceil", math.Ceil), false)
	env.Define("Round", wrapFloat1("Round", math.Round), false)
	env.Define("RoundToEven", wrapFloat1("RoundToEven", math.RoundToEven), false)
	env.Define("Trunc", wrapFloat1("Trunc", math.Trunc), false)

	env.Define("Max", wrapFloat2("Max", math.Max), false)
	env.Define("Min", wrapFloat2("Min", math.Min), false)
	env.Define("Pow", wrapFloat2("Pow", math.Pow), false)
	env.Define("Remainder", wrapFloat2("Remainder", math.Remainder), false)
	env.Define("Atan2", wrapFloat2("Atan2", math.Atan2), false)

	// constants
	env.Define("Pi", FloatValue{V: math.Pi}, true)
	env.Define("Phi", FloatValue{V: math.Phi}, true)
	env.Define("E", FloatValue{V: math.E}, true)
	env.Define("Sqrt2", FloatValue{V: math.Sqrt2}, true)
	env.Define("SqrtPi", FloatValue{V: math.SqrtPi}, true)
	env.Define("SqrtPhi", FloatValue{V: math.SqrtPhi}, true)
	env.Define("SqrtE", FloatValue{V: math.SqrtE}, true)

	module := ModuleValue{
		Name: "math",
		Env:  env,
	}

	return module, nil
}

func LoadRandModule(i *Interpreter) (ModuleValue, error) {
	env := NewEnvironment(i.Env)

	env.Define("Int", &BuiltinFunc{
		Name:  "Int",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			switch len(args) {
			case 0:
				n := rand.Intn(2)
				return IntValue{V: n}, nil
			case 1:
				max, err := argInt(node, args, 0, "rand.Int")
				if err != nil {
					return NilValue{}, err
				}

				if max <= 0 {
					return NilValue{}, NewRuntimeError(node, "rand.Int: first argument must > 0")
				}

				n := rand.Intn(max) + 1
				return IntValue{V: n}, nil
			case 2:
				min, err := argInt(node, args, 0, "rand.Int")
				if err != nil {
					return NilValue{}, err
				}

				max, err := argInt(node, args, 1, "rand.Int")
				if err != nil {
					return NilValue{}, err
				}

				if min > max {
					min, max = max, min
				}

				n := rand.Intn(max-min+1) + min
				return IntValue{V: n}, nil
			}
			return NilValue{}, expectArgsRange(node, args, 0, 2, "rand.Int")
		},
	}, false)

	env.Define("Float", &BuiltinFunc{
		Name:  "Float",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			switch len(args) {
			case 0:
				n := rand.Float64()
				return FloatValue{V: n}, nil
			case 1:
				max, err := argFloat(node, args, 0, "rand.Float")
				if err != nil {
					return NilValue{}, err
				}

				n := rand.Float64() * max
				return FloatValue{V: n}, nil
			case 2:
				min, err := argFloat(node, args, 0, "rand.Float")
				if err != nil {
					return NilValue{}, err
				}

				max, err := argFloat(node, args, 1, "rand.Float")
				if err != nil {
					return NilValue{}, err
				}

				if min > max {
					min, max = max, min
				}

				n := rand.Float64()*(max-min+1) + min
				return FloatValue{V: n}, nil
			}
			return NilValue{}, expectArgsRange(node, args, 0, 2, "rand.Float")
		},
	}, false)

	env.Define("Choice", &BuiltinFunc{
		Name:  "Choice",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			arr, err := argArray(node, args, 0, "rand.Choice")
			if err != nil {
				return NilValue{}, err
			}

			rand := rand.Intn(len(arr.Elements))
			return arr.Elements[rand], nil
		},
	}, false)

	module := ModuleValue{
		Name: "rand",
		Env:  env,
	}

	return module, nil
}

func LoadFSModule(i *Interpreter) (ModuleValue, error) {
	env := NewEnvironment(i.Env)

	env.Define("Create", &BuiltinFunc{
		Name:  "Create",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			path, err := argString(node, args, 0, "fs.Create")
			if err != nil {
				return NilValue{}, err
			}

			dir := filepath.Dir(path)

			err = os.MkdirAll(dir, 0755)
			if err != nil {
				return InterfaceValue{
					TypeInfo: i.typeEnv["error"].TypeInfo,
					Value:    Error{Message: err.Error()},
				}, nil
			}

			_, err = os.Create(path)
			if err != nil {
				return InterfaceValue{
					TypeInfo: i.typeEnv["error"].TypeInfo,
					Value:    Error{Message: err.Error()},
				}, nil
			}

			return NilValue{}, nil
		},
	}, false)

	env.Define("Write", &BuiltinFunc{
		Name:  "Write",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			path, err := argString(node, args, 0, "fs.Write")
			if err != nil {
				return NilValue{}, err
			}

			content, err := argString(node, args, 1, "fs.Write")
			if err != nil {
				return NilValue{}, err
			}

			err = os.WriteFile(path, []byte(content), 0644)
			if err != nil {
				return InterfaceValue{
					TypeInfo: i.typeEnv["error"].TypeInfo,
					Value:    Error{Message: err.Error()},
				}, nil
			}

			return NilValue{}, nil
		},
	}, false)

	env.Define("Read", &BuiltinFunc{
		Name:  "Read",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {

			path, err := argString(node, args, 0, "fs.Read")

			data, err := os.ReadFile(path)
			if err != nil {
				return TupleValue{
					Values: []Value{
						NilValue{},
						InterfaceValue{
							TypeInfo: i.typeEnv["error"].TypeInfo,
							Value:    Error{Message: err.Error()},
						},
					},
				}, nil
			}

			return TupleValue{
				Values: []Value{
					StringValue{V: string(data)},
					NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("Exists", &BuiltinFunc{
		Name:  "Exists",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			path, err := argString(node, args, 0, "fs.Exists")
			if err != nil {
				return NilValue{}, err
			}

			_, err = os.Stat(path)
			if err == nil {
				return TupleValue{Values: []Value{
					BoolValue{V: true},
					NilValue{},
				}}, nil
			}

			if os.IsNotExist(err) {
				return TupleValue{Values: []Value{
					BoolValue{V: false},
					NilValue{},
				}}, nil
			}

			return TupleValue{Values: []Value{
				NilValue{},
				InterfaceValue{
					TypeInfo: i.typeEnv["error"].TypeInfo,
					Value:    Error{Message: err.Error()},
				},
			}}, nil
		},
	}, false)

	env.Define("Append", &BuiltinFunc{
		Name:  "Append",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			path, err := argString(node, args, 0, "fs.Append")
			if err != nil {
				return NilValue{}, err
			}

			data, err := argString(node, args, 1, "fs.Append")
			if err != nil {
				return NilValue{}, err
			}

			file, err := os.OpenFile(
				path,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY,
				0644,
			)

			if err != nil {
				return InterfaceValue{
					TypeInfo: i.typeEnv["error"].TypeInfo,
					Value:    Error{Message: err.Error()},
				}, nil
			}
			defer file.Close()

			_, err = file.WriteString(data)
			if err != nil {
				return InterfaceValue{
					TypeInfo: i.typeEnv["error"].TypeInfo,
					Value:    Error{Message: err.Error()},
				}, nil
			}

			return NilValue{}, nil
		},
	}, false)

	env.Define("Delete", &BuiltinFunc{
		Name:  "Delete",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			path, err := argString(node, args, 0, "fs.Delete")
			if err != nil {
				return NilValue{}, err
			}

			err = os.Remove(path)
			if err != nil {
				return NilValue{}, NewRuntimeError(node, err.Error())
			}

			return NilValue{}, nil
		},
	}, false)

	env.Define("List", &BuiltinFunc{
		Name:  "List",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			path, err := argString(node, args, 0, "fs.List")
			if err != nil {
				return NilValue{}, err
			}

			entries, err := os.ReadDir(path)
			if err != nil {
				return TupleValue{
					Values: []Value{
						NilValue{},
						InterfaceValue{
							TypeInfo: i.typeEnv["error"].TypeInfo,
							Value:    Error{Message: err.Error()},
						},
					},
				}, nil
			}

			slice := make([]Value, 0, len(entries))
			for _, e := range entries {
				slice = append(slice, StringValue{V: e.Name()})
			}

			return TupleValue{
				Values: []Value{
					ArrayValue{
						Elements: slice,
						ElemType: i.typeEnv["string"].TypeInfo,
					},
					NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("Mkdir", &BuiltinFunc{
		Name:  "Mkdir",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			path, err := argString(node, args, 0, "fs.Mkdir")
			if err != nil {
				return NilValue{}, err
			}

			err = os.MkdirAll(path, 0755)
			if err != nil {
				return InterfaceValue{
					TypeInfo: i.typeEnv["error"].TypeInfo,
					Value:    Error{Message: err.Error()},
				}, nil
			}

			return NilValue{}, nil
		},
	}, false)

	env.Define("IsDir", &BuiltinFunc{
		Name:  "IsDir",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			path, err := argString(node, args, 0, "fs.IsDir")
			if err != nil {
				return NilValue{}, err
			}

			info, err := os.Stat(path)
			if err != nil {
				return TupleValue{
					Values: []Value{
						NilValue{},
						InterfaceValue{
							TypeInfo: i.typeEnv["error"].TypeInfo,
							Value:    Error{Message: err.Error()},
						},
					},
				}, nil
			}

			return TupleValue{
				Values: []Value{
					BoolValue{V: info.IsDir()},
					NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("Walk", &BuiltinFunc{
		Name:  "Walk",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			root, err := argString(node, args, 0, "fs.Walk")
			if err != nil {
				return NilValue{}, err
			}

			var files []Value

			err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				files = append(files, StringValue{V: path})
				return nil
			})

			if err != nil {
				return TupleValue{
					Values: []Value{
						NilValue{},
						InterfaceValue{
							TypeInfo: i.typeEnv["error"].TypeInfo,
							Value:    Error{Message: err.Error()},
						},
					},
				}, nil
			}

			return TupleValue{
				Values: []Value{
					ArrayValue{Elements: files, ElemType: i.typeEnv["string"].TypeInfo},
					NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("Cwd", &BuiltinFunc{
		Name:  "Cwd",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			wd, err := os.Getwd()
			if err != nil {
				return TupleValue{
					Values: []Value{
						NilValue{},
						InterfaceValue{
							TypeInfo: i.typeEnv["error"].TypeInfo,
							Value:    Error{Message: err.Error()},
						},
					},
				}, nil
			}

			return TupleValue{
				Values: []Value{
					StringValue{V: wd},
					NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("Size", &BuiltinFunc{
		Name:  "Size",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			path, err := argString(node, args, 0, "fs.Size")
			if err != nil {
				return NilValue{}, err
			}

			info, err := os.Stat(path)
			if err != nil {
				return TupleValue{
					Values: []Value{
						NilValue{},
						InterfaceValue{
							TypeInfo: i.typeEnv["error"].TypeInfo,
							Value:    Error{Message: err.Error()},
						},
					},
				}, nil
			}

			return TupleValue{
				Values: []Value{
					IntValue{V: int(info.Size())},
					NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("ModTime", &BuiltinFunc{
		Name:  "ModTime",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			path, err := argString(node, args, 0, "fs.ModTime")
			if err != nil {
				return NilValue{}, err
			}

			info, err := os.Stat(path)
			if err != nil {
				return TupleValue{
					Values: []Value{
						NilValue{},
						InterfaceValue{
							TypeInfo: i.typeEnv["error"].TypeInfo,
							Value:    Error{Message: err.Error()},
						},
					},
				}, nil
			}

			return TupleValue{
				Values: []Value{
					IntValue{V: int(info.ModTime().Unix())},
					NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("Rename", &BuiltinFunc{
		Name:  "Rename",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			old, err := argString(node, args, 0, "fs.Rename")
			if err != nil {
				return NilValue{}, err
			}

			new, err := argString(node, args, 1, "fs.Rename")
			if err != nil {
				return NilValue{}, err
			}

			err = os.Rename(old, new)
			if err != nil {
				return InterfaceValue{
					TypeInfo: i.typeEnv["error"].TypeInfo,
					Value:    Error{Message: err.Error()},
				}, nil
			}

			return NilValue{}, nil
		},
	}, false)

	env.Define("Copy", &BuiltinFunc{
		Name:  "Copy",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			src, err := argString(node, args, 0, "fs.Copy")
			if err != nil {
				return NilValue{}, err
			}

			dst, err := argString(node, args, 1, "fs.Copy")
			if err != nil {
				return NilValue{}, err
			}

			in, err := os.Open(src)
			if err != nil {
				return InterfaceValue{
					TypeInfo: i.typeEnv["error"].TypeInfo,
					Value:    Error{Message: err.Error()},
				}, nil
			}
			defer in.Close()

			out, err := os.Create(dst)
			if err != nil {
				return InterfaceValue{
					TypeInfo: i.typeEnv["error"].TypeInfo,
					Value:    Error{Message: err.Error()},
				}, nil
			}
			defer out.Close()

			_, err = io.Copy(out, in)
			if err != nil {
				return InterfaceValue{
					TypeInfo: i.typeEnv["error"].TypeInfo,
					Value:    Error{Message: err.Error()},
				}, nil
			}

			return NilValue{}, nil
		},
	}, false)

	module := ModuleValue{
		Name: "fs",
		Env:  env,
	}

	return module, nil
}

func LoadTimeModule(i *Interpreter) (ModuleValue, error) {
	env := NewEnvironment(i.Env)

	env.Define("Sleep", &BuiltinFunc{
		Name:  "Sleep",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			var seconds float64

			switch v := unwrapUntyped(unwrapNamed(args[0])).(type) {
			case IntValue:
				seconds = float64(v.V)
			case FloatValue:
				seconds = v.V
			default:
				return NilValue{}, NewRuntimeError(node, "time.Sleep: argument must be number")
			}

			time.Sleep(time.Duration(seconds * float64(time.Second)))
			return NilValue{}, nil
		},
	}, false)

	env.Define("Now", &BuiltinFunc{
		Name:  "Now",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			return IntValue{V: int(time.Now().Unix())}, nil
		},
	}, false)

	env.Define("Since", &BuiltinFunc{
		Name:  "Since",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			start, err := argInt(node, args, 0, "time.Since")
			if err != nil {
				return NilValue{}, err
			}

			now := time.Now().Unix()
			return IntValue{V: int(now - int64(start))}, nil
		},
	}, false)

	env.Define("Format", &BuiltinFunc{
		Name:  "Format",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			switch len(args) {
			case 1:
				ts, err := argInt(node, args, 0, "time.Format")
				if err != nil {
					return NilValue{}, err
				}

				t := time.Unix(int64(ts), 0)
				s := t.Format("2006-01-02 15:04:05")

				return StringValue{V: s}, nil
			case 2:
				ts, err := argInt(node, args, 0, "time.Format")
				if err != nil {
					return NilValue{}, err
				}

				layout, err := argString(node, args, 1, "time.Format")
				if err != nil {
					return NilValue{}, err
				}

				t := time.Unix(int64(ts), 0)
				s := t.Format(layout)

				return StringValue{V: s}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "time.Format: invalid amount of args, 0-2 args allowed")
			}
		},
	}, false)

	module := ModuleValue{
		Name: "time",
		Env:  env,
	}

	return module, nil
}

func LoadParseModule(i *Interpreter) (ModuleValue, error) {
	env := NewEnvironment(i.Env)
	typeEnv := make(map[string]TypeValue)

	env.Define("Int", &BuiltinFunc{
		Name:  "Int",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := unwrapUntyped(args[0])

			ti := unwrapAlias(i.typeInfoFromValue(v))

			switch ti.Kind {
			case TypeInt:
				return TupleValue{
					Values: []Value{
						v,
						NilValue{},
					},
				}, nil
			case TypeFloat:
				return TupleValue{
					Values: []Value{
						IntValue{V: int(v.(FloatValue).V)},
						NilValue{},
					},
				}, nil
			case TypeString:
				n, err := strconv.Atoi(v.(StringValue).V)
				if err != nil {
					return TupleValue{
						Values: []Value{
							IntValue{V: n},
							Error{Message: fmt.Sprintf("parse.Int: %s", err.Error())},
						},
					}, nil
				}

				return TupleValue{
					Values: []Value{
						IntValue{V: n},
						NilValue{},
					},
				}, nil
			case TypeBool:
				if v.(BoolValue).V {
					return TupleValue{
						Values: []Value{
							IntValue{V: 1},
							NilValue{},
						},
					}, nil
				}

				return TupleValue{
					Values: []Value{
						IntValue{V: 0},
						NilValue{},
					},
				}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "parse.Int: cannot convert to int")
			}
		},
	}, false)

	env.Define("Float", &BuiltinFunc{
		Name:  "Float",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := unwrapUntyped(args[0])

			ti := unwrapAlias(i.typeInfoFromValue(v))

			switch ti.Kind {
			case TypeFloat:
				return TupleValue{
					Values: []Value{
						v,
						NilValue{},
					},
				}, nil
			case TypeInt:
				return TupleValue{
					Values: []Value{
						FloatValue{V: float64(v.(IntValue).V)},
						NilValue{},
					},
				}, nil
			case TypeString:
				n, err := strconv.ParseFloat(v.(StringValue).V, 64)
				if err != nil {
					return TupleValue{
						Values: []Value{
							NilValue{},
							Error{Message: fmt.Sprintf("parse.Float: %s", err.Error())},
						},
					}, nil
				}

				return TupleValue{
					Values: []Value{
						FloatValue{V: n},
						NilValue{},
					},
				}, nil
			case TypeBool:
				if v.(BoolValue).V {
					return TupleValue{
						Values: []Value{
							FloatValue{V: 1},
							NilValue{},
						},
					}, nil
				}

				return TupleValue{
					Values: []Value{
						FloatValue{V: 0},
						NilValue{},
					},
				}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "parse.Float: cannot convert to float")
			}
		},
	}, false)

	env.Define("String", &BuiltinFunc{
		Name:  "String",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := unwrapUntyped(args[0])

			ti := unwrapAlias(i.typeInfoFromValue(v))

			switch ti.Kind {
			case TypeString:
				return v, nil
			case TypeInt:
				s := strconv.Itoa(v.(IntValue).V)

				return StringValue{V: s}, nil
			case TypeFloat:
				s := strconv.FormatFloat(v.(FloatValue).V, 'g', -1, 64)
				return StringValue{V: s}, nil
			case TypeBool:
				if v.(BoolValue).V {
					return StringValue{V: "yes"}, nil
				}

				return StringValue{V: "no"}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "parse.String: cannot convert to string")
			}
		},
	}, false)

	env.Define("Bool", &BuiltinFunc{
		Name:  "Bool",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := unwrapUntyped(args[0])

			ti := unwrapAlias(i.typeInfoFromValue(v))

			switch ti.Kind {
			case TypeBool:
				return TupleValue{
					Values: []Value{
						v,
						NilValue{},
					},
				}, nil
			case TypeInt:
				if v.(IntValue).V != 0 {
					return TupleValue{
						Values: []Value{
							BoolValue{V: true},
							NilValue{},
						},
					}, nil
				}

				return TupleValue{
					Values: []Value{
						BoolValue{V: false},
						NilValue{},
					},
				}, nil
			case TypeFloat:
				if v.(FloatValue).V != 0 {
					return TupleValue{
						Values: []Value{
							BoolValue{V: true},
							NilValue{},
						},
					}, nil
				}

				return TupleValue{
					Values: []Value{
						BoolValue{V: false},
						NilValue{},
					},
				}, nil
			case TypeString:
				s := strings.ToLower(v.(StringValue).V)

				if s == "yes" {
					return TupleValue{
						Values: []Value{
							BoolValue{V: true},
							NilValue{},
						},
					}, nil
				}
				if s == "no" {
					return TupleValue{
						Values: []Value{
							BoolValue{V: false},
							NilValue{},
						},
					}, nil
				}
				if s == "true" {
					return TupleValue{
						Values: []Value{
							NilValue{},
							Error{Message: "parse.Bool: invalid boolean string"},
						},
					}, nil
				}
				if s == "false" {
					return TupleValue{
						Values: []Value{
							NilValue{},
							Error{Message: "parse.Bool: invalid boolean string"},
						},
					}, nil
				}

				b, err := strconv.ParseBool(s)
				if err != nil {
					return TupleValue{
						Values: []Value{
							NilValue{},
							Error{Message: "parse.Bool: invalid boolean string"},
						},
					}, nil
				}

				return TupleValue{
					Values: []Value{
						BoolValue{V: b},
						NilValue{},
					},
				}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "parse.Bool: cannot convert to bool")
			}
		},
	}, false)

	mod := ModuleValue{
		Name:    "parse",
		Env:     env,
		typeEnv: typeEnv,
	}

	return mod, nil
}

func LoadGFXModule(i *Interpreter) (ModuleValue, error) {
	env := NewEnvironment(i.Env)
	typeEnv := make(map[string]TypeValue)

	min := 0.0
	max := 255.0

	uint8Type := &TypeInfo{
		Name:         "int[0..255]",
		Kind:         TypeInt,
		Min:          &min,
		Max:          &max,
		IsComparable: true,
	}

	typeEnv["Color"] = TypeValue{
		TypeInfo: &TypeInfo{
			Name: "Color",
			Kind: TypeStruct,
			Fields: map[string]*TypeInfo{
				"R": uint8Type,
				"G": uint8Type,
				"B": uint8Type,
				"A": uint8Type,
			},
		},
	}

	typeEnv["Vector2"] = TypeValue{
		TypeInfo: &TypeInfo{
			Name: "Vector2",
			Kind: TypeStruct,
			Fields: map[string]*TypeInfo{
				"X": i.typeEnv["float"].TypeInfo,
				"Y": i.typeEnv["float"].TypeInfo,
			},
		},
	}

	env.Define("SetWindowFlags", &BuiltinFunc{
		Name:  "SetWindowFlags",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			flag, err := argInt(node, args, 0, "gfx.SetWindowFlags")
			if err != nil {
				return NilValue{}, err
			}

			rl.SetConfigFlags(uint32(flag))
			return NilValue{}, nil
		},
	}, false)

	env.Define("InitWindow", &BuiltinFunc{
		Name:  "Init",
		Arity: 3,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			w, err := argInt(node, args, 0, "gfx.InitWindow")
			if err != nil {
				return NilValue{}, err
			}

			h, err := argInt(node, args, 1, "gfx.InitWindow")
			if err != nil {
				return NilValue{}, err
			}

			title, err := argString(node, args, 2, "gfx.InitWindow")
			if err != nil {
				return NilValue{}, err
			}

			rl.InitWindow(int32(w), int32(h), title)
			rl.SetTargetFPS(60)

			return NilValue{}, nil
		},
	}, false)

	env.Define("CloseWindow", &BuiltinFunc{
		Name:  "CloseWindow",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			rl.CloseWindow()
			return NilValue{}, nil
		},
	}, false)

	env.Define("SetTargetFPS", &BuiltinFunc{
		Name:  "SetTargetFPS",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			fps, err := argInt(node, args, 0, "gfx.SetTargetFPS")
			if err != nil {
				return NilValue{}, err
			}

			rl.SetTargetFPS(int32(fps))
			return NilValue{}, nil
		},
	}, false)

	env.Define("ShouldClose", &BuiltinFunc{
		Name:  "ShouldClose",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			return BoolValue{V: rl.WindowShouldClose()}, nil
		},
	}, false)

	env.Define("Clear", &BuiltinFunc{
		Name:  "Clear",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {

			switch len(args) {
			case 0:
				rl.BeginDrawing()
				rl.ClearBackground(rl.Black)

				return NilValue{}, nil
			case 1:
				col, err := argColor(node, typeEnv, args, 0, "gfx.Clear")
				if err != nil {
					return NilValue{}, err
				}

				rl.BeginDrawing()
				rl.ClearBackground(col)
				return NilValue{}, nil
			}

			return NilValue{}, NewRuntimeError(node, "gfx.Clear: invalid amount of arguments, expected between 0-1 arguments")
		},
	}, false)

	env.Define("Present", &BuiltinFunc{
		Name:  "Present",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			rl.EndDrawing()
			return NilValue{}, nil
		},
	}, false)

	env.Define("SetWindowTitle", &BuiltinFunc{
		Name:  "SetWindowTitle",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			title, err := argString(node, args, 0, "gfx.SetWindowTitle")
			if err != nil {
				return NilValue{}, err
			}

			rl.SetWindowTitle(title)
			return NilValue{}, nil
		},
	}, false)

	env.Define("SetWindowPos", &BuiltinFunc{
		Name:  "SetWindowPos",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			x, err := argInt(node, args, 0, "gfx.SetWindowPos")
			if err != nil {
				return NilValue{}, err
			}

			y, err := argInt(node, args, 1, "gfx.SetWindowPos")
			if err != nil {
				return NilValue{}, err
			}

			rl.SetWindowPosition(x, y)
			return NilValue{}, nil
		},
	}, false)

	env.Define("GetWindowPos", &BuiltinFunc{
		Name:  "GetWindowPos",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			return &StructValue{
				TypeName: typeEnv["Vector2"].TypeInfo,
				Fields: map[string]Value{
					"X": IntValue{int(rl.GetWindowPosition().X)},
					"Y": IntValue{int(rl.GetWindowPosition().Y)},
				},
			}, nil
		},
	}, false)

	env.Define("SetWindowSize", &BuiltinFunc{
		Name:  "SetWindowSize",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			w, err := argInt(node, args, 0, "gfx.SetWindowSize")
			if err != nil {
				return NilValue{}, err
			}

			h, err := argInt(node, args, 1, "gfx.SetWindowSize")
			if err != nil {
				return NilValue{}, err
			}

			rl.SetWindowSize(w, h)
			return NilValue{}, err
		},
	}, false)

	env.Define("SetWindowMinSize", &BuiltinFunc{
		Name:  "SetWindowMinSize",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			w, err := argInt(node, args, 0, "gfx.SetWindowMinSize")
			if err != nil {
				return NilValue{}, err
			}

			h, err := argInt(node, args, 1, "gfx.SetWindowMinSize")
			if err != nil {
				return NilValue{}, err
			}

			rl.SetWindowMinSize(w, h)
			return NilValue{}, nil
		},
	}, false)

	env.Define("SetWindowMaxSize", &BuiltinFunc{
		Name:  "SetWindowMaxSize",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			w, err := argInt(node, args, 0, "gfx.SetWindowMaxSize")
			if err != nil {
				return NilValue{}, err
			}

			h, err := argInt(node, args, 1, "gfx.SetWindowMaxSize")
			if err != nil {
				return NilValue{}, err
			}

			rl.SetWindowMaxSize(w, h)
			return NilValue{}, nil
		},
	}, false)

	env.Define("GetWindowSize", &BuiltinFunc{
		Name:  "GetWindowSize",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			return &StructValue{
				TypeName: typeEnv["Vector2"].TypeInfo,
				Fields: map[string]Value{
					"X": IntValue{V: rl.GetScreenWidth()},
					"Y": IntValue{V: rl.GetScreenHeight()},
				},
			}, nil
		},
	}, false)

	env.Define("SetFullscreen", &BuiltinFunc{
		Name:  "SetFullscreen",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			enabled, err := argBool(node, args, 0, "gfx.SetFullscreen")
			if err != nil {
				return NilValue{}, err
			}

			if rl.IsWindowFullscreen() != enabled {
				rl.ToggleFullscreen()
			}

			return NilValue{}, nil
		},
	}, false)

	env.Define("IsWindowFocused", &BuiltinFunc{
		Name:  "IsWindowFocused",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			return BoolValue{V: rl.IsWindowFocused()}, nil
		},
	}, false)

	env.Define("IsWindowMinimized", &BuiltinFunc{
		Name:  "IsWindowMinimized",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			return BoolValue{V: rl.IsWindowMinimized()}, nil
		},
	}, false)

	env.Define("IsWindowMaximized", &BuiltinFunc{
		Name:  "IsWindowMaximized",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			return BoolValue{V: rl.IsWindowMaximized()}, nil
		},
	}, false)

	env.Define("MinimizeWindow", &BuiltinFunc{
		Name:  "MinimizeWindow",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			rl.MinimizeWindow()
			return NilValue{}, nil
		},
	}, false)

	env.Define("MaximizeWindow", &BuiltinFunc{
		Name:  "MaximizeWindow",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			rl.MaximizeWindow()
			return NilValue{}, nil
		},
	}, false)

	env.Define("RestoreWindow", &BuiltinFunc{
		Name:  "RestoreWindow",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			rl.RestoreWindow()
			return NilValue{}, nil
		},
	}, false)

	env.Define("GetFPS", &BuiltinFunc{
		Name:  "GetFPS",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			return IntValue{V: int(rl.GetFPS())}, nil
		},
	}, false)

	env.Define("GetFrameTime", &BuiltinFunc{
		Name:  "GetFrameTime",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			return FloatValue{V: float64(rl.GetFrameTime())}, nil
		},
	}, false)

	env.Define("ShowCursor", &BuiltinFunc{
		Name:  "ShowCursor",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			rl.ShowCursor()
			return NilValue{}, nil
		},
	}, false)

	env.Define("HideCursor", &BuiltinFunc{
		Name:  "HideCursor",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			rl.HideCursor()
			return NilValue{}, nil
		},
	}, false)

	env.Define("LockCursor", &BuiltinFunc{
		Name:  "LockCursor",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			rl.DisableCursor()
			return NilValue{}, nil
		},
	}, false)

	env.Define("UnlockCursor", &BuiltinFunc{
		Name:  "UnlockCursor",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			rl.EnableCursor()
			return NilValue{}, nil
		},
	}, false)

	env.Define("NewColor", &BuiltinFunc{
		Name:  "NewColor",
		Arity: 4,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			r, err := argInt(node, args, 0, "gfx.NewColor")
			if err != nil {
				return NilValue{}, err
			}

			g, err := argInt(node, args, 1, "gfx.NewColor")
			if err != nil {
				return NilValue{}, err
			}

			b, err := argInt(node, args, 2, "gfx.NewColor")
			if err != nil {
				return NilValue{}, err
			}

			a, err := argInt(node, args, 3, "gfx.NewColor")
			if err != nil {
				return NilValue{}, err
			}

			return &StructValue{
				TypeName: typeEnv["Color"].TypeInfo,
				Fields: map[string]Value{
					"R": IntValue{V: r},
					"G": IntValue{V: g},
					"B": IntValue{V: b},
					"A": IntValue{V: a},
				},
			}, nil
		},
	}, false)

	env.Define("NewVector2", &BuiltinFunc{
		Name:  "NewVector2",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			x, err := argFloat(node, args, 0, "gfx.NewVector2")
			if err != nil {
				return NilValue{}, err
			}

			y, err := argFloat(node, args, 1, "gfx.NewVector2")
			if err != nil {
				return NilValue{}, err
			}

			vec := &StructValue{
				TypeName: typeEnv["Vector2"].TypeInfo,
				Fields: map[string]Value{
					"X": FloatValue{V: x},
					"Y": FloatValue{V: y},
				},
			}

			return vec, nil
		},
	}, false)

	env.Define("DrawRect", &BuiltinFunc{
		Name:  "DrawRect",
		Arity: 3,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			pv, err := argVector2(node, i, typeEnv, args, 0, "gfx.DrawRect")
			if err != nil {
				return NilValue{}, err
			}

			sv, err := argVector2(node, i, typeEnv, args, 1, "gfx.DrawRect")
			if err != nil {
				return NilValue{}, err
			}

			col, err := argColor(node, typeEnv, args, 2, "gfx.DrawRect")
			if err != nil {
				return NilValue{}, err
			}

			rl.DrawRectangleV(pv, sv, col)
			return NilValue{}, nil
		},
	}, false)

	env.Define("DrawRectLines", &BuiltinFunc{
		Name:  "DrawRectLines",
		Arity: 3,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			pv, err := argVector2(node, i, typeEnv, args, 0, "gfx.DrawRectLines")
			if err != nil {
				return NilValue{}, err
			}

			sv, err := argVector2(node, i, typeEnv, args, 1, "gfx.DrawRectLines")
			if err != nil {
				return NilValue{}, err
			}

			col, err := argColor(node, typeEnv, args, 2, "gfx.DrawRectLines")
			if err != nil {
				return NilValue{}, err
			}

			rl.DrawRectangleLines(int32(pv.X), int32(pv.Y), int32(sv.X), int32(sv.Y), col)
			return NilValue{}, nil
		},
	}, false)

	env.Define("DrawCircle", &BuiltinFunc{
		Name:  "DrawCircle",
		Arity: 3,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			pv, err := argVector2(node, i, typeEnv, args, 0, "gfx.DrawCircle")
			if err != nil {
				return NilValue{}, err
			}

			rVal, err := argFloat(node, args, 1, "gfx.DrawCircle")
			if err != nil {
				return NilValue{}, err
			}

			col, err := argColor(node, typeEnv, args, 2, "gfx.DrawCircle")
			if err != nil {
				return NilValue{}, err
			}

			r := float32(rVal)

			rl.DrawCircleV(pv, r, col)
			return NilValue{}, nil
		},
	}, false)

	env.Define("DrawCircleLines", &BuiltinFunc{
		Name:  "DrawCircleLines",
		Arity: 3,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			pv, err := argVector2(node, i, typeEnv, args, 0, "gfx.DrawCircleLines")
			if err != nil {
				return NilValue{}, err
			}

			rVal, err := argFloat(node, args, 1, "gfx.DrawCircleLines")
			if err != nil {
				return NilValue{}, err
			}

			col, err := argColor(node, typeEnv, args, 2, "gfx.DrawCircleLines")
			if err != nil {
				return NilValue{}, err
			}

			r := float32(rVal)

			rl.DrawCircleLinesV(pv, r, col)
			return NilValue{}, nil
		},
	}, false)

	env.Define("DrawText", &BuiltinFunc{
		Name:  "DrawText",
		Arity: 4,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			txtVal, err := argString(node, args, 0, "gfx.DrawText")
			if err != nil {
				return NilValue{}, err
			}

			pv, err := argVector2(node, i, typeEnv, args, 1, "gfx.DrawRect")
			if err != nil {
				return NilValue{}, err
			}

			fontVal, err := argInt(node, args, 2, "gfx.DrawText")
			if err != nil {
				return NilValue{}, err
			}

			col, err := argColor(node, typeEnv, args, 3, "gfx.DrawText")
			if err != nil {
				return NilValue{}, err
			}

			txt := txtVal
			x := int32(pv.X)
			y := int32(pv.Y)
			font := int32(fontVal)

			rl.DrawText(txt, x, y, font, col)
			return NilValue{}, nil
		},
	}, false)

	env.Define("DrawLine", &BuiltinFunc{
		Name:  "DrawLine",
		Arity: 3,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			pv, err := argVector2(node, i, typeEnv, args, 0, "gfx.DrawLine")
			if err != nil {
				return NilValue{}, err
			}

			sv, err := argVector2(node, i, typeEnv, args, 1, "gfx.DrawLine")
			if err != nil {
				return NilValue{}, err
			}

			col, err := argColor(node, typeEnv, args, 2, "gfx.DrawLine")
			if err != nil {
				return NilValue{}, err
			}

			rl.DrawLineV(pv, sv, col)
			return NilValue{}, nil
		},
	}, false)

	env.Define("DrawTriangle", &BuiltinFunc{
		Name:  "DrawTriangle",
		Arity: 4,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			pv1, err := argVector2(node, i, typeEnv, args, 0, "DrawTriangle")
			if err != nil {
				return NilValue{}, err
			}

			pv2, err := argVector2(node, i, typeEnv, args, 1, "DrawTriangle")
			if err != nil {
				return NilValue{}, err
			}

			pv3, err := argVector2(node, i, typeEnv, args, 2, "DrawTriangle")
			if err != nil {
				return NilValue{}, err
			}

			col, err := argColor(node, typeEnv, args, 3, "DrawTriangle")
			if err != nil {
				return NilValue{}, err
			}

			rl.DrawTriangle(pv1, pv3, pv2, col)
			return NilValue{}, nil
		},
	}, false)

	env.Define("DrawTriangleLines", &BuiltinFunc{
		Name:  "DrawTriangleLines",
		Arity: 4,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			pv1, err := argVector2(node, i, typeEnv, args, 0, "DrawTriangle")
			if err != nil {
				return NilValue{}, err
			}

			pv2, err := argVector2(node, i, typeEnv, args, 1, "DrawTriangle")
			if err != nil {
				return NilValue{}, err
			}

			pv3, err := argVector2(node, i, typeEnv, args, 2, "DrawTriangle")
			if err != nil {
				return NilValue{}, err
			}

			col, err := argColor(node, typeEnv, args, 3, "DrawTriangle")
			if err != nil {
				return NilValue{}, err
			}

			rl.DrawTriangleLines(pv1, pv3, pv2, col)
			return NilValue{}, nil
		},
	}, false)

	env.Define("IsKeyDown", &BuiltinFunc{
		Name:  "IsKeyDown",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := argInt(node, args, 0, "gfx.KeyDown")
			if err != nil {
				return NilValue{}, err
			}

			return BoolValue{V: rl.IsKeyDown(int32(v))}, nil
		},
	}, false)

	env.Define("GetKeyPressed", &BuiltinFunc{
		Name:  "GetKeyPressed",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			return IntValue{V: int(rl.GetKeyPressed())}, nil
		},
	}, false)

	env.Define("IsKeyPressed", &BuiltinFunc{
		Name:  "IsKeyPressed",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := argInt(node, args, 0, "gfx.KeyPressed")
			if err != nil {
				return NilValue{}, err
			}

			return BoolValue{V: rl.IsKeyPressed(int32(v))}, nil
		},
	}, false)

	env.Define("IsKeyReleased", &BuiltinFunc{
		Name:  "IsKeyReleased",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := argInt(node, args, 0, "gfx.KeyReleased")
			if err != nil {
				return NilValue{}, err
			}

			return BoolValue{V: rl.IsKeyReleased(int32(v))}, nil
		},
	}, false)

	env.Define("IsKeyUp", &BuiltinFunc{
		Name:  "IsKeyUp",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := argInt(node, args, 0, "gfx.KeyUp")
			if err != nil {
				return NilValue{}, err
			}

			return BoolValue{V: rl.IsKeyUp(int32(v))}, nil
		},
	}, false)

	env.Define("IsMouseDown", &BuiltinFunc{
		Name:  "IsMouseDown",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := argInt(node, args, 0, "gfx.MouseDown")
			if err != nil {
				return NilValue{}, err
			}

			return BoolValue{V: rl.IsMouseButtonDown(rl.MouseButton((v)))}, nil
		},
	}, false)

	env.Define("IsMousePressed", &BuiltinFunc{
		Name:  "IsMousePressed",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := argInt(node, args, 0, "gfx.MousePressed")
			if err != nil {
				return NilValue{}, err
			}

			return BoolValue{V: rl.IsMouseButtonPressed(rl.MouseButton(int32(v)))}, nil
		},
	}, false)

	env.Define("IsMouseReleased", &BuiltinFunc{
		Name:  "IsMouseReleased",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := argInt(node, args, 0, "gfx.MouseReleased")
			if err != nil {
				return NilValue{}, err
			}

			return BoolValue{V: rl.IsMouseButtonReleased(rl.MouseButton(int32(v)))}, nil
		},
	}, false)

	env.Define("IsMouseUp", &BuiltinFunc{
		Name:  "IsMouseUp",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := argInt(node, args, 0, "gfx.MouseUp")
			if err != nil {
				return NilValue{}, err
			}

			return BoolValue{V: rl.IsMouseButtonUp(rl.MouseButton(int32(v)))}, nil
		},
	}, false)

	env.Define("SetMousePos", &BuiltinFunc{
		Name:  "SetMousePos",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			x, err := argInt(node, args, 0, "gfx.SetMousePos")
			if err != nil {
				return NilValue{}, err
			}

			y, err := argInt(node, args, 1, "gfx.SetMousePos")
			if err != nil {
				return NilValue{}, err
			}

			rl.SetMousePosition(x, y)
			return NilValue{}, nil
		},
	}, false)

	env.Define("GetMousePos", &BuiltinFunc{
		Name:  "GetMousePos",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {

			return &StructValue{
				TypeName: typeEnv["Vector2"].TypeInfo,
				Fields: map[string]Value{
					"X": FloatValue{V: float64(rl.GetMousePosition().X)},
					"Y": FloatValue{V: float64(rl.GetMousePosition().Y)},
				},
			}, nil
		},
	}, false)

	env.Define("GetMouseDelta", &BuiltinFunc{
		Name:  "GetMouseDelta",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			return &StructValue{
				TypeName: typeEnv["Vector2"].TypeInfo,
				Fields: map[string]Value{
					"X": FloatValue{V: float64(rl.GetMouseDelta().X)},
					"Y": FloatValue{V: float64(rl.GetMouseDelta().Y)},
				},
			}, nil
		},
	}, false)

	// consts
	env.Define("MouseLeft", IntValue{V: int(rl.MouseButtonLeft)}, true)
	env.Define("MouseRight", IntValue{V: int(rl.MouseButtonRight)}, true)
	env.Define("MouseMiddle", IntValue{V: int(rl.MouseMiddleButton)}, true)

	env.Define("KeyA", IntValue{V: rl.KeyA}, true)
	env.Define("KeyB", IntValue{V: rl.KeyB}, true)
	env.Define("KeyC", IntValue{V: rl.KeyC}, true)
	env.Define("KeyD", IntValue{V: rl.KeyD}, true)
	env.Define("KeyE", IntValue{V: rl.KeyE}, true)
	env.Define("KeyF", IntValue{V: rl.KeyF}, true)
	env.Define("KeyG", IntValue{V: rl.KeyG}, true)
	env.Define("KeyH", IntValue{V: rl.KeyH}, true)
	env.Define("KeyI", IntValue{V: rl.KeyI}, true)
	env.Define("KeyJ", IntValue{V: rl.KeyJ}, true)
	env.Define("KeyK", IntValue{V: rl.KeyK}, true)
	env.Define("KeyL", IntValue{V: rl.KeyL}, true)
	env.Define("KeyM", IntValue{V: rl.KeyM}, true)
	env.Define("KeyN", IntValue{V: rl.KeyN}, true)
	env.Define("KeyO", IntValue{V: rl.KeyO}, true)
	env.Define("KeyP", IntValue{V: rl.KeyP}, true)
	env.Define("KeyQ", IntValue{V: rl.KeyQ}, true)
	env.Define("KeyR", IntValue{V: rl.KeyR}, true)
	env.Define("KeyS", IntValue{V: rl.KeyS}, true)
	env.Define("KeyT", IntValue{V: rl.KeyT}, true)
	env.Define("KeyU", IntValue{V: rl.KeyU}, true)
	env.Define("KeyV", IntValue{V: rl.KeyV}, true)
	env.Define("KeyW", IntValue{V: rl.KeyW}, true)
	env.Define("KeyX", IntValue{V: rl.KeyX}, true)
	env.Define("KeyY", IntValue{V: rl.KeyY}, true)
	env.Define("KeyZ", IntValue{V: rl.KeyZ}, true)
	env.Define("KeyUp", IntValue{V: rl.KeyUp}, true)
	env.Define("KeyDown", IntValue{V: rl.KeyDown}, true)
	env.Define("KeyLeft", IntValue{V: rl.KeyLeft}, true)
	env.Define("KeyRight", IntValue{V: rl.KeyRight}, true)
	env.Define("KeyEsc", IntValue{V: rl.KeyEscape}, true)
	env.Define("KeyF1", IntValue{V: rl.KeyF1}, true)
	env.Define("KeyF2", IntValue{V: rl.KeyF2}, true)
	env.Define("KeyF3", IntValue{V: rl.KeyF3}, true)
	env.Define("KeyF4", IntValue{V: rl.KeyF4}, true)
	env.Define("KeyF5", IntValue{V: rl.KeyF5}, true)
	env.Define("KeyF6", IntValue{V: rl.KeyF6}, true)
	env.Define("KeyF7", IntValue{V: rl.KeyF7}, true)
	env.Define("KeyF8", IntValue{V: rl.KeyF8}, true)
	env.Define("KeyF9", IntValue{V: rl.KeyF9}, true)
	env.Define("KeyF10", IntValue{V: rl.KeyF10}, true)
	env.Define("KeyF11", IntValue{V: rl.KeyF11}, true)
	env.Define("KeyF12", IntValue{V: rl.KeyF12}, true)
	env.Define("KeyPeriod", IntValue{V: rl.KeyPeriod}, true)
	env.Define("KeyComma", IntValue{V: rl.KeyComma}, true)
	env.Define("KeySpace", IntValue{V: rl.KeySpace}, true)
	env.Define("KeyEnter", IntValue{V: rl.KeyEnter}, true)
	env.Define("KeyCapsLock", IntValue{V: rl.KeyCapsLock}, true)
	env.Define("KeyLeftShift", IntValue{V: rl.KeyLeftShift}, true)
	env.Define("KeyLShift", IntValue{V: rl.KeyLeftShift}, true)
	env.Define("KeyRightShift", IntValue{V: rl.KeyRightShift}, true)
	env.Define("KeyRShift", IntValue{V: rl.KeyRightShift}, true)
	env.Define("KeyNumLock", IntValue{V: rl.KeyNumLock}, true)
	env.Define("KeyOne", IntValue{V: rl.KeyOne}, true)
	env.Define("KeyTwo", IntValue{V: rl.KeyTwo}, true)
	env.Define("KeyThree", IntValue{V: rl.KeyThree}, true)
	env.Define("KeyFour", IntValue{V: rl.KeyFour}, true)
	env.Define("KeyFive", IntValue{V: rl.KeyFive}, true)
	env.Define("KeySix", IntValue{V: rl.KeySix}, true)
	env.Define("KeySeven", IntValue{V: rl.KeySeven}, true)
	env.Define("KeyEight", IntValue{V: rl.KeyEight}, true)
	env.Define("KeyNine", IntValue{V: rl.KeyNine}, true)
	env.Define("KeyZero", IntValue{V: rl.KeyZero}, true)
	env.Define("KeySlash", IntValue{V: rl.KeySlash}, true)
	env.Define("KeyBackspace", IntValue{V: rl.KeyBackspace}, true)
	env.Define("KeyBackSlash", IntValue{V: rl.KeyBackSlash}, true)
	env.Define("KeyTab", IntValue{V: rl.KeyTab}, true)
	env.Define("KeyLeftBracket", IntValue{V: rl.KeyLeftBracket}, true)
	env.Define("KeyRightBracket", IntValue{V: rl.KeyRightBracket}, true)
	env.Define("KeyDelete", IntValue{V: rl.KeyDelete}, true)
	env.Define("KeyPrintScreen", IntValue{V: rl.KeyPrintScreen}, true)
	env.Define("KeyPause", IntValue{V: rl.KeyPause}, true)
	env.Define("KeyEnd", IntValue{V: rl.KeyEnd}, true)
	env.Define("KeyHome", IntValue{V: rl.KeyHome}, true)
	env.Define("KeyLeftLAlt", IntValue{V: rl.KeyLeftAlt}, true)
	env.Define("KeyRightAlt", IntValue{V: rl.KeyRightAlt}, true)
	env.Define("KeyRightControl", IntValue{V: rl.KeyRightControl}, true)
	env.Define("KeyLeftControl", IntValue{V: rl.KeyLeftControl}, true)

	env.Define("FlagWindowResizable", IntValue{V: rl.FlagWindowResizable}, true)
	env.Define("FlagWindowFullscreen", IntValue{V: rl.FlagFullscreenMode}, true)
	env.Define("FlagWindowMaximized", IntValue{V: rl.FlagWindowMaximized}, true)
	env.Define("FlagWindowMinimized", IntValue{V: rl.FlagWindowMinimized}, true)
	env.Define("FlagWindowAlwaysRun", IntValue{V: rl.FlagWindowAlwaysRun}, true)
	env.Define("FlagWindowVsync", IntValue{V: rl.FlagVsyncHint}, true)
	env.Define("FlagWindowHighDPI", IntValue{V: rl.FlagWindowHighdpi}, true)
	env.Define("FlagWindowBorderless", IntValue{V: rl.FlagBorderlessWindowedMode}, true)
	env.Define("FlagWindowMsaa4x", IntValue{V: rl.FlagMsaa4xHint}, true)

	module := ModuleValue{
		Name:    "gfx",
		Env:     env,
		typeEnv: typeEnv,
	}

	return module, nil
}
