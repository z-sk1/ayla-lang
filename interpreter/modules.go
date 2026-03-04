package interpreter

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/z-sk1/ayla-lang/parser"
)

type NativeLoader func(i *Interpreter) (ModuleValue, error)

func wrapFloat1(name string, fn func(float64) float64) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			f, ok := toFloat(unwrapNamed(args[0]))
			if !ok {
				return NilValue{}, NewRuntimeError(node, "expected number")
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
			f1, ok := toFloat(unwrapNamed(args[0]))
			if !ok {
				return NilValue{}, NewRuntimeError(node, "expected number")
			}

			f2, ok := toFloat(unwrapNamed(args[1]))
			if !ok {
				return NilValue{}, NewRuntimeError(node, "expected number")
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
				maxV, ok := unwrapNamed(args[0]).(IntValue)
				if !ok {
					return NilValue{}, NewRuntimeError(node, "rand.Int: first argument must be int")
				}

				if maxV.V <= 0 {
					return NilValue{}, NewRuntimeError(node, "rand.Int: first argument must > 0")
				}

				max := maxV.V
				n := rand.Intn(max) + 1

				return IntValue{V: n}, nil
			case 2:
				minV, ok1 := unwrapNamed(args[0]).(IntValue)
				maxV, ok2 := unwrapNamed(args[1]).(IntValue)

				if !ok1 || !ok2 {
					return NilValue{}, NewRuntimeError(node, "rand.Int: both arguments must be an int")
				}

				min := minV.V
				max := maxV.V

				if min > max {
					min, max = max, min
				}

				n := rand.Intn(max-min+1) + min
				return IntValue{V: n}, nil
			}
			return NilValue{}, NewRuntimeError(node, "invalid amount of args, rand.Int expects 0-2 args")
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
				maxV, ok := toFloat(unwrapNamed(args[0]))
				if !ok {
					return NilValue{}, NewRuntimeError(node, "rand.Float: first argument must be a float")
				}

				n := rand.Float64() * maxV
				return FloatValue{V: n}, nil
			case 2:
				minV, ok1 := toFloat(unwrapNamed(args[0]))
				maxV, ok2 := toFloat(unwrapNamed(args[1]))

				if !ok1 || !ok2 {
					return NilValue{}, NewRuntimeError(node, "rand.Float: both arguments must be a float")
				}

				if minV > maxV {
					minV, maxV = maxV, minV
				}

				n := rand.Float64()*(maxV-minV+1) + minV
				return FloatValue{V: n}, nil
			}
			return NilValue{}, NewRuntimeError(node, "invalid amount of args, rand.Float expects 0-2 args")
		},
	})

	env.Define("Choice", &BuiltinFunc{
		Name:  "Choice",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			arr, ok := unwrapNamed(args[0]).(ArrayValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("expected array or slice but got '%s'", i.typeInfoFromValue(args[0]).Name))
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
			path, ok := unwrapNamed(args[0]).(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("fs.Write: first argument must be a string, got '%s'", i.typeInfoFromValue(args[0]).Name))
			}

			content, ok := unwrapNamed(args[1]).(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("fs.Write: second argument must be a string, got '%s'", i.typeInfoFromValue(args[1]).Name))
			}

			err := os.WriteFile(path.V, []byte(content.V), 0644)
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

			path, ok := unwrapNamed(args[0]).(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "fs.Read: argument must be string")
			}

			data, err := os.ReadFile(path.V)
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
			v := unwrapNamed(args[0])

			pathVal, ok := v.(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "fs.Read: argument must be string")
			}

			path := pathVal.V

			_, err := os.Stat(path)
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
			pathVal, ok1 := unwrapNamed(args[0]).(StringValue)
			dataVal, ok2 := unwrapNamed(args[1]).(StringValue)

			if !ok1 || !ok2 {
				return NilValue{}, NewRuntimeError(node, "fs.Append: both arguments must be a string")
			}

			path := pathVal.V
			data := dataVal.V

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
			pathVal, ok := unwrapNamed(args[0]).(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "fs.Delete: argument must be a string")
			}

			path := pathVal.V

			err := os.Remove(path)
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
			pathVal, ok := unwrapNamed(args[0]).(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "fs.List: argument must be a string")
			}

			path := pathVal.V

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
						ElemType: i.typeEnv["string"],
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
			pathVal, ok := unwrapNamed(args[0]).(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "fs.Mkdir: argument must be a string")
			}

			path := pathVal.V

			err := os.MkdirAll(path, 0755)
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
			pathVal, ok := unwrapNamed(args[0]).(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "fs.IsDir: argument must be a string")
			}

			path := pathVal.V

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
			root, ok := unwrapNamed(args[0]).(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "fs.Walk: argument must be a string")
			}

			var files []Value

			err := filepath.Walk(root.V, func(path string, info os.FileInfo, err error) error {
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
					ArrayValue{Elements: files, ElemType: i.typeEnv["string"]},
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
			pathVal, ok := unwrapNamed(args[0]).(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "fs.Size: argument must be a string")
			}

			path := pathVal.V

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
			pathVal, ok := unwrapNamed(args[0]).(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "fs.ModTime: argument must be a string")
			}

			path := pathVal.V

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
		Name: "Rename",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			oldVal, ok1 := unwrapNamed(args[0]).(StringValue)
			newVal, ok2 := unwrapNamed(args[1]).(StringValue)
			
			if !ok1 || !ok2 {
				return NilValue{}, NewRuntimeError(node, "fs.Rename: both arguments must be a string")
			}
			
			old := oldVal.V
			new := newVal.V
			
			err := os.Rename(old, new)
			if err != nil {
				return ErrorValue{
					V: NewRuntimeError(node, err.Error()),
				}, nil
			}
			
			return NilValue{}, nil
		},
	})
	
	env.Define("Copy", &BuiltinFunc{
		Name: "Copy",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			srcVal, ok1 := unwrapNamed(args[0]).(StringValue)
			dstVal, ok2 := unwrapNamed(args[1]).(StringValue)
			
			if !ok1 || !ok2 {
				return NilValue{}, NewRuntimeError(node, "fs.Copy: both arguments must be a string")
			}
			
			src := srcVal.V
			dst := dstVal.V
			
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
