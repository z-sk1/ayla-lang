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

func ArgArray(node parser.Node, args []Value, i int, name string, slice string) (ArrayValue, error) {
	v := UnwrapFully(args[i])
	av, ok := v.(ArrayValue)
	if !ok {
		return ArrayValue{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be a []%s", name, i+1, slice))
	}
	return av, nil
}

func ArgColor(node parser.Node, TypeEnv map[string]TypeValue, args []Value, i int, name string) (rl.Color, error) {
	colTI := TypeEnv["Color"].TypeInfo

	sv, err := ArgStruct(node, args, i, name, "rl.Color")
	if err != nil {
		return rl.Color{}, err
	}

	if !TypesAssignable(sv.TypeName, colTI) {
		return rl.Color{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Color", name, i+1))
	}

	return ColorFromValue(sv)
}

func ArgVector2(node parser.Node, i *Interpreter, TypeEnv map[string]TypeValue, args []Value, idx int, name string) (rl.Vector2, error) {
	vecTI := TypeEnv["Vector2"].TypeInfo

	sv, err := ArgStruct(node, args, idx, name, "rl.Vector2")
	if err != nil {
		return rl.Vector2{}, err
	}

	if !TypesAssignable(i.TypeInfoFromValue(sv), vecTI) {
		return rl.Vector2{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Vector2", name, idx+1))
	}

	x, _ := toFloat(UnwrapUntyped(sv.Fields["X"]))
	y, _ := toFloat(UnwrapUntyped(sv.Fields["Y"]))

	return rl.Vector2{
		X: float32(x),
		Y: float32(y),
	}, nil
}

func ArgSound(node parser.Node, i *Interpreter, TypeEnv map[string]TypeValue, args []Value, idx int, name string) (*rl.Sound, error) {
	soundTI := TypeEnv["Sound"].TypeInfo

	sv, err := ArgStruct(node, args, idx, name, "rl.Sound")
	if err != nil {
		return &rl.Sound{}, err
	}

	if !TypesAssignable(i.TypeInfoFromValue(sv), soundTI) {
		return &rl.Sound{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Sound", name, idx+1))
	}

	sound, ok := sv.Native.(*rl.Sound)
	if !ok {
		return &rl.Sound{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Sound", name, idx+1))
	}

	return sound, nil
}

func ArgMusic(node parser.Node, i *Interpreter, TypeEnv map[string]TypeValue, args []Value, idx int, name string) (*rl.Music, error) {
	musTI := TypeEnv["Music"].TypeInfo

	sv, err := ArgStruct(node, args, idx, name, "rl.Music")
	if err != nil {
		return &rl.Music{}, err
	}

	if !TypesAssignable(i.TypeInfoFromValue(sv), musTI) {
		return &rl.Music{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Music", name, idx+1))
	}

	mus, ok := sv.Native.(*rl.Music)
	if !ok {
		return &rl.Music{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Music", name, idx+1))
	}

	return mus, nil
}

func ArgRectangle(node parser.Node, i *Interpreter, TypeEnv map[string]TypeValue, args []Value, idx int, name string) (rl.Rectangle, error) {
	rectTI := TypeEnv["Rectangle"].TypeInfo

	v := UnwrapFully(args[idx])

	rectVal, ok := v.(*StructValue)
	if !ok {
		return rl.Rectangle{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Rectangle", name, idx+1))
	}

	if !TypesAssignable(i.TypeInfoFromValue(rectVal), rectTI) {
		return rl.Rectangle{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Rectangle", name, idx+1))
	}

	rect := rl.Rectangle{
		X:      float32(rectVal.Fields["X"].(FloatValue).V),
		Y:      float32(rectVal.Fields["Y"].(FloatValue).V),
		Width:  float32(rectVal.Fields["Width"].(FloatValue).V),
		Height: float32(rectVal.Fields["Height"].(FloatValue).V),
	}

	return rect, nil
}

func ArgTexture2D(node parser.Node, i *Interpreter, TypeEnv map[string]TypeValue, args []Value, idx int, name string) (rl.Texture2D, error) {
	texTI := TypeEnv["Texture2D"].TypeInfo

	v := UnwrapFully(args[idx])

	texVal, ok := v.(*StructValue)
	if !ok {
		return rl.Texture2D{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Texture2D", name, idx+1))
	}

	if !TypesAssignable(i.TypeInfoFromValue(texVal), texTI) {
		return rl.Texture2D{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Texture2D", name, idx+1))
	}

	mus, ok := texVal.Native.(rl.Texture2D)
	if !ok {
		return rl.Texture2D{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Texture2D", name, idx+1))
	}

	return mus, nil
}

func ArgCamera2D(node *parser.FuncCall, i *Interpreter, typeEnv map[string]TypeValue, args []Value, idx int, name string) (rl.Camera2D, error) {
	camTI := typeEnv["Camera2D"].TypeInfo
	vecTI := typeEnv["Vector2"].TypeInfo

	sv, err := ArgStruct(node, args, idx, name, "rl.Camera2D")

	if err != nil {
		return rl.Camera2D{}, err
	}

	if !TypesAssignable(i.TypeInfoFromValue(sv), camTI) {
		return rl.Camera2D{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	if _, ok := sv.Fields["Offset"].(*StructValue); !ok {
		return rl.Camera2D{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	if !TypesAssignable(sv.Fields["Offset"].(*StructValue).TypeName, vecTI) {
		return rl.Camera2D{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	offsetVal := sv.Fields["Offset"].(*StructValue)

	offset := rl.Vector2{
		X: float32(offsetVal.Fields["X"].(FloatValue).V),
		Y: float32(offsetVal.Fields["Y"].(FloatValue).V),
	}

	if _, ok := sv.Fields["Target"].(*StructValue); !ok {
		return rl.Camera2D{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	if !TypesAssignable(sv.Fields["Target"].(*StructValue).TypeName, vecTI) {
		return rl.Camera2D{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	targetVal := sv.Fields["Target"].(*StructValue)

	target := rl.Vector2{
		X: float32(targetVal.Fields["X"].(FloatValue).V),
		Y: float32(targetVal.Fields["Y"].(FloatValue).V),
	}

	if _, ok := sv.Fields["Rotation"].(FloatValue); !ok {
		return rl.Camera2D{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	rotVal := sv.Fields["Rotation"].(FloatValue)

	rot := float32(rotVal.V)

	if _, ok := sv.Fields["Zoom"].(FloatValue); !ok {
		return rl.Camera2D{}, NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	zoomVal := sv.Fields["Zoom"].(FloatValue)

	zoom := float32(zoomVal.V)

	cam := rl.Camera2D{
		Offset:   offset,
		Target:   target,
		Rotation: rot,
		Zoom:     zoom,
	}

	return cam, nil
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

func MakeVector2(v rl.Vector2, TypeEnv map[string]TypeValue) Value {
	return &StructValue{
		TypeName: TypeEnv["Vector2"].TypeInfo,
		Fields: map[string]Value{
			"X": FloatValue{V: float64(v.X)},
			"Y": FloatValue{V: float64(v.Y)},
		},
	}
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

func WrapVector2D1(name string, TypeEnv map[string]TypeValue, fn func(rl.Vector2) rl.Vector2) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := ArgVector2(node, i, TypeEnv, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			return MakeVector2(fn(v), TypeEnv), nil
		},
	}
}

func WrapVector2D1RFloat(name string, TypeEnv map[string]TypeValue, fn func(rl.Vector2) float32) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := ArgVector2(node, i, TypeEnv, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			return FloatValue{V: float64(fn(v))}, nil
		},
	}
}

func WrapVector2D2(name string, TypeEnv map[string]TypeValue, fn func(rl.Vector2, rl.Vector2) rl.Vector2) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := ArgVector2(node, i, TypeEnv, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			v2, err := ArgVector2(node, i, TypeEnv, args, 1, name)
			if err != nil {
				return NilValue{}, err
			}

			return MakeVector2(fn(v, v2), TypeEnv), nil
		},
	}
}

func WrapVector2D2RFloat(name string, TypeEnv map[string]TypeValue, fn func(rl.Vector2, rl.Vector2) float32) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := ArgVector2(node, i, TypeEnv, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			v2, err := ArgVector2(node, i, TypeEnv, args, 1, name)
			if err != nil {
				return NilValue{}, err
			}

			return FloatValue{V: float64(fn(v, v2))}, nil
		},
	}
}

func WrapVector2DFloat(name string, TypeEnv map[string]TypeValue, fn func(rl.Vector2, float32) rl.Vector2) *BuiltinFunc {
	return &BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error) {
			v, err := ArgVector2(node, i, TypeEnv, args, 0, name)
			if err != nil {
				return NilValue{}, err
			}

			f, err := ArgFloat(node, args, 1, name)
			if err != nil {
				return NilValue{}, err
			}

			return MakeVector2(fn(v, float32(f)), TypeEnv), nil
		},
	}
}
