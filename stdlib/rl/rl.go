package rl

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/registry"
)

func init() {
	registry.Register("rl", LoadRLModule)
}

func LoadRLModule(i *interpreter.Interpreter) (interpreter.ModuleValue, error) {
	env := interpreter.NewEnvironment(i.Env)
	TypeEnv := make(map[string]interpreter.TypeValue)

	min := 0.0
	max := 255.0

	uint8Type := &interpreter.TypeInfo{
		Name:         "int[0..255]",
		Kind:         interpreter.TypeInt,
		Min:          &min,
		Max:          &max,
		IsComparable: true,
	}

	TypeEnv["Color"] = interpreter.TypeValue{
		TypeInfo: &interpreter.TypeInfo{
			Name: "Color",
			Kind: interpreter.TypeStruct,
			Fields: map[string]*interpreter.TypeInfo{
				"R": uint8Type,
				"G": uint8Type,
				"B": uint8Type,
				"A": uint8Type,
			},
		},
	}

	TypeEnv["Vector2"] = interpreter.TypeValue{
		TypeInfo: &interpreter.TypeInfo{
			Name: "Vector2",
			Kind: interpreter.TypeStruct,
			Fields: map[string]*interpreter.TypeInfo{
				"X": i.TypeEnv["float"].TypeInfo,
				"Y": i.TypeEnv["float"].TypeInfo,
			},
		},
	}

	TypeEnv["Sound"] = interpreter.TypeValue{
		TypeInfo: &interpreter.TypeInfo{
			Name:   "Sound",
			Kind:   interpreter.TypeStruct,
			Fields: nil,
			Opaque: true,
		},
	}

	TypeEnv["Music"] = interpreter.TypeValue{
		TypeInfo: &interpreter.TypeInfo{
			Name:   "Music",
			Kind:   interpreter.TypeStruct,
			Fields: nil,
			Opaque: true,
		},
	}

	env.Define("SetWindowFlags", &interpreter.BuiltinFunc{
		Name:  "SetWindowFlags",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			flag, err := interpreter.ArgInt(node, args, 0, "rl.SetWindowFlags")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetConfigFlags(uint32(flag))
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("InitWindow", &interpreter.BuiltinFunc{
		Name:  "Init",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			w, err := interpreter.ArgInt(node, args, 0, "rl.InitWindow")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			h, err := interpreter.ArgInt(node, args, 1, "rl.InitWindow")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			title, err := interpreter.ArgString(node, args, 2, "rl.InitWindow")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.InitWindow(int32(w), int32(h), title)
			rl.SetTargetFPS(60)

			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("CloseWindow", &interpreter.BuiltinFunc{
		Name:  "CloseWindow",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rl.CloseWindow()
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("SetTargetFPS", &interpreter.BuiltinFunc{
		Name:  "SetTargetFPS",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			fps, err := interpreter.ArgInt(node, args, 0, "rl.SetTargetFPS")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetTargetFPS(int32(fps))
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("ShouldClose", &interpreter.BuiltinFunc{
		Name:  "ShouldClose",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return interpreter.BoolValue{V: rl.WindowShouldClose()}, nil
		},
	}, false)

	env.Define("SetExitKey", &interpreter.BuiltinFunc{
		Name:  "SetExitKey",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			key, err := interpreter.ArgInt(node, args, 0, "rl.SetExitKey")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetExitKey(int32(key))
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("Clear", &interpreter.BuiltinFunc{
		Name:  "Clear",
		Arity: -1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {

			switch len(args) {
			case 0:
				rl.BeginDrawing()
				rl.ClearBackground(rl.Black)

				return interpreter.NilValue{}, nil
			case 1:
				col, err := interpreter.ArgColor(node, TypeEnv, args, 0, "rl.Clear")
				if err != nil {
					return interpreter.NilValue{}, err
				}

				rl.BeginDrawing()
				rl.ClearBackground(col)
				return interpreter.NilValue{}, nil
			}

			return interpreter.NilValue{}, interpreter.ExpectArgsRange(node, args, 0, 1, "rl.Clear")
		},
	}, false)

	env.Define("Present", &interpreter.BuiltinFunc{
		Name:  "Present",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rl.EndDrawing()
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("SetWindowTitle", &interpreter.BuiltinFunc{
		Name:  "SetWindowTitle",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			title, err := interpreter.ArgString(node, args, 0, "rl.SetWindowTitle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetWindowTitle(title)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("SetWindowPos", &interpreter.BuiltinFunc{
		Name:  "SetWindowPos",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			x, err := interpreter.ArgInt(node, args, 0, "rl.SetWindowPos")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			y, err := interpreter.ArgInt(node, args, 1, "rl.SetWindowPos")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetWindowPosition(x, y)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("GetWindowPos", &interpreter.BuiltinFunc{
		Name:  "GetWindowPos",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return &interpreter.StructValue{
				TypeName: TypeEnv["Vector2"].TypeInfo,
				Fields: map[string]interpreter.Value{
					"X": interpreter.IntValue{V: int(rl.GetWindowPosition().X)},
					"Y": interpreter.IntValue{V: int(rl.GetWindowPosition().Y)},
				},
			}, nil
		},
	}, false)

	env.Define("SetWindowSize", &interpreter.BuiltinFunc{
		Name:  "SetWindowSize",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			w, err := interpreter.ArgInt(node, args, 0, "rl.SetWindowSize")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			h, err := interpreter.ArgInt(node, args, 1, "rl.SetWindowSize")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetWindowSize(w, h)
			return interpreter.NilValue{}, err
		},
	}, false)

	env.Define("SetWindowMinSize", &interpreter.BuiltinFunc{
		Name:  "SetWindowMinSize",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			w, err := interpreter.ArgInt(node, args, 0, "rl.SetWindowMinSize")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			h, err := interpreter.ArgInt(node, args, 1, "rl.SetWindowMinSize")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetWindowMinSize(w, h)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("SetWindowMaxSize", &interpreter.BuiltinFunc{
		Name:  "SetWindowMaxSize",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			w, err := interpreter.ArgInt(node, args, 0, "rl.SetWindowMaxSize")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			h, err := interpreter.ArgInt(node, args, 1, "rl.SetWindowMaxSize")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetWindowMaxSize(w, h)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("GetWindowSize", &interpreter.BuiltinFunc{
		Name:  "GetWindowSize",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return &interpreter.StructValue{
				TypeName: TypeEnv["Vector2"].TypeInfo,
				Fields: map[string]interpreter.Value{
					"X": interpreter.IntValue{V: rl.GetScreenWidth()},
					"Y": interpreter.IntValue{V: rl.GetScreenHeight()},
				},
			}, nil
		},
	}, false)

	env.Define("SetFullscreen", &interpreter.BuiltinFunc{
		Name:  "SetFullscreen",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			enabled, err := interpreter.ArgBool(node, args, 0, "rl.SetFullscreen")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			if rl.IsWindowFullscreen() != enabled {
				rl.ToggleFullscreen()
			}

			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("IsWindowFocused", &interpreter.BuiltinFunc{
		Name:  "IsWindowFocused",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return interpreter.BoolValue{V: rl.IsWindowFocused()}, nil
		},
	}, false)

	env.Define("IsWindowMinimized", &interpreter.BuiltinFunc{
		Name:  "IsWindowMinimized",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return interpreter.BoolValue{V: rl.IsWindowMinimized()}, nil
		},
	}, false)

	env.Define("IsWindowMaximized", &interpreter.BuiltinFunc{
		Name:  "IsWindowMaximized",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return interpreter.BoolValue{V: rl.IsWindowMaximized()}, nil
		},
	}, false)

	env.Define("MinimizeWindow", &interpreter.BuiltinFunc{
		Name:  "MinimizeWindow",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rl.MinimizeWindow()
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("MaximizeWindow", &interpreter.BuiltinFunc{
		Name:  "MaximizeWindow",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rl.MaximizeWindow()
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("RestoreWindow", &interpreter.BuiltinFunc{
		Name:  "RestoreWindow",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rl.RestoreWindow()
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("GetFPS", &interpreter.BuiltinFunc{
		Name:  "GetFPS",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return interpreter.IntValue{V: int(rl.GetFPS())}, nil
		},
	}, false)

	env.Define("GetFrameTime", &interpreter.BuiltinFunc{
		Name:  "GetFrameTime",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return interpreter.FloatValue{V: float64(rl.GetFrameTime())}, nil
		},
	}, false)

	env.Define("ShowCursor", &interpreter.BuiltinFunc{
		Name:  "ShowCursor",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rl.ShowCursor()
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("HideCursor", &interpreter.BuiltinFunc{
		Name:  "HideCursor",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rl.HideCursor()
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("LockCursor", &interpreter.BuiltinFunc{
		Name:  "LockCursor",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rl.DisableCursor()
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("UnlockCursor", &interpreter.BuiltinFunc{
		Name:  "UnlockCursor",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rl.EnableCursor()
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("NewColor", &interpreter.BuiltinFunc{
		Name:  "NewColor",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			r, err := interpreter.ArgInt(node, args, 0, "rl.NewColor")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			g, err := interpreter.ArgInt(node, args, 1, "rl.NewColor")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			b, err := interpreter.ArgInt(node, args, 2, "rl.NewColor")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			a, err := interpreter.ArgInt(node, args, 3, "rl.NewColor")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return &interpreter.StructValue{
				TypeName: TypeEnv["Color"].TypeInfo,
				Fields: map[string]interpreter.Value{
					"R": interpreter.IntValue{V: r},
					"G": interpreter.IntValue{V: g},
					"B": interpreter.IntValue{V: b},
					"A": interpreter.IntValue{V: a},
				},
			}, nil
		},
	}, false)

	env.Define("NewVector2", &interpreter.BuiltinFunc{
		Name:  "NewVector2",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			x, err := interpreter.ArgFloat(node, args, 0, "rl.NewVector2")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			y, err := interpreter.ArgFloat(node, args, 1, "rl.NewVector2")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			vec := &interpreter.StructValue{
				TypeName: TypeEnv["Vector2"].TypeInfo,
				Fields: map[string]interpreter.Value{
					"X": interpreter.FloatValue{V: x},
					"Y": interpreter.FloatValue{V: y},
				},
			}

			return vec, nil
		},
	}, false)

	env.Define("DrawRect", &interpreter.BuiltinFunc{
		Name:  "DrawRect",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawRect")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.DrawRect")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawRect")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawRectangleV(pv, sv, col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawRectLines", &interpreter.BuiltinFunc{
		Name:  "DrawRectLines",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawRectLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.DrawRectLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawRectLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawRectangleLines(int32(pv.X), int32(pv.Y), int32(sv.X), int32(sv.Y), col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawCircle", &interpreter.BuiltinFunc{
		Name:  "DrawCircle",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawCircle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rVal, err := interpreter.ArgFloat(node, args, 1, "rl.DrawCircle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawCircle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			r := float32(rVal)

			rl.DrawCircleV(pv, r, col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawCircleLines", &interpreter.BuiltinFunc{
		Name:  "DrawCircleLines",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawCircleLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rVal, err := interpreter.ArgFloat(node, args, 1, "rl.DrawCircleLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawCircleLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			r := float32(rVal)

			rl.DrawCircleLinesV(pv, r, col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawText", &interpreter.BuiltinFunc{
		Name:  "DrawText",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			txtVal, err := interpreter.ArgString(node, args, 0, "rl.DrawText")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.DrawRect")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			fontVal, err := interpreter.ArgInt(node, args, 2, "rl.DrawText")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 3, "rl.DrawText")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			txt := txtVal
			x := int32(pv.X)
			y := int32(pv.Y)
			font := int32(fontVal)

			rl.DrawText(txt, x, y, font, col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawLine", &interpreter.BuiltinFunc{
		Name:  "DrawLine",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawLine")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.DrawLine")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawLine")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawLineV(pv, sv, col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawTriangle", &interpreter.BuiltinFunc{
		Name:  "DrawTriangle",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pv1, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "DrawTriangle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			pv2, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "DrawTriangle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			pv3, err := interpreter.ArgVector2(node, i, TypeEnv, args, 2, "DrawTriangle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 3, "DrawTriangle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawTriangle(pv1, pv3, pv2, col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawTriangleLines", &interpreter.BuiltinFunc{
		Name:  "DrawTriangleLines",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pv1, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "DrawTriangle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			pv2, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "DrawTriangle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			pv3, err := interpreter.ArgVector2(node, i, TypeEnv, args, 2, "DrawTriangle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 3, "DrawTriangle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawTriangleLines(pv1, pv3, pv2, col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("IsKeyDown", &interpreter.BuiltinFunc{
		Name:  "IsKeyDown",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := interpreter.ArgInt(node, args, 0, "rl.KeyDown")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: rl.IsKeyDown(int32(v))}, nil
		},
	}, false)

	env.Define("GetKeyPressed", &interpreter.BuiltinFunc{
		Name:  "GetKeyPressed",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return interpreter.IntValue{V: int(rl.GetKeyPressed())}, nil
		},
	}, false)

	env.Define("IsKeyPressed", &interpreter.BuiltinFunc{
		Name:  "IsKeyPressed",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := interpreter.ArgInt(node, args, 0, "rl.KeyPressed")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: rl.IsKeyPressed(int32(v))}, nil
		},
	}, false)

	env.Define("IsKeyReleased", &interpreter.BuiltinFunc{
		Name:  "IsKeyReleased",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := interpreter.ArgInt(node, args, 0, "rl.KeyReleased")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: rl.IsKeyReleased(int32(v))}, nil
		},
	}, false)

	env.Define("IsKeyUp", &interpreter.BuiltinFunc{
		Name:  "IsKeyUp",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := interpreter.ArgInt(node, args, 0, "rl.KeyUp")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: rl.IsKeyUp(int32(v))}, nil
		},
	}, false)

	env.Define("IsMouseDown", &interpreter.BuiltinFunc{
		Name:  "IsMouseDown",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := interpreter.ArgInt(node, args, 0, "rl.MouseDown")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: rl.IsMouseButtonDown(rl.MouseButton((v)))}, nil
		},
	}, false)

	env.Define("IsMousePressed", &interpreter.BuiltinFunc{
		Name:  "IsMousePressed",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := interpreter.ArgInt(node, args, 0, "rl.MousePressed")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: rl.IsMouseButtonPressed(rl.MouseButton(int32(v)))}, nil
		},
	}, false)

	env.Define("IsMouseReleased", &interpreter.BuiltinFunc{
		Name:  "IsMouseReleased",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := interpreter.ArgInt(node, args, 0, "rl.MouseReleased")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: rl.IsMouseButtonReleased(rl.MouseButton(int32(v)))}, nil
		},
	}, false)

	env.Define("IsMouseUp", &interpreter.BuiltinFunc{
		Name:  "IsMouseUp",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := interpreter.ArgInt(node, args, 0, "rl.MouseUp")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: rl.IsMouseButtonUp(rl.MouseButton(int32(v)))}, nil
		},
	}, false)

	env.Define("SetMousePos", &interpreter.BuiltinFunc{
		Name:  "SetMousePos",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			x, err := interpreter.ArgInt(node, args, 0, "rl.SetMousePos")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			y, err := interpreter.ArgInt(node, args, 1, "rl.SetMousePos")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetMousePosition(x, y)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("GetMousePos", &interpreter.BuiltinFunc{
		Name:  "GetMousePos",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {

			return &interpreter.StructValue{
				TypeName: TypeEnv["Vector2"].TypeInfo,
				Fields: map[string]interpreter.Value{
					"X": interpreter.FloatValue{V: float64(rl.GetMousePosition().X)},
					"Y": interpreter.FloatValue{V: float64(rl.GetMousePosition().Y)},
				},
			}, nil
		},
	}, false)

	env.Define("GetMouseDelta", &interpreter.BuiltinFunc{
		Name:  "GetMouseDelta",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return &interpreter.StructValue{
				TypeName: TypeEnv["Vector2"].TypeInfo,
				Fields: map[string]interpreter.Value{
					"X": interpreter.FloatValue{V: float64(rl.GetMouseDelta().X)},
					"Y": interpreter.FloatValue{V: float64(rl.GetMouseDelta().Y)},
				},
			}, nil
		},
	}, false)

	env.Define("InitAudio", &interpreter.BuiltinFunc{
		Name:  "InitSound",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rl.InitAudioDevice()
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("CloseAudio", &interpreter.BuiltinFunc{
		Name:  "CloseSound",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rl.CloseAudioDevice()
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("LoadSound", &interpreter.BuiltinFunc{
		Name:  "LoadSound",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "rl.LoadSound")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sound := rl.LoadSound(path)

			return &interpreter.StructValue{
				TypeName: TypeEnv["Sound"].TypeInfo,
				Native:   sound,
			}, nil
		},
	}, false)

	env.Define("UnloadSound", &interpreter.BuiltinFunc{
		Name:  "UnloadSound",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			sound, err := interpreter.ArgSound(node, i, TypeEnv, args, 0, "rl.UnloadSound")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.UnloadSound(sound)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("PlaySound", &interpreter.BuiltinFunc{
		Name:  "PlaySound",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			sound, err := interpreter.ArgSound(node, i, TypeEnv, args, 0, "rl.PlaySound")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.PlaySound(sound)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("StopSound", &interpreter.BuiltinFunc{
		Name:  "StopSound",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			sound, err := interpreter.ArgSound(node, i, TypeEnv, args, 0, "rl.StopSound")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.StopSound(sound)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("IsSoundPlaying", &interpreter.BuiltinFunc{
		Name:  "IsSoundPlaying",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			sound, err := interpreter.ArgSound(node, i, TypeEnv, args, 0, "rl.IsSoundPlaying")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: rl.IsSoundPlaying(sound)}, nil
		},
	}, false)

	env.Define("PauseSound", &interpreter.BuiltinFunc{
		Name:  "PauseSound",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			sound, err := interpreter.ArgSound(node, i, TypeEnv, args, 0, "rl.PauseSound")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.PauseSound(sound)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("ResumeSound", &interpreter.BuiltinFunc{
		Name:  "ResumeSound",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			sound, err := interpreter.ArgSound(node, i, TypeEnv, args, 0, "rl.ResumeSound")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.ResumeSound(sound)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("SetSoundVolume", &interpreter.BuiltinFunc{
		Name:  "SetSoundVolume",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			sound, err := interpreter.ArgSound(node, i, TypeEnv, args, 0, "rl.SetSoundVolume")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			volume, err := interpreter.ArgFloat(node, args, 1, "rl.SetSoundVolume")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetSoundVolume(sound, float32(volume))
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("SetSoundPitch", &interpreter.BuiltinFunc{
		Name:  "SetSoundPitch",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			sound, err := interpreter.ArgSound(node, i, TypeEnv, args, 0, "rl.SetSoundPitch")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			pitch, err := interpreter.ArgFloat(node, args, 1, "rl.SetSoundPitch")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetSoundPitch(sound, float32(pitch))
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("LoadMusic", &interpreter.BuiltinFunc{
		Name:  "LoadMusic",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "rl.LoadMusic")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			mus := rl.LoadMusicStream(path)

			return &interpreter.StructValue{
				TypeName: TypeEnv["Music"].TypeInfo,
				Native:   &mus,
			}, nil
		},
	}, false)

	env.Define("UnloadMusic", &interpreter.BuiltinFunc{
		Name:  "UnloadMusic",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			mus, err := interpreter.ArgMusic(node, i, TypeEnv, args, 0, "rl.UnloadMusic")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.UnloadMusicStream(*mus)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("PlayMusic", &interpreter.BuiltinFunc{
		Name:  "PlayMusic",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			mus, err := interpreter.ArgMusic(node, i, TypeEnv, args, 0, "rl.PlayMusic")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.PlayMusicStream(*mus)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("UpdateMusic", &interpreter.BuiltinFunc{
		Name:  "UpdateMusic",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			mus, err := interpreter.ArgMusic(node, i, TypeEnv, args, 0, "rl.UpdateMusic")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.UpdateMusicStream(*mus)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("StopMusic", &interpreter.BuiltinFunc{
		Name:  "StopMusic",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			mus, err := interpreter.ArgMusic(node, i, TypeEnv, args, 0, "rl.StopMusic")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.StopMusicStream(*mus)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("IsMusicPlaying", &interpreter.BuiltinFunc{
		Name:  "IsMusicPlaying",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			mus, err := interpreter.ArgMusic(node, i, TypeEnv, args, 0, "rl.IsMusicPlaying")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: rl.IsMusicStreamPlaying(*mus)}, nil
		},
	}, false)

	env.Define("PauseMusic", &interpreter.BuiltinFunc{
		Name:  "PauseMusic",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			mus, err := interpreter.ArgMusic(node, i, TypeEnv, args, 0, "rl.PauseMusic")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.PauseMusicStream(*mus)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("ResumeMusic", &interpreter.BuiltinFunc{
		Name:  "ResumeMusic",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			mus, err := interpreter.ArgMusic(node, i, TypeEnv, args, 0, "rl.ResumeMusic")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.ResumeMusicStream(*mus)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("SetMusicVolume", &interpreter.BuiltinFunc{
		Name:  "SetMusicVolume",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			mus, err := interpreter.ArgMusic(node, i, TypeEnv, args, 0, "rl.SetMusicVolume")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			vol, err := interpreter.ArgFloat(node, args, 1, "rl.SetMusicVolume")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetMusicVolume(*mus, float32(vol))
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("SetMusicPitch", &interpreter.BuiltinFunc{
		Name:  "SetMusicPitch",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			mus, err := interpreter.ArgMusic(node, i, TypeEnv, args, 0, "rl.SetMusicPitch")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			pitch, err := interpreter.ArgFloat(node, args, 1, "rl.SetMusicPitch")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetMusicPitch(*mus, float32(pitch))
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("SetMusicLooping", &interpreter.BuiltinFunc{
		Name:  "SetMusicLooping",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			mus, err := interpreter.ArgMusic(node, i, TypeEnv, args, 0, "rl.SetMusicLooping")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			looping, err := interpreter.ArgBool(node, args, 1, "rl.SetMusicLooping")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			mus.Looping = looping
			return interpreter.NilValue{}, nil
		},
	}, false)

	// consts
	env.Define("MouseLeft", interpreter.IntValue{V: int(rl.MouseButtonLeft)}, true)
	env.Define("MouseRight", interpreter.IntValue{V: int(rl.MouseButtonRight)}, true)
	env.Define("MouseMiddle", interpreter.IntValue{V: int(rl.MouseMiddleButton)}, true)

	env.Define("KeyNil", interpreter.IntValue{V: rl.KeyNull}, true)
	env.Define("KeyA", interpreter.IntValue{V: rl.KeyA}, true)
	env.Define("KeyB", interpreter.IntValue{V: rl.KeyB}, true)
	env.Define("KeyC", interpreter.IntValue{V: rl.KeyC}, true)
	env.Define("KeyD", interpreter.IntValue{V: rl.KeyD}, true)
	env.Define("KeyE", interpreter.IntValue{V: rl.KeyE}, true)
	env.Define("KeyF", interpreter.IntValue{V: rl.KeyF}, true)
	env.Define("KeyG", interpreter.IntValue{V: rl.KeyG}, true)
	env.Define("KeyH", interpreter.IntValue{V: rl.KeyH}, true)
	env.Define("KeyI", interpreter.IntValue{V: rl.KeyI}, true)
	env.Define("KeyJ", interpreter.IntValue{V: rl.KeyJ}, true)
	env.Define("KeyK", interpreter.IntValue{V: rl.KeyK}, true)
	env.Define("KeyL", interpreter.IntValue{V: rl.KeyL}, true)
	env.Define("KeyM", interpreter.IntValue{V: rl.KeyM}, true)
	env.Define("KeyN", interpreter.IntValue{V: rl.KeyN}, true)
	env.Define("KeyO", interpreter.IntValue{V: rl.KeyO}, true)
	env.Define("KeyP", interpreter.IntValue{V: rl.KeyP}, true)
	env.Define("KeyQ", interpreter.IntValue{V: rl.KeyQ}, true)
	env.Define("KeyR", interpreter.IntValue{V: rl.KeyR}, true)
	env.Define("KeyS", interpreter.IntValue{V: rl.KeyS}, true)
	env.Define("KeyT", interpreter.IntValue{V: rl.KeyT}, true)
	env.Define("KeyU", interpreter.IntValue{V: rl.KeyU}, true)
	env.Define("KeyV", interpreter.IntValue{V: rl.KeyV}, true)
	env.Define("KeyW", interpreter.IntValue{V: rl.KeyW}, true)
	env.Define("KeyX", interpreter.IntValue{V: rl.KeyX}, true)
	env.Define("KeyY", interpreter.IntValue{V: rl.KeyY}, true)
	env.Define("KeyZ", interpreter.IntValue{V: rl.KeyZ}, true)
	env.Define("KeyUp", interpreter.IntValue{V: rl.KeyUp}, true)
	env.Define("KeyDown", interpreter.IntValue{V: rl.KeyDown}, true)
	env.Define("KeyLeft", interpreter.IntValue{V: rl.KeyLeft}, true)
	env.Define("KeyRight", interpreter.IntValue{V: rl.KeyRight}, true)
	env.Define("KeyEsc", interpreter.IntValue{V: rl.KeyEscape}, true)
	env.Define("KeyF1", interpreter.IntValue{V: rl.KeyF1}, true)
	env.Define("KeyF2", interpreter.IntValue{V: rl.KeyF2}, true)
	env.Define("KeyF3", interpreter.IntValue{V: rl.KeyF3}, true)
	env.Define("KeyF4", interpreter.IntValue{V: rl.KeyF4}, true)
	env.Define("KeyF5", interpreter.IntValue{V: rl.KeyF5}, true)
	env.Define("KeyF6", interpreter.IntValue{V: rl.KeyF6}, true)
	env.Define("KeyF7", interpreter.IntValue{V: rl.KeyF7}, true)
	env.Define("KeyF8", interpreter.IntValue{V: rl.KeyF8}, true)
	env.Define("KeyF9", interpreter.IntValue{V: rl.KeyF9}, true)
	env.Define("KeyF10", interpreter.IntValue{V: rl.KeyF10}, true)
	env.Define("KeyF11", interpreter.IntValue{V: rl.KeyF11}, true)
	env.Define("KeyF12", interpreter.IntValue{V: rl.KeyF12}, true)
	env.Define("KeyPeriod", interpreter.IntValue{V: rl.KeyPeriod}, true)
	env.Define("KeyComma", interpreter.IntValue{V: rl.KeyComma}, true)
	env.Define("KeySpace", interpreter.IntValue{V: rl.KeySpace}, true)
	env.Define("KeyEnter", interpreter.IntValue{V: rl.KeyEnter}, true)
	env.Define("KeyCapsLock", interpreter.IntValue{V: rl.KeyCapsLock}, true)
	env.Define("KeyLeftShift", interpreter.IntValue{V: rl.KeyLeftShift}, true)
	env.Define("KeyLShift", interpreter.IntValue{V: rl.KeyLeftShift}, true)
	env.Define("KeyRightShift", interpreter.IntValue{V: rl.KeyRightShift}, true)
	env.Define("KeyRShift", interpreter.IntValue{V: rl.KeyRightShift}, true)
	env.Define("KeyNumLock", interpreter.IntValue{V: rl.KeyNumLock}, true)
	env.Define("KeyTab", interpreter.IntValue{V: rl.KeyTab}, true)
	env.Define("KeyLeftBracket", interpreter.IntValue{V: rl.KeyLeftBracket}, true)
	env.Define("KeyRightBracket", interpreter.IntValue{V: rl.KeyRightBracket}, true)
	env.Define("KeyDelete", interpreter.IntValue{V: rl.KeyDelete}, true)
	env.Define("KeyPrintScreen", interpreter.IntValue{V: rl.KeyPrintScreen}, true)
	env.Define("KeyPause", interpreter.IntValue{V: rl.KeyPause}, true)
	env.Define("KeyEnd", interpreter.IntValue{V: rl.KeyEnd}, true)
	env.Define("KeyHome", interpreter.IntValue{V: rl.KeyHome}, true)
	env.Define("KeyLeftLAlt", interpreter.IntValue{V: rl.KeyLeftAlt}, true)
	env.Define("KeyRightAlt", interpreter.IntValue{V: rl.KeyRightAlt}, true)
	env.Define("KeyRightControl", interpreter.IntValue{V: rl.KeyRightControl}, true)
	env.Define("KeyLeftControl", interpreter.IntValue{V: rl.KeyLeftControl}, true)

	env.Define("FlagWindowResizable", interpreter.IntValue{V: rl.FlagWindowResizable}, true)
	env.Define("FlagWindowFullscreen", interpreter.IntValue{V: rl.FlagFullscreenMode}, true)
	env.Define("FlagWindowMaximized", interpreter.IntValue{V: rl.FlagWindowMaximized}, true)
	env.Define("FlagWindowMinimized", interpreter.IntValue{V: rl.FlagWindowMinimized}, true)
	env.Define("FlagWindowAlwaysRun", interpreter.IntValue{V: rl.FlagWindowAlwaysRun}, true)
	env.Define("FlagWindowVsync", interpreter.IntValue{V: rl.FlagVsyncHint}, true)
	env.Define("FlagWindowHighDPI", interpreter.IntValue{V: rl.FlagWindowHighdpi}, true)
	env.Define("FlagWindowBorderless", interpreter.IntValue{V: rl.FlagBorderlessWindowedMode}, true)
	env.Define("FlagWindowMsaa4x", interpreter.IntValue{V: rl.FlagMsaa4xHint}, true)

	module := interpreter.ModuleValue{
		Name:    "rl",
		Env:     env,
		TypeEnv: TypeEnv,
	}

	return module, nil
}
