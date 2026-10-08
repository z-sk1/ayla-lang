//go:build rl

package rl

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/parser"
)

func ArgColor(node parser.Node, TypeEnv map[string]interpreter.TypeValue, args []interpreter.Value, i int, name string) (rl.Color, error) {
	colTI := TypeEnv["Color"].TypeInfo

	sv, err := interpreter.ArgStruct(node, args, i, name, "rl.Color")
	if err != nil {
		return rl.Color{}, err
	}

	if !interpreter.TypesAssignable(sv.TypeName, colTI) {
		return rl.Color{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Color", name, i+1))
	}

	return ColorFromValue(sv)
}

func ArgVector2(node parser.Node, i *interpreter.Interpreter, TypeEnv map[string]interpreter.TypeValue, args []interpreter.Value, idx int, name string) (rl.Vector2, error) {
	vecTI := TypeEnv["Vector2"].TypeInfo

	sv, err := interpreter.ArgStruct(node, args, idx, name, "rl.Vector2")
	if err != nil {
		return rl.Vector2{}, err
	}

	if !interpreter.TypesAssignable(i.TypeInfoFromValue(sv), vecTI) {
		return rl.Vector2{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Vector2", name, idx+1))
	}

	x, _ := interpreter.ToFloat32(i, interpreter.UnwrapUntyped(sv.Fields["X"]))
	y, _ := interpreter.ToFloat32(i, interpreter.UnwrapUntyped(sv.Fields["Y"]))

	return rl.Vector2{
		X: float32(x),
		Y: float32(y),
	}, nil
}

func ArgSound(node parser.Node, i *interpreter.Interpreter, TypeEnv map[string]interpreter.TypeValue, args []interpreter.Value, idx int, name string) (*rl.Sound, error) {
	soundTI := TypeEnv["Sound"].TypeInfo

	sv, err := interpreter.ArgStruct(node, args, idx, name, "rl.Sound")
	if err != nil {
		return &rl.Sound{}, err
	}

	if !interpreter.TypesAssignable(i.TypeInfoFromValue(sv), soundTI) {
		return &rl.Sound{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Sound", name, idx+1))
	}

	sound, ok := sv.Native.(*rl.Sound)
	if !ok {
		return &rl.Sound{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Sound", name, idx+1))
	}

	return sound, nil
}

func ArgMusic(node parser.Node, i *interpreter.Interpreter, TypeEnv map[string]interpreter.TypeValue, args []interpreter.Value, idx int, name string) (*rl.Music, error) {
	musTI := TypeEnv["Music"].TypeInfo

	sv, err := interpreter.ArgStruct(node, args, idx, name, "rl.Music")
	if err != nil {
		return &rl.Music{}, err
	}

	if !interpreter.TypesAssignable(i.TypeInfoFromValue(sv), musTI) {
		return &rl.Music{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Music", name, idx+1))
	}

	mus, ok := sv.Native.(*rl.Music)
	if !ok {
		return &rl.Music{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Music", name, idx+1))
	}

	return mus, nil
}

func ArgFont(node parser.Node, i *interpreter.Interpreter, TypeEnv map[string]interpreter.TypeValue, args []interpreter.Value, idx int, name string) (rl.Font, error) {
	fontTI := TypeEnv["Font"].TypeInfo

	sv, err := interpreter.ArgStruct(node, args, idx, name, "rl.Font")
	if err != nil {
		return rl.Font{}, err
	}

	if !interpreter.TypesAssignable(i.TypeInfoFromValue(sv), fontTI) {
		return rl.Font{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Font", name, idx+1))
	}

	font, ok := sv.Native.(rl.Font)
	if !ok {
		return rl.Font{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Font", name, idx+1))
	}

	return font, nil
}

func ArgRectangle(node parser.Node, i *interpreter.Interpreter, TypeEnv map[string]interpreter.TypeValue, args []interpreter.Value, idx int, name string) (rl.Rectangle, error) {
	rectTI := TypeEnv["Rectangle"].TypeInfo

	v := interpreter.UnwrapFully(args[idx])

	rectVal, ok := v.(*interpreter.StructValue)
	if !ok {
		return rl.Rectangle{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Rectangle", name, idx+1))
	}

	if !interpreter.TypesAssignable(i.TypeInfoFromValue(rectVal), rectTI) {
		return rl.Rectangle{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Rectangle", name, idx+1))
	}

	rect := rl.Rectangle{
		X:      float32(rectVal.Fields["X"].(interpreter.FloatValue).V),
		Y:      float32(rectVal.Fields["Y"].(interpreter.FloatValue).V),
		Width:  float32(rectVal.Fields["Width"].(interpreter.FloatValue).V),
		Height: float32(rectVal.Fields["Height"].(interpreter.FloatValue).V),
	}

	return rect, nil
}

func ArgTexture2D(node parser.Node, i *interpreter.Interpreter, TypeEnv map[string]interpreter.TypeValue, args []interpreter.Value, idx int, name string) (rl.Texture2D, error) {
	texTI := TypeEnv["Texture2D"].TypeInfo

	v := interpreter.UnwrapFully(args[idx])

	texVal, ok := v.(*interpreter.StructValue)
	if !ok {
		return rl.Texture2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Texture2D", name, idx+1))
	}

	if !interpreter.TypesAssignable(i.TypeInfoFromValue(texVal), texTI) {
		return rl.Texture2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Texture2D", name, idx+1))
	}

	tex, ok := texVal.Native.(rl.Texture2D)
	if !ok {
		return rl.Texture2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Texture2D", name, idx+1))
	}

	return tex, nil
}

func ArgRenderTexture2D(node parser.Node, i *interpreter.Interpreter, TypeEnv map[string]interpreter.TypeValue, args []interpreter.Value, idx int, name string) (rl.RenderTexture2D, error) {
	texTI := TypeEnv["RenderTexture2D"].TypeInfo

	v := interpreter.UnwrapFully(args[idx])

	texVal, ok := v.(*interpreter.StructValue)
	if !ok {
		return rl.RenderTexture2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.RenderTexture2D", name, idx+1))
	}

	if !interpreter.TypesAssignable(i.TypeInfoFromValue(texVal), texTI) {
		return rl.RenderTexture2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.RenderTexture2D", name, idx+1))
	}

	tex, ok := texVal.Native.(rl.RenderTexture2D)
	if !ok {
		return rl.RenderTexture2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.RenderTexture2D", name, idx+1))
	}

	return tex, nil
}

func ArgCamera2D(node *parser.FuncCall, i *interpreter.Interpreter, typeEnv map[string]interpreter.TypeValue, args []interpreter.Value, idx int, name string) (rl.Camera2D, error) {
	camTI := typeEnv["Camera2D"].TypeInfo
	vecTI := typeEnv["Vector2"].TypeInfo

	sv, err := interpreter.ArgStruct(node, args, idx, name, "rl.Camera2D")

	if err != nil {
		return rl.Camera2D{}, err
	}

	if !interpreter.TypesAssignable(i.TypeInfoFromValue(sv), camTI) {
		return rl.Camera2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	if _, ok := sv.Fields["Offset"].(*interpreter.StructValue); !ok {
		return rl.Camera2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	if !interpreter.TypesAssignable(sv.Fields["Offset"].(*interpreter.StructValue).TypeName, vecTI) {
		return rl.Camera2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	offsetVal := sv.Fields["Offset"].(*interpreter.StructValue)

	x, ok := interpreter.ToFloat32(i, offsetVal.Fields["X"])
	if !ok {
		return rl.Camera2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	y, ok := interpreter.ToFloat32(i, offsetVal.Fields["Y"])
	if !ok {
		return rl.Camera2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	offset := rl.Vector2{
		X: x,
		Y: y,
	}

	if _, ok := sv.Fields["Target"].(*interpreter.StructValue); !ok {
		return rl.Camera2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	if !interpreter.TypesAssignable(sv.Fields["Target"].(*interpreter.StructValue).TypeName, vecTI) {
		return rl.Camera2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	targetVal := sv.Fields["Target"].(*interpreter.StructValue)

	x, ok = interpreter.ToFloat32(i, targetVal.Fields["X"])
	if !ok {
		return rl.Camera2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	y, ok = interpreter.ToFloat32(i, targetVal.Fields["Y"])
	if !ok {
		return rl.Camera2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	target := rl.Vector2{
		X: x,
		Y: y,
	}

	rotVal := i.PromoteValueToType(sv.Fields["Rotation"], i.TypeEnv["float"].TypeInfo)
	if _, ok := sv.Fields["Rotation"].(interpreter.FloatValue); !ok {
		return rl.Camera2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	rot, ok := interpreter.ToFloat32(i, rotVal)
	if !ok {
		return rl.Camera2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	zoomVal := i.PromoteValueToType(sv.Fields["Zoom"], i.TypeEnv["float"].TypeInfo)

	if _, ok := sv.Fields["Zoom"].(interpreter.FloatValue); !ok {
		return rl.Camera2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	zoom, ok := interpreter.ToFloat32(i, zoomVal)
	if !ok {
		return rl.Camera2D{}, interpreter.NewRuntimeError(node, fmt.Sprintf("%s: argument %d must be rl.Camera2D", name, idx+1))
	}

	cam := rl.Camera2D{
		Offset:   offset,
		Target:   target,
		Rotation: rot,
		Zoom:     zoom,
	}

	return cam, nil
}

func ColorFromValue(v interpreter.Value) (rl.Color, error) {
	colVal := v.(*interpreter.StructValue)

	r := colVal.Fields["R"].(interpreter.IntValue).V
	g := colVal.Fields["G"].(interpreter.IntValue).V
	b := colVal.Fields["B"].(interpreter.IntValue).V
	a := colVal.Fields["A"].(interpreter.IntValue).V

	return rl.Color{
		R: uint8(r),
		G: uint8(g),
		B: uint8(b),
		A: uint8(a),
	}, nil
}

func UnwrapVector2(i *interpreter.Interpreter, v interpreter.Value) (float32, float32) {
	x, ok := interpreter.ToFloat32(i, v.(*interpreter.StructValue).Fields["X"])
	if !ok {
		return 0, 0
	}

	y, ok := interpreter.ToFloat32(i, v.(*interpreter.StructValue).Fields["Y"])
	if !ok {
		return 0, 0
	}

	return float32(x), float32(y)
}

func MakeVector2(v rl.Vector2, TypeEnv map[string]interpreter.TypeValue) interpreter.Value {
	return &interpreter.StructValue{
		TypeName: TypeEnv["Vector2"].TypeInfo,
		Fields: map[string]interpreter.Value{
			"X": interpreter.FloatValue{V: float64(v.X)},
			"Y": interpreter.FloatValue{V: float64(v.Y)},
		},
	}
}

func WrapVector2D1(name string, TypeEnv map[string]interpreter.TypeValue, fn func(rl.Vector2) rl.Vector2) *interpreter.BuiltinFunc {
	return &interpreter.BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := ArgVector2(node, i, TypeEnv, args, 0, name)
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return MakeVector2(fn(v), TypeEnv), nil
		},
	}
}

func WrapVector2D1RFloat(name string, TypeEnv map[string]interpreter.TypeValue, fn func(rl.Vector2) float32) *interpreter.BuiltinFunc {
	return &interpreter.BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := ArgVector2(node, i, TypeEnv, args, 0, name)
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.FloatValue{V: float64(fn(v))}, nil
		},
	}
}

func WrapVector2D2(name string, TypeEnv map[string]interpreter.TypeValue, fn func(rl.Vector2, rl.Vector2) rl.Vector2) *interpreter.BuiltinFunc {
	return &interpreter.BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := ArgVector2(node, i, TypeEnv, args, 0, name)
			if err != nil {
				return interpreter.NilValue{}, err
			}

			v2, err := ArgVector2(node, i, TypeEnv, args, 1, name)
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return MakeVector2(fn(v, v2), TypeEnv), nil
		},
	}
}

func WrapVector2D2RFloat(name string, TypeEnv map[string]interpreter.TypeValue, fn func(rl.Vector2, rl.Vector2) float32) *interpreter.BuiltinFunc {
	return &interpreter.BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := ArgVector2(node, i, TypeEnv, args, 0, name)
			if err != nil {
				return interpreter.NilValue{}, err
			}

			v2, err := ArgVector2(node, i, TypeEnv, args, 1, name)
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.FloatValue{V: float64(fn(v, v2))}, nil
		},
	}
}

func WrapVector2DFloat(name string, TypeEnv map[string]interpreter.TypeValue, fn func(rl.Vector2, float32) rl.Vector2) *interpreter.BuiltinFunc {
	return &interpreter.BuiltinFunc{
		Name:  name,
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := ArgVector2(node, i, TypeEnv, args, 0, name)
			if err != nil {
				return interpreter.NilValue{}, err
			}

			f, err := interpreter.ArgFloat(node, args, 1, name)
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return MakeVector2(fn(v, float32(f)), TypeEnv), nil
		},
	}
}
