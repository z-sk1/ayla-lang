package interpreter

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
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
	iv, ok := v.(IntValue)
	if !ok {
		return 0, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be an int", name, i+1))
	}
	return iv.V, nil
}

func argFloat(node parser.Node, args []Value, i int, name string) (float64, error) {
	v, ok := toFloat(unwrapNamed(args[i]))
	if !ok {
		return 0, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be an float", name, i+1))
	}
	return v, nil
}

func argString(node parser.Node, args []Value, i int, name string) (string, error) {
	v := unwrapNamed(args[i])
	iv, ok := v.(StringValue)
	if !ok {
		return "", NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a string", name, i+1))
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

func argVector2(node parser.Node, i *Interpreter, typeEnv map[string]TypeValue, args []Value, idx int, name string) (float32, float32, error) {
	vecTI := typeEnv["Vector2"].TypeInfo

	vecVal, ok := unwrapNamed(args[idx]).(*StructValue)
	if !ok {
		return 0, 0, NewRuntimeError(node, "gfx.DrawLine: first argument must be a gfx.Vector2")
	}

	if !typesAssignable(i.typeInfoFromValue(vecVal), vecTI) {
		return 0, 0, NewRuntimeError(node, "gfx.DrawLine: first argument must be a gfx.Vector2")
	}

	x, y := unwrapVector2(vecVal)

	return x, y, nil
}

func colorFromValue(v Value) (rl.Color, error) {
	colVal := v.(*StructValue)

	r := colVal.Fields["R"].(IntValue).V
	g := colVal.Fields["G"].(IntValue).V
	b := colVal.Fields["B"].(IntValue).V
	a := colVal.Fields["A"].(IntValue).V

	if r < 0 || r > 255 ||
		g < 0 || g > 255 ||
		b < 0 || b > 255 ||
		a < 0 || a > 255 {
		return rl.Color{}, fmt.Errorf("gfx.Color fields must be between 0 and 255")
	}

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
	env.Define("Abs", wrapFloat1("Abs", math.Abs))
	env.Define("Sin", wrapFloat1("Sin", math.Sin))
	env.Define("Asin", wrapFloat1("Asin", math.Asin))
	env.Define("Cos", wrapFloat1("Cos", math.Cos))
	env.Define("Acos", wrapFloat1("Acos", math.Acos))
	env.Define("Tan", wrapFloat1("Tan", math.Tan))
	env.Define("Atan", wrapFloat1("Atan", math.Atan))
	env.Define("Sqrt", wrapFloat1("Sqrt", math.Sqrt))
	env.Define("Log", wrapFloat1("Log", math.Log))
	env.Define("Exp", wrapFloat1("Exp", math.Exp))
	env.Define("Floor", wrapFloat1("Floor", math.Floor))
	env.Define("Ceil", wrapFloat1("Ceil", math.Ceil))
	env.Define("Round", wrapFloat1("Round", math.Round))
	env.Define("RoundToEven", wrapFloat1("RoundToEven", math.RoundToEven))
	env.Define("Trunc", wrapFloat1("Trunc", math.Trunc))

	env.Define("Max", wrapFloat2("Max", math.Max))
	env.Define("Min", wrapFloat2("Min", math.Min))
	env.Define("Pow", wrapFloat2("Pow", math.Pow))
	env.Define("Remainder", wrapFloat2("Remainder", math.Remainder))

	// constants
	env.Define("Pi", ConstValue{FloatValue{V: math.Pi}})
	env.Define("Phi", ConstValue{Value: FloatValue{V: math.Phi}})
	env.Define("E", ConstValue{Value: FloatValue{V: math.E}})
	env.Define("Sqrt2", ConstValue{Value: FloatValue{V: math.Sqrt2}})
	env.Define("SqrtPi", ConstValue{FloatValue{V: math.SqrtPi}})
	env.Define("SqrtPhi", ConstValue{Value: FloatValue{V: math.SqrtPhi}})
	env.Define("SqrtE", ConstValue{Value: FloatValue{V: math.SqrtE}})

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
	})

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
	})

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
	})

	module := ModuleValue{
		Name: "rand",
		Env:  env,
	}

	return module, nil
}

func LoadFSModule(i *Interpreter) (ModuleValue, error) {
	env := NewEnvironment(i.Env)

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
				return ErrorValue{V: NewRuntimeError(node, err.Error())}, nil
			}

			return NilValue{}, nil
		},
	})

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
						ErrorValue{
							V: NewRuntimeError(node, err.Error()),
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
	})

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
				ErrorValue{
					V: NewRuntimeError(node, err.Error()),
				},
			}}, nil
		},
	})

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
				return ErrorValue{
					V: NewRuntimeError(node, err.Error()),
				}, nil
			}
			defer file.Close()

			_, err = file.WriteString(data)
			if err != nil {
				return ErrorValue{
					V: NewRuntimeError(node, err.Error()),
				}, nil
			}

			return NilValue{}, nil
		},
	})

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
	})

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
						ErrorValue{
							V: NewRuntimeError(node, err.Error()),
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
	})

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
				return ErrorValue{
					V: NewRuntimeError(node, err.Error()),
				}, nil
			}

			return NilValue{}, nil
		},
	})

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
						ErrorValue{
							V: NewRuntimeError(node, err.Error()),
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
	})

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
						ErrorValue{
							V: NewRuntimeError(node, err.Error()),
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
	})

	env.Define("Cwd", &BuiltinFunc{
		Name:  "Cwd",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			wd, err := os.Getwd()
			if err != nil {
				return TupleValue{
					Values: []Value{
						NilValue{},
						ErrorValue{
							V: NewRuntimeError(node, err.Error()),
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
	})

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
						ErrorValue{
							V: NewRuntimeError(node, err.Error()),
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
	})

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
						ErrorValue{
							V: NewRuntimeError(node, err.Error()),
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
	})

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
				return ErrorValue{
					V: NewRuntimeError(node, err.Error()),
				}, nil
			}

			return NilValue{}, nil
		},
	})

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
				return ErrorValue{
					V: NewRuntimeError(node, err.Error()),
				}, nil
			}
			defer in.Close()

			out, err := os.Create(dst)
			if err != nil {
				return ErrorValue{
					V: NewRuntimeError(node, err.Error()),
				}, nil
			}
			defer out.Close()

			_, err = io.Copy(out, in)
			if err != nil {
				return ErrorValue{
					V: NewRuntimeError(node, err.Error()),
				}, nil
			}

			return NilValue{}, nil
		},
	})

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
			v := unwrapNamed(args[0])

			switch v := v.(type) {
			case IntValue:
				time.Sleep(time.Duration(v.V) * time.Second)
			case FloatValue:
				time.Sleep(time.Duration(v.V) * time.Second)
			default:
				return NilValue{}, NewRuntimeError(node, "time.Sleep: argument must be int or float seconds")
			}

			return NilValue{}, nil
		},
	})

	env.Define("Now", &BuiltinFunc{
		Name:  "Now",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			return IntValue{V: int(time.Now().Unix())}, nil
		},
	})

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
	})

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
	})

	module := ModuleValue{
		Name: "time",
		Env:  env,
	}

	return module, nil
}

func LoadGFXModule(i *Interpreter) (ModuleValue, error) {
	env := NewEnvironment(i.Env)
	typeEnv := make(map[string]TypeValue)

	typeEnv["Color"] = TypeValue{
		TypeInfo: &TypeInfo{
			Name: "Color",
			Kind: TypeStruct,
			Fields: map[string]*TypeInfo{
				"R": i.typeEnv["int"].TypeInfo,
				"G": i.typeEnv["int"].TypeInfo,
				"B": i.typeEnv["int"].TypeInfo,
				"A": i.typeEnv["int"].TypeInfo,
			},
			Validator: func(fields map[string]Value) error {
				var r, g, b, a int

				if fields["R"] != nil {
					r = unwrapNamed(fields["R"]).(IntValue).V
				}

				if fields["G"] != nil {
					g = unwrapNamed(fields["G"]).(IntValue).V
				}

				if fields["B"] != nil {
					b = unwrapNamed(fields["B"]).(IntValue).V
				}

				if fields["A"] != nil {
					a = unwrapNamed(fields["A"]).(IntValue).V
				}

				if r < 0 || r > 255 {
					return fmt.Errorf("gfx.Color.R must be between 0-255, got %d", r)
				}

				if g < 0 || g > 255 {
					return fmt.Errorf("gfx.Color.G must be between 0-255, got %d", g)
				}

				if b < 0 || b > 255 {
					return fmt.Errorf("gfx.Color.B must be between 0-255, got %d", b)
				}

				if a < 0 || a > 255 {
					return fmt.Errorf("gfx.Color.A must be between 0-255, got %d", a)
				}

				return nil
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

	env.Define("Init", &BuiltinFunc{
		Name:  "Init",
		Arity: 3,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			w, err := argInt(node, args, 0, "gfx.Init")
			if err != nil {
				return NilValue{}, err
			}

			h, err := argInt(node, args, 1, "gfx.Init")
			if err != nil {
				return NilValue{}, err
			}

			title, err := argString(node, args, 2, "gfx.Init")
			if err != nil {
				return NilValue{}, err
			}

			rl.InitWindow(int32(w), int32(h), title)
			rl.SetTargetFPS(60)

			return NilValue{}, nil
		},
	})

	env.Define("ShouldClose", &BuiltinFunc{
		Name:  "ShouldClose",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			return BoolValue{V: rl.WindowShouldClose()}, nil
		},
	})

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
	})

	env.Define("Present", &BuiltinFunc{
		Name:  "Present",
		Arity: 0,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			rl.EndDrawing()
			return NilValue{}, nil
		},
	})

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

			if r > 255 || r < 0 {
				return NilValue{}, NewRuntimeError(node, "gfx.NewColor: first argument must be between 0-255")
			}

			if g > 255 || g < 0 {
				return NilValue{}, NewRuntimeError(node, "gfx.NewColor: second argument must be between 0-255")
			}

			if b > 255 || b < 0 {
				return NilValue{}, NewRuntimeError(node, "gfx.NewColor: third argument must be between 0-255")
			}

			if a > 255 || a < 0 {
				return NilValue{}, NewRuntimeError(node, "gfx.NewColor: fourth argument must be between 0-255")
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
	})

	env.Define("DrawRect", &BuiltinFunc{
		Name:  "DrawRect",
		Arity: 5,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			xVal, err := argInt(node, args, 0, "gfx.DrawRect")
			if err != nil {
				return NilValue{}, err
			}

			yVal, err := argInt(node, args, 1, "gfx.DrawRect")
			if err != nil {
				return NilValue{}, err
			}

			wVal, err := argInt(node, args, 2, "gfx.DrawRect")
			if err != nil {
				return NilValue{}, err
			}

			hVal, err := argInt(node, args, 3, "gfx.DrawRect")
			if err != nil {
				return NilValue{}, err
			}

			col, err := argColor(node, typeEnv, args, 4, "gfx.DrawRect")
			if err != nil {
				return NilValue{}, err
			}

			x := int32(xVal)
			y := int32(yVal)
			w := int32(wVal)
			h := int32(hVal)

			rl.DrawRectangle(x, y, w, h, col)
			return NilValue{}, nil
		},
	})

	env.Define("DrawCircle", &BuiltinFunc{
		Name:  "DrawCircle",
		Arity: 4,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			cxVal, err := argInt(node, args, 0, "gfx.DrawCircle")
			if err != nil {
				return NilValue{}, err
			}

			cyVal, err := argInt(node, args, 1, "gfx.DrawCircle")
			if err != nil {
				return NilValue{}, err
			}

			rVal, err := argInt(node, args, 2, "gfx.DrawCircle")
			if err != nil {
				return NilValue{}, err
			}

			col, err := argColor(node, typeEnv, args, 3, "gfx.DrawCircle")
			if err != nil {
				return NilValue{}, err
			}

			cx := int32(cxVal)
			cy := int32(cyVal)
			r := float32(rVal)

			rl.DrawCircle(cx, cy, r, col)
			return NilValue{}, nil
		},
	})

	env.Define("DrawText", &BuiltinFunc{
		Name:  "DrawText",
		Arity: 5,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			txtVal, err := argString(node, args, 0, "gfx.DrawText")
			if err != nil {
				return NilValue{}, err
			}

			xVal, err := argInt(node, args, 1, "gfx.DrawText")
			if err != nil {
				return NilValue{}, err
			}

			yVal, err := argInt(node, args, 2, "gfx.DrawText")
			if err != nil {
				return NilValue{}, err
			}

			fontVal, err := argInt(node, args, 3, "gfx.DrawText")
			if err != nil {
				return NilValue{}, err
			}

			col, err := argColor(node, typeEnv, args, 4, "gfx.DrawText")
			if err != nil {
				return NilValue{}, err
			}

			txt := txtVal
			x := int32(xVal)
			y := int32(yVal)
			font := int32(fontVal)

			rl.DrawText(txt, x, y, font, col)
			return NilValue{}, nil
		},
	})

	env.Define("DrawLine", &BuiltinFunc{
		Name:  "DrawLine",
		Arity: 3,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			x1, y1, err := argVector2(node, i, typeEnv, args, 0, "gfx.DrawLine")
			if err != nil {
				return NilValue{}, err
			}

			x2, y2, err := argVector2(node, i, typeEnv, args, 1, "gfx.DrawLine")
			if err != nil {
				return NilValue{}, err
			}

			col, err := argColor(node, typeEnv, args, 2, "gfx.DrawLine")
			if err != nil {
				return NilValue{}, err
			}

			rl.DrawLine(int32(x1), int32(y1), int32(x2), int32(y2), col)
			return NilValue{}, nil
		},
	})

	module := ModuleValue{
		Name:    "gfx",
		Env:     env,
		typeEnv: typeEnv,
	}

	return module, nil
}
