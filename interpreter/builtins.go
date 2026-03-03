package interpreter

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/z-sk1/ayla-lang/parser"
)

func initBuiltinTypes(typeEnv map[string]*TypeInfo) {
	typeEnv["int"] = &TypeInfo{
		Name:         "int",
		Kind:         TypeInt,
		IsComparable: true,
	}

	typeEnv["float"] = &TypeInfo{
		Name:         "float",
		Kind:         TypeFloat,
		IsComparable: true,
	}

	typeEnv["string"] = &TypeInfo{
		Name:         "string",
		Kind:         TypeString,
		IsComparable: true,
	}

	typeEnv["bool"] = &TypeInfo{
		Name:         "bool",
		Kind:         TypeBool,
		IsComparable: true,
	}

	typeEnv["nil"] = &TypeInfo{
		Name:         "nil",
		Kind:         TypeNil,
		IsComparable: true,
	}

	typeEnv["thing"] = &TypeInfo{
		Name: "thing",
		Kind: TypeAny,
	}

	typeEnv["error"] = &TypeInfo{
		Name: "error",
		Kind: TypeError,
	}
}

func (i *Interpreter) registerBuiltins() {
	env := i.env

	env.builtins["toInt"] = &BuiltinFunc{
		Name:  "toInt",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v := args[0]
			v = unwrapNamed(v)

			switch v.Type() {
			case INT:
				return IntValue{V: v.(IntValue).V}, nil
			case FLOAT:
				return IntValue{V: int(v.(FloatValue).V)}, nil
			case BOOL:
				if v.(BoolValue).V {
					return IntValue{V: 1}, nil
				}
				return IntValue{V: 0}, nil
			case STRING:
				n, err := strconv.Atoi(v.(StringValue).V)
				if err != nil {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("could not parse string to int: %s", err.Error()))
				}
				return IntValue{V: n}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "unsupported toInt() parse")
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
				return FloatValue{V: v.(FloatValue).V}, nil
			case INT:
				return FloatValue{V: float64(v.(IntValue).V)}, nil
			case BOOL:
				if v.(BoolValue).V {
					return FloatValue{V: 1.0}, nil
				}
				return FloatValue{V: 0.0}, nil
			case STRING:
				n, err := strconv.ParseFloat(v.(StringValue).V, 64)
				if err != nil {
					return NilValue{}, NewRuntimeError(node, "could not convert string to float")
				}
				return FloatValue{V: n}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "unsupported toFloat() parse")
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
				return StringValue{V: v.String()}, nil
			default:
				return NilValue{}, NewRuntimeError(node, "unsupported toString() parse")
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
				return BoolValue{V: v.(BoolValue).V}, nil
			case INT:
				return BoolValue{V: v.(IntValue).V != 0}, nil
			case FLOAT:
				return BoolValue{V: v.(FloatValue).V != 0}, nil
			case STRING:
				s := strings.ToLower(v.(StringValue).V)
				if s == "true" || s == "yes" || s == "1" {
					return BoolValue{V: true}, nil
				}
				if s == "false" || s == "no" || s == "0" || s == "" {
					return BoolValue{V: false}, nil
				}
				return NilValue{}, NewRuntimeError(node, "unsupported toBool() parse")
			default:
				return NilValue{}, NewRuntimeError(node, "unsupported toBool() parse")
			}
		},
	}

	env.builtins["toArr"] = &BuiltinFunc{
		Name:  "toArr",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			if len(args) == 0 {
				return NilValue{}, NewRuntimeError(node, "toArr requires at least one argument")
			}

			elemType := unwrapAlias(i.typeInfoFromValue(args[0]))

			elements := make([]Value, 0, len(args))
			elements = append(elements, args[0])

			for idx := 1; idx < len(args); idx++ {
				t := unwrapAlias(i.typeInfoFromValue(args[idx]))

				if !typesAssignable(t, elemType) {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("toArr argument %d expected %s but got %s (use []thing for mixed arrays)", idx, elemType.Name, t.Name))
				}

				elements = append(elements, args[idx])
			}

			return ArrayValue{
				Elements: elements,
				ElemType: elemType,
			}, nil
		},
	}

	env.builtins["ord"] = &BuiltinFunc{
		Name:  "ord",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			s, ok := args[0].(StringValue)
			if !ok {
				return NilValue{}, fmt.Errorf("ord expects string")
			}

			r := []rune(s.V)
			if len(r) != 1 {
				return NilValue{}, fmt.Errorf("ord expects single character")
			}

			return IntValue{V: int(r[0])}, nil
		},
	}

	env.builtins["chr"] = &BuiltinFunc{
		Name:  "chr",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			if len(args) != 1 {
				return NilValue{}, fmt.Errorf("chr expects 1 argument")
			}

			v, ok := args[0].(IntValue)
			if !ok {
				return NilValue{}, fmt.Errorf("chr expects int")
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

			ti := typeVal.TypeName

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

			val, ok2 := i.env.Get(ident.Value)
			if !ok2 {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown var: '%s'", ident.Value))
			}

			if !ok {
				_, ok = args[0].(MapValue)
				if !ok {
					return NilValue{}, NewRuntimeError(node, "delete(map[T]T, key)")
				}
			}

			delete(val.(MapValue).Entries, args[1])
			i.env.Set(ident.Value, val)
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

			return StringValue{V: string(v.Type())}, nil
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
					return NilValue{}, NewRuntimeError(node, "scanln(t ...thing) expected")
				}

				varName := ident.Value
				val, ok := i.env.Get(varName)
				if !ok {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown var: %s", varName))
				}

				input := fields[idx]

				err := i.env.assignInput(node, varName, val, input)
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

				val, ok := i.env.Get(varName)
				if !ok {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown var: %s", varName))
				}

				if _, isConst := val.(ConstValue); isConst {
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

				err = i.env.assignInput(node, varName, val, input)
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
				return NilValue{}, NewRuntimeError(node, "scanf(format string, t ...thing) expected")
			}

			formatVal, ok := args[0].(StringValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "scanf(format string, t ...thing) expected")
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

				val, ok := i.env.Get(varName)
				if !ok {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown var: %s", varName))
				}

				if _, isConst := val.(ConstValue); isConst {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot assign to const: %s", varName))
				}

				input := fields[idx]

				switch val.(type) {

				case IntValue:
					n, err := strconv.Atoi(input)
					if err != nil {
						return NilValue{}, NewRuntimeError(node, "invalid int input")
					}
					i.env.Set(varName, IntValue{V: n})

				case FloatValue:
					f, err := strconv.ParseFloat(input, 64)
					if err != nil {
						return NilValue{}, NewRuntimeError(node, "invalid float input")
					}
					i.env.Set(varName, FloatValue{V: f})

				case BoolValue:
					b, err := strconv.ParseBool(input)
					if err != nil {
						return NilValue{}, NewRuntimeError(node, "invalid bool input")
					}
					i.env.Set(varName, BoolValue{V: b})

				case StringValue:
					i.env.Set(varName, StringValue{V: input})

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
			v, ok := i.env.Get(varName)
			if ok {
				if _, isConst := v.(ConstValue); isConst {
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

			i.env.Set(varName, newVal)
			return NilValue{}, nil
		},
	}

	env.builtins["wait"] = &BuiltinFunc{
		Name:  "wait",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			intVal, ok := args[0].(IntValue)
			if !ok {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("wait arg: %v, must be int (milliseconds)", intVal.V))
			}

			ms := intVal.V
			if ms < 0 {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("wait duration: %d, cannot be negative", ms))
			}

			time.Sleep(time.Duration(ms) * time.Millisecond)
			return NilValue{}, nil
		},
	}

	env.builtins["randi"] = &BuiltinFunc{
		Name:  "randi",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			switch len(args) {
			case 0:
				n := rand.Intn(2)
				return IntValue{V: n}, nil
			case 1:
				maxV, ok := args[0].(IntValue)
				if !ok {
					return NilValue{}, NewRuntimeError(node, "randi expects (int <= 0)")
				}

				if maxV.V <= 0 {
					return NilValue{}, NewRuntimeError(node, "randi expects (int <= 0)")
				}

				max := maxV.V
				n := rand.Intn(max) + 1

				return IntValue{V: n}, nil
			case 2:
				minV, ok1 := args[0].(IntValue)
				maxV, ok2 := args[1].(IntValue)

				if !ok1 || !ok2 {
					return NilValue{}, NewRuntimeError(node, "randi expects 0-2 args")
				}

				min := minV.V
				max := maxV.V

				if min > max {
					min, max = max, min
				}

				n := rand.Intn(max-min+1) + min
				return IntValue{V: n}, nil
			}
			return NilValue{}, NewRuntimeError(node, "invalid amount of args, randi expects 0-2 args")
		},
	}

	env.builtins["randf"] = &BuiltinFunc{
		Name:  "randf",
		Arity: -1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			switch len(args) {
			case 0:
				n := rand.Float64()
				return FloatValue{V: n}, nil
			case 1:
				maxV, ok := toFloat(args[0])
				if !ok {
					return NilValue{}, NewRuntimeError(node, "randf expects 0-2 args")
				}

				n := rand.Float64() * maxV
				return FloatValue{V: n}, nil
			case 2:
				minV, ok1 := toFloat(args[0])
				maxV, ok2 := toFloat(args[1])

				if !ok1 || !ok2 {
					return NilValue{}, NewRuntimeError(node, "randf expects 0-2 args")
				}

				if minV > maxV {
					minV, maxV = maxV, minV
				}

				n := rand.Float64()*(maxV-minV+1) + minV
				return FloatValue{V: n}, nil
			}
			return NilValue{}, NewRuntimeError(node, "invalid amount of args, randf expects 0-2 args")
		},
	}

	env.builtins["sin"] = &BuiltinFunc{
		Name:  "sin",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			val := args[0]

			switch v := val.(type) {
			case IntValue:
				val = FloatValue{V: float64(v.V)}
			}

			return FloatValue{V: math.Sin(val.(FloatValue).V)}, nil
		},
	}

	env.builtins["cos"] = &BuiltinFunc{
		Name:  "cos",
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			val := args[0]

			switch v := val.(type) {
			case IntValue:
				val = FloatValue{V: float64(v.V)}
			}

			return FloatValue{V: math.Cos(val.(FloatValue).V)}, nil
		},
	}
}
