package interpreter

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/z-sk1/ayla-lang/parser"
)

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
			Kind: TypeError,
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

	env.builtins["toInt"] = &BuiltinFunc{
		Name:  "toInt",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]
			v = unwrapNamed(v)

			switch v.Type() {
			case INT:
				return TupleValue{Values: []Value{IntValue{V: v.(IntValue).V}, NilValue{}}}, nil
			case FLOAT:
				return TupleValue{Values: []Value{IntValue{V: int(v.(FloatValue).V)}, NilValue{}}}, nil
			case BOOL:
				if v.(BoolValue).V {
					return TupleValue{Values: []Value{IntValue{V: 1}, NilValue{}}}, nil
				}
				return TupleValue{Values: []Value{IntValue{V: 0}, NilValue{}}}, nil
			case STRING:
				n, err := strconv.Atoi(v.(StringValue).V)
				if err != nil {
					return TupleValue{Values: []Value{NilValue{}, ErrorValue{
						V: NewRuntimeError(node, fmt.Sprintf("could not parse string to int: %s", err.Error())),
					}}}, nil
				}
				return TupleValue{Values: []Value{IntValue{V: n}, NilValue{}}}, nil
			default:
				return TupleValue{Values: []Value{NilValue{}, ErrorValue{
					V: NewRuntimeError(node, "unsupported toInt() parse"),
				}}}, nil
			}
		},
	}

	env.builtins["toFloat"] = &BuiltinFunc{
		Name:  "toFloat",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]
			v = unwrapNamed(v)

			switch v.Type() {
			case FLOAT:
				return TupleValue{Values: []Value{FloatValue{V: v.(FloatValue).V}, NilValue{}}}, nil
			case INT:
				return TupleValue{Values: []Value{FloatValue{V: float64(v.(IntValue).V)}, NilValue{}}}, nil
			case BOOL:
				if v.(BoolValue).V {
					return TupleValue{Values: []Value{FloatValue{V: 1.0}, NilValue{}}}, nil
				}
				return TupleValue{Values: []Value{FloatValue{V: 0.0}, NilValue{}}}, nil
			case STRING:
				n, err := strconv.ParseFloat(v.(StringValue).V, 64)
				if err != nil {
					return TupleValue{Values: []Value{NilValue{}, ErrorValue{
						V: NewRuntimeError(node, "could not convert string to float"),
					}}}, nil
				}
				return TupleValue{Values: []Value{FloatValue{V: n}, NilValue{}}}, nil
			default:
				return TupleValue{Values: []Value{NilValue{}, ErrorValue{
					V: NewRuntimeError(node, "unsupported toFloat() parse"),
				}}}, nil
			}
		},
	}

	env.builtins["toString"] = &BuiltinFunc{
		Name:  "toString",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]
			v = unwrapNamed(v)

			switch v.Type() {
			case INT,
				FLOAT,
				STRING,
				BOOL,
				ARR,
				STRUCT,
				TUPLE,
				NIL:
				return TupleValue{Values: []Value{StringValue{V: v.String()}}}, nil
			default:
				return TupleValue{Values: []Value{NilValue{}, ErrorValue{
					V: NewRuntimeError(node, "unsupported toString() parse"),
				}}}, nil
			}

		},
	}

	env.builtins["toBool"] = &BuiltinFunc{
		Name:  "toBool",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]
			v = unwrapNamed(v)

			switch v.Type() {
			case BOOL:
				return TupleValue{Values: []Value{BoolValue{V: v.(BoolValue).V}, NilValue{}}}, nil
			case INT:
				return TupleValue{Values: []Value{BoolValue{V: v.(IntValue).V != 0}, NilValue{}}}, nil
			case FLOAT:
				return TupleValue{Values: []Value{BoolValue{V: v.(FloatValue).V != 0}, NilValue{}}}, nil
			case STRING:
				s := strings.ToLower(v.(StringValue).V)
				if s == "true" || s == "yes" || s == "1" {
					return TupleValue{Values: []Value{BoolValue{V: true}, NilValue{}}}, nil
				}
				if s == "false" || s == "no" || s == "0" || s == "" {
					return TupleValue{Values: []Value{BoolValue{V: false}, NilValue{}}}, nil
				}
				return TupleValue{Values: []Value{NilValue{}, ErrorValue{
					V: NewRuntimeError(node, "unsupported toBool() parse"),
				}}}, nil
			default:
				return TupleValue{Values: []Value{NilValue{}, ErrorValue{
					V: NewRuntimeError(node, "unsupported toBool() parse"),
				}}}, nil
			}
		},
	}

	env.builtins["ord"] = &BuiltinFunc{
		Name:  "ord",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			s, ok := args[0].(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "ord expects string")
			}

			r := []rune(s.V)
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
			if len(args) != 1 {
				return NilValue{}, NewRuntimeError(node, "chr expects 1 argument")
			}

			v, ok := args[0].(IntValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "chr expects int")
			}

			return StringValue{V: string(rune(v.V))}, nil
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
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("len() not supported for type %s", string(v.Type())))
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
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cap() not supported for type %s", string(v.Type())))
			}
		},
	}

	env.builtins["make"] = &BuiltinFunc{
		Name:  "make",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			if len(args) < 1 {
				return NilValue{}, NewRuntimeError(node, "make() requires atleast a type arg")
			}

			typeVal, ok := args[0].(TypeValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "make([]T, len [, cap]) required")
			}

			ti := typeVal.TypeInfo

			switch ti.Kind {
			case TypeArray:
				if len(args) < 2 {
					return NilValue{}, NewRuntimeError(node, "make([]T, len [, cap]) required")
				}

				length := args[1].(IntValue).V
				capacity := length

				if len(args) == 3 {
					capacity = args[2].(IntValue).V
				}

				if capacity < length {
					return NilValue{}, NewRuntimeError(node, "capacity must be >= length")
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
				return NilValue{}, NewRuntimeError(node, "make() only supports slices and maps")
			}
		},
	}

	env.builtins["append"] = &BuiltinFunc{
		Name:  "append",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			slice, ok := args[0].(ArrayValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "append([]T, ...T) required")
			}
			elemType := slice.ElemType

			for idx, arg := range args[1:] {
				argType := i.typeInfoFromValue(arg)
				if !typesAssignable(argType, elemType) {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("arg %d expected '%s' but got '%s'", idx, elemType.Name, argType.Name))
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
				return NilValue{}, NewRuntimeError(node, "delete: map must not be a const")
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
				fmt.Print(v.String())
			}
			return NilValue{}, nil
		},
	}

	env.builtins["putln"] = &BuiltinFunc{
		Name:  "putln",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {

			for i, v := range args {
				if i > 0 {
					fmt.Print(" ")
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
				return NilValue{}, NewRuntimeError(node, "putf(format string, t ...thing) expected")
			}

			formatVal, ok := args[0].(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "putf(format string, t... thing) expected")
			}

			format := formatVal.V

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
			case ErrorValue:
				msg = v.V.Error()
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
				return NilValue{}, NewRuntimeError(node, "explodef(format string, t ...thing) expected")
			}

			formatVal, ok := args[0].(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "explodef(format string, t... thing) expected")
			}

			format := formatVal.V

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

			if len(node.Args) == 0 {
				return NilValue{}, NewRuntimeError(node, "scanln(t ...thing) expected")
			}

			reader := bufio.NewReader(os.Stdin)
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				return NilValue{}, err
			}

			fields := strings.Fields(line)

			if len(fields) < len(node.Args) {
				return NilValue{}, NewRuntimeError(node, "not enough input values")
			}

			for idx := 0; idx < len(node.Args) && idx < len(fields); idx++ {
				arg := node.Args[idx]

				ident, ok := arg.(*parser.Identifier)
				if !ok {
					return NilValue{}, NewRuntimeError(node, "scanln: first argument expects a variable name")
				}

				varName := ident.Value
				val, ok, isConst := i.Env.Get(varName)
				if !ok {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown var: %s", varName))
				}

				if isConst {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot assign to const: %s", varName))
				}

				input := fields[idx]

				err := i.Env.assignInput(node, varName, val, input)
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

			if len(node.Args) == 0 {
				return NilValue{}, NewRuntimeError(node, "scan(t ...thing) expected")
			}

			for _, arg := range node.Args {

				ident, ok := arg.(*parser.Identifier)
				if !ok {
					return NilValue{}, NewRuntimeError(node, "scan(t ...thing) expected")
				}

				varName := ident.Value

				val, ok, isConst := i.Env.Get(varName)
				if !ok {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown var: %s", varName))
				}

				if isConst {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot assign to const: %s", varName))
				}

				var input string
				_, err := fmt.Scan(&input)

				if err != nil {
					if err == io.EOF {
						return NilValue{}, NewRuntimeError(node, "unexpected end of input")
					}
					return NilValue{}, err
				}

				err = i.Env.assignInput(node, varName, val, input)
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

			if len(args) == 0 {
				return NilValue{}, NewRuntimeError(node, "scanf: atleast 2 arguments expected")
			}

			formatVal, ok := args[0].(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "scanf: first argument must be a string")
			}

			_ = formatVal.V // ignore format for now

			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')

			fields := strings.Fields(line)
			vars := node.Args[1:]

			if len(fields) < len(vars) {
				return NilValue{}, NewRuntimeError(node, "not enough input values")
			}

			for idx, arg := range vars {

				ident, ok := arg.(*parser.Identifier)
				if !ok {
					return NilValue{}, NewRuntimeError(node, "scanf(format string, t ...thing) expected")
				}

				varName := ident.Value

				val, ok, isConst := i.Env.Get(varName)
				if !ok {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown var: %s", varName))
				}

				if isConst {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot assign to const: %s", varName))
				}

				input := fields[idx]

				switch val.(type) {

				case IntValue:
					n, err := strconv.Atoi(input)
					if err != nil {
						return NilValue{}, NewRuntimeError(node, "invalid int input")
					}
					i.Env.Set(varName, IntValue{V: n})

				case FloatValue:
					f, err := strconv.ParseFloat(input, 64)
					if err != nil {
						return NilValue{}, NewRuntimeError(node, "invalid float input")
					}
					i.Env.Set(varName, FloatValue{V: f})

				case BoolValue:
					b, err := strconv.ParseBool(input)
					if err != nil {
						return NilValue{}, NewRuntimeError(node, "invalid bool input")
					}
					i.Env.Set(varName, BoolValue{V: b})

				case StringValue:
					i.Env.Set(varName, StringValue{V: input})

				case UninitializedValue, NilValue:
					return NilValue{}, NewRuntimeError(node, "variable must have a type before scan")

				default:
					return NilValue{}, NewRuntimeError(node, "unsupported type")
				}
			}

			return NilValue{}, nil
		},
	}

	env.builtins["scankey"] = &BuiltinFunc{
		Name:  "scankey",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			ident, ok := node.Args[0].(*parser.Identifier)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "scankey expects a variable")
			}

			varName := ident.Value

			// is it const?
			v, ok, isConst := i.Env.Get(varName)
			if ok {
				if isConst {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot assign to const: %s", varName))
				}
			} else {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown var: %s", varName))
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
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("scankey only supports int or string variables, got %s", string(v.Type())))
			}

			i.Env.Set(varName, newVal)
			return NilValue{}, nil
		},
	}
}
