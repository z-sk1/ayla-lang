package interpreter

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"

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

func ArgArray(node parser.Node, args []Value, i int, name string) (ArrayValue, error) {
	v := UnwrapFully(args[i])
	av, ok := v.(ArrayValue)
	if !ok {
		return ArrayValue{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be an array or slice", name, i+1))
	}
	return av, nil
}

func ArgColor(node parser.Node, TypeEnv map[string]TypeValue, args []Value, i int, name string) (rl.Color, error) {
	colTI := TypeEnv["Color"].TypeInfo

	sv, err := ArgStruct(node, args, i, name, "rl.Color")
	if err != nil {
		return rl.Color{}, err
	}

	if !typesAssignable(sv.TypeName, colTI) {
		return rl.Color{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Color", name, i+1))
	}

	return ColorFromValue(sv)
}

func ArgVector2(node parser.Node, i *Interpreter, TypeEnv map[string]TypeValue, args []Value, idx int, name string) (rl.Vector2, error) {
	vecTI := TypeEnv["Vector2"].TypeInfo

	v := UnwrapFully(args[idx])

	vecVal, ok := v.(*StructValue)
	if !ok {
		return rl.Vector2{}, NewRuntimeError(node, name+": argument must be rl.Vector2")
	}

	if !typesAssignable(i.TypeInfoFromValue(vecVal), vecTI) {
		return rl.Vector2{}, NewRuntimeError(node, name+": argument must be rl.Vector2")
	}

	x, _ := toFloat(UnwrapUntyped(vecVal.Fields["X"]))
	y, _ := toFloat(UnwrapUntyped(vecVal.Fields["Y"]))

	return rl.Vector2{
		X: float32(x),
		Y: float32(y),
	}, nil
}

func ArgSound(node parser.Node, i *Interpreter, TypeEnv map[string]TypeValue, args []Value, idx int, name string) (rl.Sound, error) {
	soundTI := TypeEnv["Sound"].TypeInfo

	v := UnwrapFully(args[idx])

	soundVal, ok := v.(*StructValue)
	if !ok {
		return rl.Sound{}, NewRuntimeError(node, fmt.Sprintf("%s: argument must be rl.Sound", name))
	}

	if !typesAssignable(i.TypeInfoFromValue(soundVal), soundTI) {
		return rl.Sound{}, NewRuntimeError(node, fmt.Sprintf("%s: argument must be rl.Sound", name))
	}

	sound, ok := soundVal.Native.(rl.Sound)
	if !ok {
		return rl.Sound{}, NewRuntimeError(node, fmt.Sprintf("%s: argument must be rl.Sound", name))
	}

	return sound, nil
}

func ArgMusic(node parser.Node, i *Interpreter, TypeEnv map[string]TypeValue, args []Value, idx int, name string) (*rl.Music, error) {
	musTI := TypeEnv["Music"].TypeInfo

	v := UnwrapFully(args[idx])

	musVal, ok := v.(*StructValue)
	if !ok {
		return &rl.Music{}, NewRuntimeError(node, fmt.Sprintf("%s: argument must be rl.Music", name))
	}

	if !typesAssignable(i.TypeInfoFromValue(musVal), musTI) {
		return &rl.Music{}, NewRuntimeError(node, fmt.Sprintf("%s: argument must be rl.Music", name))
	}

	mus, ok := musVal.Native.(*rl.Music)
	if !ok {
		return &rl.Music{}, NewRuntimeError(node, fmt.Sprintf("%s: argument must be rl.Music", name))
	}

	return mus, nil
}

func ColorFromValue(v Value) (rl.Color, error) {
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

func UnwrapVector2(v Value) (float32, float32) {
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
