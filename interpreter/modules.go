package interpreter

import (
	"fmt"

	"github.com/z-sk1/ayla-lang/parser"
)

type NativeLoader func(i *Interpreter) (ModuleValue, error)

func ExpectArgsRange(node parser.Node, args []Value, startRange, endRange int, name string) error {
	return NewRuntimeError(node, fmt.Sprintf("%s: expected %d-%d arguments, got %d", name, startRange, endRange, len(args)))
}

func ArgInt(node parser.Node, args []Value, i int, name string) (int, error) {
	v := UnwrapFully(args[i])
	iv, ok := v.(IntValue)
	if !ok {
		return 0, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be an int", name, i+1))
	}
	return iv.V, nil
}

func ArgFloat(node parser.Node, args []Value, i int, name string) (float64, error) {
	v, ok := toFloat(UnwrapFully(args[i]))
	if !ok {
		return 0, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be an float", name, i+1))
	}
	return v, nil
}

func ArgString(node parser.Node, args []Value, i int, name string) (string, error) {
	v := UnwrapFully(args[i])
	iv, ok := v.(StringValue)
	if !ok {
		return "", NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a string", name, i+1))
	}
	return iv.V, nil
}

func ArgBool(node parser.Node, args []Value, i int, name string) (bool, error) {
	v := UnwrapFully(args[i])
	iv, ok := v.(BoolValue)
	if !ok {
		return false, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a boolean", name, i+1))
	}
	return iv.V, nil
}

func ArgStruct(node parser.Node, args []Value, i int, name, sname string) (*StructValue, error) {
	v := UnwrapFully(args[i])
	sv, ok := v.(*StructValue)
	if !ok {
		return nil, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a %s", name, i+1, sname))
	}
	return sv, nil
}

func ArgType(node parser.Node, args []Value, i int, name string) (TypeValue, error) {
	v := UnwrapFully(args[i])
	tv, ok := v.(TypeValue)
	if !ok {
		return TypeValue{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a type signature", name, i+1))
	}
	return tv, nil
}

func ArgPointer(node parser.Node, args []Value, i int, name string) (*PointerValue, error) {
	v := UnwrapFully(args[i])
	pv, ok := v.(*PointerValue)
	if !ok {
		return &PointerValue{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a pointer", name, i+1))
	}
	return pv, nil
}

func ArgArray(node parser.Node, args []Value, i int, name string, elem string) (ArrayValue, error) {
	v := UnwrapFully(args[i])
	av, ok := v.(ArrayValue)
	if !ok {
		return ArrayValue{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a []%s", name, i+1, elem))
	}
	return av, nil
}

func ArgChan(node parser.Node, args []Value, i int, name string, elem string) (*Channel, error) {
	v := UnwrapFully(args[i])
	ch, ok := v.(*Channel)
	if !ok {
		return nil, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be chan %s", name, i+1, elem))
	}
	return ch, nil
}

func ArgChanRecv(node parser.Node, args []Value, i int, name string, elem string) (*Channel, error) {
	v := UnwrapFully(args[i])
	ch, ok := v.(*Channel)
	if !ok {
		return nil, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be chan %s", name, i+1, elem))
	}

	if !ch.canRecv {
		return nil, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a receive-capable channel", name, i+1))
	}

	return ch, nil
}

func ArgChanSend(node parser.Node, args []Value, i int, name string, elem string) (*Channel, error) {
	v := UnwrapFully(args[i])
	ch, ok := v.(*Channel)
	if !ok {
		return nil, NewRuntimeError(node, fmt.Sprintf(
			"%s: argument %d must be chan %s", name, i+1, elem))
	}

	if !ch.canSend {
		return nil, NewRuntimeError(node, fmt.Sprintf(
			"%s: argument %d must be a send-capable channel", name, i+1))
	}

	if ch.closed {
		return nil, NewRuntimeError(node, fmt.Sprintf(
			"%s: send on closed channel", name))
	}

	return ch, nil
}

func ToFloat32(i *Interpreter, v Value) (float32, bool) {
	fv := i.PromoteValueToType(v, i.TypeEnv["float"].TypeInfo)
	f, ok := fv.(FloatValue)
	if !ok {
		return 0, false
	}
	return float32(f.V), true
}

func WrapFloat1(name string, fn func(float64) float64) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			f, err := ArgFloat(node, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			return FloatValue{V: fn(f)}, nil
		},
	}
}

func WrapFloat2(name string, fn func(float64, float64) float64) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			f1, err := ArgFloat(node, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			f2, err := ArgFloat(node, args, 1, name)
			if err != nil {
				return NilValue{}, err
			}

			return FloatValue{V: fn(f1, f2)}, nil
		},
	}
}

func WrapString1(name string, fn func(string) string) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			s, err := ArgString(node, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			return StringValue{V: fn(s)}, nil
		},
	}
}

func WrapString1RSlice(name string, fn func(string) []string) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			s, err := ArgString(node, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			arr := []Value{}

			for _, s := range fn(s) {
				arr = append(arr, StringValue{V: s})
			}

			return ArrayValue{
				Elements: arr,
				ElemType: i.TypeEnv["string"].TypeInfo,
			}, nil
		},
	}
}

func WrapString2(name string, fn func(string, string) string) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			s, err := ArgString(node, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			s2, err := ArgString(node, args, 1, name)
			if err != nil {
				return NilValue{}, err
			}

			return StringValue{V: fn(s, s2)}, nil
		},
	}
}

func WrapString2IntRSlice(name string, fn func(string, string, int) []string) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			s, err := ArgString(node, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			s2, err := ArgString(node, args, 1, name)
			if err != nil {
				return NilValue{}, err
			}

			n, err := ArgInt(node, args, 2, name)
			if err != nil {
				return NilValue{}, err
			}

			arr := []Value{}

			for _, s := range fn(s, s2, n) {
				arr = append(arr, StringValue{V: s})
			}

			return ArrayValue{
				Elements: arr,
				ElemType: i.TypeEnv["string"].TypeInfo,
			}, nil
		},
	}
}

func WrapString2RSlice(name string, fn func(string, string) []string) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			s, err := ArgString(node, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			s2, err := ArgString(node, args, 1, name)
			if err != nil {
				return NilValue{}, err
			}

			arr := []Value{}

			for _, s := range fn(s, s2) {
				arr = append(arr, StringValue{V: s})
			}

			return ArrayValue{
				Elements: arr,
				ElemType: i.TypeEnv["string"].TypeInfo,
			}, nil
		},
	}
}

func WrapString2RInt(name string, fn func(string, string) int) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			s, err := ArgString(node, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			s2, err := ArgString(node, args, 1, name)
			if err != nil {
				return NilValue{}, err
			}

			return IntValue{V: fn(s, s2)}, nil
		},
	}
}

func WrapSliceStringRString(name string, fn func([]string, string) string) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 2,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			sliceVal, err := ArgArray(node, args, 0, name, "string")
			if err != nil {
				return NilValue{}, err
			}

			slice := []string{}

			for _, s := range sliceVal.Elements {
				if _, ok := s.(StringValue); !ok {
					return NilValue{}, NewRuntimeError(node, fmt.Sprintf("%s: first argument must be a []string", name))
				}

				slice = append(slice, s.(StringValue).V)
			}

			s, err := ArgString(node, args, 1, name)
			if err != nil {
				return NilValue{}, err
			}

			return StringValue{V: fn(slice, s)}, nil
		},
	}
}
