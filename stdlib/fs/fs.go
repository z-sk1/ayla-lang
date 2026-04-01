package fs

import (
	"io"
	"os"
	"path/filepath"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/registry"
)

func init() {
	registry.Register("fs", Load)
}

func Load(i *interpreter.Interpreter) (interpreter.ModuleValue, error) {
	env := interpreter.NewEnvironment(i.Env)

	env.Define("Create", &interpreter.BuiltinFunc{
		Name:  "Create",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "fs.Create")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			dir := filepath.Dir(path)

			err = os.MkdirAll(dir, 0755)
			if err != nil {
				return interpreter.InterfaceValue{
					TypeInfo: i.TypeEnv["error"].TypeInfo,
					Value:    interpreter.Error{Message: err.Error()},
				}, nil
			}

			_, err = os.Create(path)
			if err != nil {
				return interpreter.InterfaceValue{
					TypeInfo: i.TypeEnv["error"].TypeInfo,
					Value:    interpreter.Error{Message: err.Error()},
				}, nil
			}

			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("Write", &interpreter.BuiltinFunc{
		Name:  "Write",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "fs.Write")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			content, err := interpreter.ArgString(node, args, 1, "fs.Write")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			err = os.WriteFile(path, []byte(content), 0644)
			if err != nil {
				return interpreter.InterfaceValue{
					TypeInfo: i.TypeEnv["error"].TypeInfo,
					Value:    interpreter.Error{Message: err.Error()},
				}, nil
			}

			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("Read", &interpreter.BuiltinFunc{
		Name:  "Read",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {

			path, err := interpreter.ArgString(node, args, 0, "fs.Read")

			data, err := os.ReadFile(path)
			if err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.InterfaceValue{
							TypeInfo: i.TypeEnv["error"].TypeInfo,
							Value:    interpreter.Error{Message: err.Error()},
						},
					},
				}, nil
			}

			return interpreter.TupleValue{
				Values: []interpreter.Value{
					interpreter.StringValue{V: string(data)},
					interpreter.NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("Exists", &interpreter.BuiltinFunc{
		Name:  "Exists",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "fs.Exists")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			_, err = os.Stat(path)
			if err == nil {
				return interpreter.TupleValue{Values: []interpreter.Value{
					interpreter.BoolValue{V: true},
					interpreter.NilValue{},
				}}, nil
			}

			if os.IsNotExist(err) {
				return interpreter.TupleValue{Values: []interpreter.Value{
					interpreter.BoolValue{V: false},
					interpreter.NilValue{},
				}}, nil
			}

			return interpreter.TupleValue{Values: []interpreter.Value{
				interpreter.NilValue{},
				interpreter.InterfaceValue{
					TypeInfo: i.TypeEnv["error"].TypeInfo,
					Value:    interpreter.Error{Message: err.Error()},
				},
			}}, nil
		},
	}, false)

	env.Define("Append", &interpreter.BuiltinFunc{
		Name:  "Append",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "fs.Append")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			data, err := interpreter.ArgString(node, args, 1, "fs.Append")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			file, err := os.OpenFile(
				path,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY,
				0644,
			)

			if err != nil {
				return interpreter.InterfaceValue{
					TypeInfo: i.TypeEnv["error"].TypeInfo,
					Value:    interpreter.Error{Message: err.Error()},
				}, nil
			}
			defer file.Close()

			_, err = file.WriteString(data)
			if err != nil {
				return interpreter.InterfaceValue{
					TypeInfo: i.TypeEnv["error"].TypeInfo,
					Value:    interpreter.Error{Message: err.Error()},
				}, nil
			}

			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("Delete", &interpreter.BuiltinFunc{
		Name:  "Delete",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "fs.Delete")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			err = os.Remove(path)
			if err != nil {
				return interpreter.NilValue{}, interpreter.NewRuntimeError(node, err.Error())
			}

			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("List", &interpreter.BuiltinFunc{
		Name:  "List",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "fs.List")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			entries, err := os.ReadDir(path)
			if err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.InterfaceValue{
							TypeInfo: i.TypeEnv["error"].TypeInfo,
							Value:    interpreter.Error{Message: err.Error()},
						},
					},
				}, nil
			}

			slice := make([]interpreter.Value, 0, len(entries))
			for _, e := range entries {
				slice = append(slice, interpreter.StringValue{V: e.Name()})
			}

			return interpreter.TupleValue{
				Values: []interpreter.Value{
					interpreter.ArrayValue{
						Elements: slice,
						ElemType: i.TypeEnv["string"].TypeInfo,
					},
					interpreter.NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("Mkdir", &interpreter.BuiltinFunc{
		Name:  "Mkdir",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "fs.Mkdir")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			err = os.MkdirAll(path, 0755)
			if err != nil {
				return interpreter.InterfaceValue{
					TypeInfo: i.TypeEnv["error"].TypeInfo,
					Value:    interpreter.Error{Message: err.Error()},
				}, nil
			}

			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("IsDir", &interpreter.BuiltinFunc{
		Name:  "IsDir",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "fs.IsDir")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			info, err := os.Stat(path)
			if err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.InterfaceValue{
							TypeInfo: i.TypeEnv["error"].TypeInfo,
							Value:    interpreter.Error{Message: err.Error()},
						},
					},
				}, nil
			}

			return interpreter.TupleValue{
				Values: []interpreter.Value{
					interpreter.BoolValue{V: info.IsDir()},
					interpreter.NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("Walk", &interpreter.BuiltinFunc{
		Name:  "Walk",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			root, err := interpreter.ArgString(node, args, 0, "fs.Walk")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			var files []interpreter.Value

			err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				files = append(files, interpreter.StringValue{V: path})
				return nil
			})

			if err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.InterfaceValue{
							TypeInfo: i.TypeEnv["error"].TypeInfo,
							Value:    interpreter.Error{Message: err.Error()},
						},
					},
				}, nil
			}

			return interpreter.TupleValue{
				Values: []interpreter.Value{
					interpreter.ArrayValue{Elements: files, ElemType: i.TypeEnv["string"].TypeInfo},
					interpreter.NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("Cwd", &interpreter.BuiltinFunc{
		Name:  "Cwd",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			wd, err := os.Getwd()
			if err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.InterfaceValue{
							TypeInfo: i.TypeEnv["error"].TypeInfo,
							Value:    interpreter.Error{Message: err.Error()},
						},
					},
				}, nil
			}

			return interpreter.TupleValue{
				Values: []interpreter.Value{
					interpreter.StringValue{V: wd},
					interpreter.NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("Size", &interpreter.BuiltinFunc{
		Name:  "Size",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "fs.Size")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			info, err := os.Stat(path)
			if err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.InterfaceValue{
							TypeInfo: i.TypeEnv["error"].TypeInfo,
							Value:    interpreter.Error{Message: err.Error()},
						},
					},
				}, nil
			}

			return interpreter.TupleValue{
				Values: []interpreter.Value{
					interpreter.IntValue{V: int(info.Size())},
					interpreter.NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("ModTime", &interpreter.BuiltinFunc{
		Name:  "ModTime",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "fs.ModTime")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			info, err := os.Stat(path)
			if err != nil {
				return interpreter.TupleValue{
					Values: []interpreter.Value{
						interpreter.NilValue{},
						interpreter.InterfaceValue{
							TypeInfo: i.TypeEnv["error"].TypeInfo,
							Value:    interpreter.Error{Message: err.Error()},
						},
					},
				}, nil
			}

			return interpreter.TupleValue{
				Values: []interpreter.Value{
					interpreter.IntValue{V: int(info.ModTime().Unix())},
					interpreter.NilValue{},
				},
			}, nil
		},
	}, false)

	env.Define("Rename", &interpreter.BuiltinFunc{
		Name:  "Rename",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			old, err := interpreter.ArgString(node, args, 0, "fs.Rename")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			new, err := interpreter.ArgString(node, args, 1, "fs.Rename")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			err = os.Rename(old, new)
			if err != nil {
				return interpreter.InterfaceValue{
					TypeInfo: i.TypeEnv["error"].TypeInfo,
					Value:    interpreter.Error{Message: err.Error()},
				}, nil
			}

			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("Copy", &interpreter.BuiltinFunc{
		Name:  "Copy",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			src, err := interpreter.ArgString(node, args, 0, "fs.Copy")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			dst, err := interpreter.ArgString(node, args, 1, "fs.Copy")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			in, err := os.Open(src)
			if err != nil {
				return interpreter.InterfaceValue{
					TypeInfo: i.TypeEnv["error"].TypeInfo,
					Value:    interpreter.Error{Message: err.Error()},
				}, nil
			}
			defer in.Close()

			out, err := os.Create(dst)
			if err != nil {
				return interpreter.InterfaceValue{
					TypeInfo: i.TypeEnv["error"].TypeInfo,
					Value:    interpreter.Error{Message: err.Error()},
				}, nil
			}
			defer out.Close()

			_, err = io.Copy(out, in)
			if err != nil {
				return interpreter.InterfaceValue{
					TypeInfo: i.TypeEnv["error"].TypeInfo,
					Value:    interpreter.Error{Message: err.Error()},
				}, nil
			}

			return interpreter.NilValue{}, nil
		},
	}, false)

	module := interpreter.ModuleValue{
		Name: "fs",
		Env:  env,
	}

	return module, nil
}
