package interpreter

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/z-sk1/ayla-lang/parser"
)

type Error struct {
	Message string
}

func (e Error) Error() string {
	return e.Message
}

func (e Error) Type() ValueType {
	return ERROR
}

func (e Error) String() string {
	return e.Message
}

func initBuiltinTypes(typeEnv map[string]TypeValue) {
	typeEnv["int"] = TypeValue{
		TypeInfo: &TypeInfo{
			Name:         "int",
			Kind:         TypeInt,
			IsComparable: true,
		},
	}

	typeEnv["float"] = TypeValue{
		TypeInfo: &TypeInfo{
			Name:         "float",
			Kind:         TypeFloat,
			IsComparable: true,
		},
	}
	typeEnv["string"] = TypeValue{
		TypeInfo: &TypeInfo{
			Name:         "string",
			Kind:         TypeString,
			IsComparable: true,
		},
	}

	typeEnv["bool"] = TypeValue{
		TypeInfo: &TypeInfo{
			Name:         "bool",
			Kind:         TypeBool,
			IsComparable: true,
		},
	}

	typeEnv["nil"] = TypeValue{
		TypeInfo: &TypeInfo{
			Name:         "nil",
			Kind:         TypeNil,
			IsComparable: true,
		},
	}

	typeEnv["thing"] = TypeValue{
		TypeInfo: &TypeInfo{
			Name: "thing",
			Kind: TypeAny,
		},
	}

	typeEnv["error"] = TypeValue{
		TypeInfo: &TypeInfo{
			Name: "error",
			Kind: TypeInterface,
			Methods: map[string]*Func{
				"Error": &Func{
					TypeName: &TypeInfo{
						Kind:    TypeFunc,
						Params:  []*TypeInfo{},
						Returns: []*TypeInfo{typeEnv["string"].TypeInfo},
					},
				},
			},
		},
	}
}

func (i *Interpreter) registerNativeModules() {
	i.nativeModules = map[string]NativeLoader{
		"math": LoadMathModule,
		"rand": LoadRandModule,
		"fs":   LoadFSModule,
		"time": LoadTimeModule,
		"gfx":  LoadGFXModule,
	}
}

func (i *Interpreter) registerBuiltins() {
	env := i.Env

	env.builtins["ord"] = &BuiltinFunc{
		Name:  "ord",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			s, err := argString(node, args, 0, "ord")
			if err != nil {
				return NilValue{}, err
			}

			r := []rune(s)
			if len(r) != 1 {
				return NilValue{}, NewRuntimeError(node, "ord expects single character")
			}

			return IntValue{V: int(r[0])}, nil
		},
	}

	env.builtins["chr"] = &BuiltinFunc{
		Name:  "chr",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := argInt(node, args, 0, "chr")
			if err != nil {
				return NilValue{}, err
			}

			return StringValue{V: string(rune(v))}, nil
		},
	}

	env.builtins["len"] = &BuiltinFunc{
		Name:  "len",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]

			switch v.Type() {
			case STRING:
				return IntValue{V: len(v.(StringValue).V)}, nil
			case ARR:
				return IntValue{V: len(v.(ArrayValue).Elements)}, nil
			case MAP:
				return IntValue{V: len(v.(MapValue).Entries)}, nil
			default:
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("len: type %s not supported", i.typeInfoFromValue(v).Name))
			}
		},
	}

	env.builtins["cap"] = &BuiltinFunc{
		Name:  "cap",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]

			switch v.Type() {
			case ARR:
				return IntValue{V: cap(v.(ArrayValue).Elements)}, nil
			default:
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cap: type %s not supported", i.typeInfoFromValue(v).Name))
			}
		},
	}

	env.builtins["make"] = &BuiltinFunc{
		Name:  "make",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, NewRuntimeError(node, "make: expected at least one argument")
			}

			typeVal, err := argType(node, args, 0, "make")
			if err != nil {
				return NilValue{}, err
			}

			ti := typeVal.TypeInfo

			switch ti.Kind {
			case TypeArray:
				if len(args) < 2 {
					return NilValue{}, NewRuntimeError(node, "make: expected second argument, length, for arrays")
				}

				length, err := argInt(node, args, 1, "make")
				if err != nil {
					return NilValue{}, err
				}

				capacity := length

				if len(args) == 3 {
					var err error
					capacity, err = argInt(node, args, 2, "make")

					if err != nil {
						return NilValue{}, err
					}
				}

				if capacity < length {
					return NilValue{}, NewRuntimeError(node, "make: capacity must be >= length")
				}

				elements := make([]Value, length)

				for idx := range length {
					elem, err := i.defaultValueFromTypeInfo(node, ti.Elem)
					if err != nil {
						return NilValue{}, err
					}

					elements[idx] = elem
				}

				return ArrayValue{
					Elements: elements,
					ElemType: ti.Elem,
					Capacity: capacity,
					Fixed:    false,
				}, nil
			case TypeMap:
				m := make(map[Value]Value)
				return MapValue{
					Entries:   m,
					KeyType:   ti.Key,
					ValueType: ti.Value,
				}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "make: slices, arrays and maps are only supported")
			}
		},
	}

	env.builtins["append"] = &BuiltinFunc{
		Name:  "append",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			slice, err := argArray(node, args, 0, "append")
			if err != nil {
				return NilValue{}, err
			}

			elemType := slice.ElemType

			for idx, arg := range args[1:] {
				argType := i.typeInfoFromValue(arg)
				if !typesAssignable(argType, elemType) {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("append: arg %d expected '%s' but got '%s'", idx, elemType.Name, argType.Name))
				}

				slice.Elements = append(slice.Elements, arg)
			}

			return slice, nil
		},
	}

	env.builtins["delete"] = &BuiltinFunc{
		Name:  "delete",
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			ident, ok := node.Args[0].(*parser.Identifier)

			val, ok2, isConst := i.Env.Get(ident.Value)
			if !ok2 {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown var: '%s'", ident.Value))
			}

			if !ok {
				_, ok = args[0].(MapValue)
				if !ok {
					return NilValue{}, NewRuntimeError(node, "delete: first argument must be a map")
				}
			}

			if isConst {
				return NilValue{}, NewRuntimeError(node, "delete: cannot assign to a constant")
			}

			expectedTI := args[0].(MapValue).KeyType

			key, err := i.assignWithType(node, args[1], expectedTI)
			if err != nil {
				return NilValue{}, err
			}

			delete(val.(MapValue).Entries, key)
			i.Env.Set(ident.Value, val)
			return NilValue{}, nil
		},
	}

	env.builtins["typeof"] = &BuiltinFunc{
		Name:  "typeof",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]

			switch v := v.(type) {
			case ArrayValue:
				if v.Fixed {
					return StringValue{V: fmt.Sprintf("[%d]%s", v.Capacity, v.ElemType.Name)}, nil
				}

				return StringValue{V: fmt.Sprintf("[]%s", v.ElemType.Name)}, nil
			case NamedValue:
				return StringValue{V: v.TypeName.Name}, nil
			}

			return StringValue{V: i.typeInfoFromValue(v).Name}, nil
		},
	}

	env.builtins["put"] = &BuiltinFunc{
		Name:  "put",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			if len(args) == 0 {
				fmt.Print()
				return NilValue{}, nil
			}

			for _, v := range args {
				ti := unwrapAlias(i.typeInfoFromValue(v))

				if ti != nil && typesAssignable(ti, i.typeEnv["error"].TypeInfo) {
					method, ok := i.Env.GetMethod(ti, "Error")
					if ok {
						receiver := v
						if iv, ok := v.(InterfaceValue); ok {
							receiver = iv.Value
						}

						res, err := i.callFunction(method, []Value{receiver}, node)
						if err != nil {
							return NilValue{}, err
						}

						fmt.Print(res.String())
						continue
					}
				}

				fmt.Print(v.String())
			}

			return NilValue{}, nil
		},
	}

	env.builtins["putln"] = &BuiltinFunc{
		Name:  "putln",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			for idx, v := range args {
				if idx > 0 {
					fmt.Print(" ")
				}

				ti := unwrapAlias(i.typeInfoFromValue(v))

				if ti != nil && typesAssignable(ti, i.typeEnv["error"].TypeInfo) {
					method, ok := i.Env.GetMethod(ti, "Error")
					if ok {
						receiver := v
						if iv, ok := v.(InterfaceValue); ok {
							receiver = iv.Value
						}

						res, err := i.callFunction(method, []Value{receiver}, node)
						if err != nil {
							return NilValue{}, err
						}

						fmt.Print(res.String())
						continue
					}
				}

				fmt.Print(v.String())
			}

			fmt.Println()
			return NilValue{}, nil
		},
	}

	env.builtins["putf"] = &BuiltinFunc{
		Name:  "putf",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			if len(args) == 0 {
				return NilValue{}, NewRuntimeError(node, "putf: expected at least one argument")
			}

			format, err := argString(node, args, 0, "putf")
			if err != nil {
				return NilValue{}, err
			}

			goArgs := []any{}
			for _, v := range args[1:] {
				switch val := v.(type) {
				case IntValue:
					goArgs = append(goArgs, val.V)
				case FloatValue:
					goArgs = append(goArgs, val.V)
				case StringValue:
					goArgs = append(goArgs, val.V)
				case BoolValue:
					goArgs = append(goArgs, val.V)
				default:
					goArgs = append(goArgs, val.String())
				}
			}

			fmt.Printf(format, goArgs...)
			return NilValue{}, nil
		},
	}

	env.builtins["explode"] = &BuiltinFunc{
		Name:  "explode",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			var msg string

			switch v := args[0].(type) {
			case StringValue:
				msg = v.V
			case InterfaceValue:
				if typesAssignable(v.TypeInfo, i.typeEnv["error"].TypeInfo) {
					msg = v.Value.(Error).Error()
				}
			default:
				msg = v.String()
			}

			return NilValue{}, NewRuntimeError(node, msg)
		},
	}

	env.builtins["explodef"] = &BuiltinFunc{
		Name:  "explodef",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			if len(args) == 0 {
				return NilValue{}, NewRuntimeError(node, "explodef: expected at least one argument")
			}

			format, err := argString(node, args, 0, "explodef")
			if err != nil {
				return NilValue{}, err
			}

			goArgs := []any{}
			for _, v := range args[1:] {
				switch val := v.(type) {
				case IntValue:
					goArgs = append(goArgs, val.V)
				case FloatValue:
					goArgs = append(goArgs, val.V)
				case StringValue:
					goArgs = append(goArgs, val.V)
				case BoolValue:
					goArgs = append(goArgs, val.V)
				default:
					goArgs = append(goArgs, val.String())
				}
			}

			msg := fmt.Sprintf(format, goArgs...)

			return NilValue{}, NewRuntimeError(node, msg)
		},
	}

	env.builtins["scanln"] = &BuiltinFunc{
		Name:  "scanln",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			reader := bufio.NewReader(os.Stdin)
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				return NilValue{}, err
			}

			fields := strings.Fields(line)

			if len(fields) < len(args) {
				return NilValue{}, NewRuntimeError(node, "scanln: not enough input values")
			}

			for idx := 0; idx < len(args) && idx < len(fields); idx++ {
				ass, ok := args[idx].(Assignable)
				if !ok {
					return NilValue{}, NewRuntimeError(node, "scanln: assignable values expected")
				}

				val, err := ass.Get(i)
				if err != nil {
					return NilValue{}, NewRuntimeError(node, err.Error())
				}

				input := fields[idx]

				err = i.assignInput(node, ass, val, input, "scanln")
				if err != nil {
					return NilValue{}, err
				}
			}

			return NilValue{}, nil
		},
	}

	env.builtins["scan"] = &BuiltinFunc{
		Name:  "scan",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {

			reader := bufio.NewReader(os.Stdin)

			for _, arg := range args {
				ass, ok := arg.(Assignable)
				if !ok {
					return NilValue{}, NewRuntimeError(node, "scan: assignable values expected")
				}

				val, err := ass.Get(i)
				if err != nil {
					return NilValue{}, NewRuntimeError(node, err.Error())
				}

				var input string
				_, err = fmt.Fscan(reader, &input)
				if err != nil {
					if err == io.EOF {
						return NilValue{}, NewRuntimeError(node, "scan: unexpected end of input")
					}
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("scan: %s", err.Error()))
				}

				err = i.assignInput(node, ass, val, input, "scan")
				if err != nil {
					return NilValue{}, err
				}
			}

			return NilValue{}, nil
		},
	}

	env.builtins["scanf"] = &BuiltinFunc{
		Name:  "scanf",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {

			if len(args) < 2 {
				return NilValue{}, NewRuntimeError(node, "scanf: format and variables required")
			}

			format, err := argString(node, args, 0, "scanf")
			if err != nil {
				return NilValue{}, err
			}

			reader := bufio.NewReader(os.Stdin)

			var scanArgs []any
			var setters []func()

			for _, arg := range args[1:] {

				ass, ok := arg.(Assignable)
				if !ok {
					return NilValue{}, NewRuntimeError(node, "scanf: assignable value expected")
				}

				val, err := ass.Get(i)
				if err != nil {
					return NilValue{}, err
				}

				switch val.(type) {

				case IntValue:
					var v int
					scanArgs = append(scanArgs, &v)
					setters = append(setters, func() {
						ass.Set(i, IntValue{V: v})
					})

				case FloatValue:
					var v float64
					scanArgs = append(scanArgs, &v)
					setters = append(setters, func() {
						ass.Set(i, FloatValue{V: v})
					})

				case BoolValue:
					var v bool
					scanArgs = append(scanArgs, &v)
					setters = append(setters, func() {
						ass.Set(i, BoolValue{V: v})
					})

				case StringValue:
					var v string
					scanArgs = append(scanArgs, &v)
					setters = append(setters, func() {
						ass.Set(i, StringValue{V: v})
					})

				default:
					return NilValue{}, NewRuntimeError(node, "scanf: unsupported type")
				}
			}

			_, err = fmt.Fscanf(reader, format, scanArgs...)
			if err != nil {
				return NilValue{}, err
			}

			for _, set := range setters {
				set()
			}

			return NilValue{}, nil
		},
	}

	env.builtins["scankey"] = &BuiltinFunc{
		Name:  "scankey",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			ass, ok := args[0].(Assignable)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "scankey: first argument must be an assignable value")
			}

			v, err := ass.Get(i)
			if err != nil {
				return NilValue{}, NewRuntimeError(node, err.Error())
			}

			ch, err := readKey()
			if err != nil {
				return NilValue{}, NewRuntimeError(node, err.Error())
			}

			expectedTI := i.typeInfoFromValue(v)
			expectedTI = unwrapAlias(expectedTI)

			var newVal Value
			switch expectedTI.Kind {
			case TypeString:
				newVal = StringValue{V: string(ch)}
			case TypeInt:
				newVal = IntValue{V: int(ch)}
			default:
				return NilValue{}, NewRuntimeError(node, "scankey: unsupported type")
			}

			ass.Set(i, newVal)
			return NilValue{}, nil
		},
	}
}
