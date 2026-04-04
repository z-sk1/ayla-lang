package rl

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/registry"
)

func init() {
	registry.Register("rl", Load)
}

func Load(i *interpreter.Interpreter) (interpreter.ModuleValue, error) {
	env := interpreter.NewEnvironment(i.Env)
	TypeEnv := make(map[string]interpreter.TypeValue)

	min := 0.0
	max := 255.0

	uint8Type := &interpreter.TypeInfo{
		Name:         "int<0..255>",
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

	TypeEnv["Rectangle"] = interpreter.TypeValue{
		TypeInfo: &interpreter.TypeInfo{
			Name: "Rectangle",
			Kind: interpreter.TypeStruct,
			Fields: map[string]*interpreter.TypeInfo{
				"X":      i.TypeEnv["float"].TypeInfo,
				"Y":      i.TypeEnv["float"].TypeInfo,
				"Width":  i.TypeEnv["float"].TypeInfo,
				"Height": i.TypeEnv["float"].TypeInfo,
			},
		},
	}

	TypeEnv["Texture2D"] = interpreter.TypeValue{
		TypeInfo: &interpreter.TypeInfo{
			Name:   "Texture2D",
			Kind:   interpreter.TypeStruct,
			Fields: nil,
			Opaque: true,
		},
	}

	TypeEnv["RenderTexture2D"] = interpreter.TypeValue{
		TypeInfo: &interpreter.TypeInfo{
			Name:   "RenderTexture2D",
			Kind:   interpreter.TypeStruct,
			Fields: nil,
			Opaque: true,
		},
	}

	TypeEnv["Camera2D"] = interpreter.TypeValue{
		TypeInfo: &interpreter.TypeInfo{
			Name: "Camera2D",
			Kind: interpreter.TypeStruct,
			Fields: map[string]*interpreter.TypeInfo{
				"Offset":   TypeEnv["Vector2"].TypeInfo,
				"Target":   TypeEnv["Vector2"].TypeInfo,
				"Rotation": i.TypeEnv["float"].TypeInfo,
				"Zoom":     i.TypeEnv["float"].TypeInfo,
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

	TypeEnv["Font"] = interpreter.TypeValue{
		TypeInfo: &interpreter.TypeInfo{
			Name:   "Font",
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

	env.Define("SetWindowPosition", &interpreter.BuiltinFunc{
		Name:  "SetWindowPosition",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			x, err := interpreter.ArgInt(node, args, 0, "rl.SetWindowPosition")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			y, err := interpreter.ArgInt(node, args, 1, "rl.SetWindowPosition")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetWindowPosition(x, y)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("GetWindowPosition", &interpreter.BuiltinFunc{
		Name:  "GetWindowPosition",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return interpreter.MakeVector2(rl.GetWindowPosition(), TypeEnv), nil
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

	env.Define("Vector2Add", interpreter.WrapVector2D2("rl.Vector2Add", TypeEnv, rl.Vector2Add), false)
	env.Define("Vector2Subtract", interpreter.WrapVector2D2("rl.Vector2Subtract", TypeEnv, rl.Vector2Subtract), false)
	env.Define("Vector2Multiply", interpreter.WrapVector2D2("rl.Vector2Multiply", TypeEnv, rl.Vector2Multiply), false)
	env.Define("Vector2Divide", interpreter.WrapVector2D2("rl.Vector2Divide", TypeEnv, rl.Vector2Divide), false)
	env.Define("Vector2Scale", interpreter.WrapVector2DFloat("rl.Vector2Scale", TypeEnv, rl.Vector2Scale), false)
	env.Define("Vector2Negate", interpreter.WrapVector2D1("rl.Vector2Negate", TypeEnv, rl.Vector2Negate), false)
	env.Define("Vector2Length", interpreter.WrapVector2D1RFloat("rl.Vector2Length", TypeEnv, rl.Vector2Length), false)
	env.Define("Vector2LengthSqr", interpreter.WrapVector2D1RFloat("rl.Vector2LengthSqr", TypeEnv, rl.Vector2LengthSqr), false)
	env.Define("Vector2Distance", interpreter.WrapVector2D2RFloat("rl.Vector2Distance", TypeEnv, rl.Vector2Distance), false)
	env.Define("Vector2DistanceSqr", interpreter.WrapVector2D2RFloat("rl.Vector2DistanceSqr", TypeEnv, rl.Vector2DistanceSqr), false)
	env.Define("Vector2Normalize", interpreter.WrapVector2D1("rl.Vector2Normalize", TypeEnv, rl.Vector2Normalize), false)
	env.Define("Vector2Dot", interpreter.WrapVector2D2RFloat("rl.Vector2Dot", TypeEnv, rl.Vector2DotProduct), false)
	env.Define("Vector2Angle", interpreter.WrapVector2D2RFloat("rl.Vector2Angle", TypeEnv, rl.Vector2Angle), false)
	env.Define("Vector2Lerp", &interpreter.BuiltinFunc{
		Name:  "Vector2Lerp",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.Vector2Lerp")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			v2, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.Vector2Lerp")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			t, err := interpreter.ArgFloat(node, args, 2, "rl.Vector2Lerp")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.MakeVector2(rl.Vector2Lerp(v, v2, float32(t)), TypeEnv), nil
		}}, false)
	env.Define("Vector2Equals", &interpreter.BuiltinFunc{
		Name:  "Vector2Equals",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.Vector2Equals")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			v2, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.Vector2Equals")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.BoolValue{V: rl.Vector2Equals(v, v2)}, nil
		}}, false)
	env.Define("Vector2Zero", &interpreter.BuiltinFunc{
		Name:  "Vector2Zero",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return interpreter.MakeVector2(rl.Vector2Zero(), TypeEnv), nil
		}}, false)
	env.Define("Vector2One", &interpreter.BuiltinFunc{
		Name:  "Vector2One",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return interpreter.MakeVector2(rl.Vector2Zero(), TypeEnv), nil
		}}, false)
	env.Define("Vector2Clamp", &interpreter.BuiltinFunc{
		Name:  "Vector2Clamp",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			v, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.Vector2Clamp")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			min, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.Vector2Clamp")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			max, err := interpreter.ArgVector2(node, i, TypeEnv, args, 2, "rl.Vector2Clamp")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return interpreter.MakeVector2(rl.Vector2Clamp(v, min, max), TypeEnv), nil
		}}, false)

	env.Define("NewRectangle", &interpreter.BuiltinFunc{
		Name:  "NewRectangle",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pos, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.NewRectangle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			size, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.NewRectangle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return &interpreter.StructValue{
				TypeName: TypeEnv["Rectangle"].TypeInfo,
				Fields: map[string]interpreter.Value{
					"X":      interpreter.FloatValue{V: float64(pos.X)},
					"Y":      interpreter.FloatValue{V: float64(pos.Y)},
					"Width":  interpreter.FloatValue{V: float64(size.X)},
					"Height": interpreter.FloatValue{V: float64(size.Y)},
				},
			}, nil
		},
	}, false)

	env.Define("NewCamera2D", &interpreter.BuiltinFunc{
		Name:  "NewCamera2D",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			offset, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.NewCamera2D")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			target, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.NewCamera2D")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rot, err := interpreter.ArgFloat(node, args, 2, "rl.NewCamera2D")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			zoom, err := interpreter.ArgFloat(node, args, 3, "rl.NewCamera2D")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return &interpreter.StructValue{
				TypeName: TypeEnv["Camera2D"].TypeInfo,
				Fields: map[string]interpreter.Value{
					"Offset":   interpreter.MakeVector2(offset, TypeEnv),
					"Target":   interpreter.MakeVector2(target, TypeEnv),
					"Rotation": interpreter.FloatValue{V: rot},
					"Zoom":     interpreter.FloatValue{V: zoom},
				},
			}, nil
		},
	}, false)

	env.Define("BeginMode2D", &interpreter.BuiltinFunc{
		Name:  "BeginMode2D",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			cam, err := interpreter.ArgCamera2D(node, i, TypeEnv, args, 0, "rl.BeginMode2D")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.BeginMode2D(cam)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("EndMode2D", &interpreter.BuiltinFunc{
		Name:  "EndMode2D",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rl.EndMode2D()
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("GetWorldToScreen2D", &interpreter.BuiltinFunc{
		Name:  "GetWorldToScreen2D",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pos, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.GetWorldToScreen2D")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			cam, err := interpreter.ArgCamera2D(node, i, TypeEnv, args, 1, "rl.GetWorldToScreen2D")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			res := rl.GetWorldToScreen2D(pos, cam)
			return interpreter.MakeVector2(res, TypeEnv), nil
		},
	}, false)

	env.Define("GetScreenToWorld2D", &interpreter.BuiltinFunc{
		Name:  "GetScreenToWorld2D",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pos, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.GetScreenToWorld2D")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			cam, err := interpreter.ArgCamera2D(node, i, TypeEnv, args, 1, "rl.GetScreenToWorld2D")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			res := rl.GetScreenToWorld2D(pos, cam)
			return interpreter.MakeVector2(res, TypeEnv), nil
		},
	}, false)

	env.Define("LoadFont", &interpreter.BuiltinFunc{
		Name:  "LoadFont",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "rl.LoadFont")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return &interpreter.StructValue{
				TypeName: TypeEnv["Font"].TypeInfo,
				Native:   rl.LoadFont(path),
			}, nil
		},
	}, false)

	env.Define("UnloadFont", &interpreter.BuiltinFunc{
		Name:  "UnloadFont",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			font, err := interpreter.ArgFont(node, i, TypeEnv, args, 0, "rl.UnloadFont")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.UnloadFont(font)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("GetFontDefault", &interpreter.BuiltinFunc{
		Name:  "GetFontDefault",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			return &interpreter.StructValue{
				TypeName: TypeEnv["Font"].TypeInfo,
				Native:   rl.GetFontDefault(),
			}, nil
		},
	}, false)

	env.Define("LoadRenderTexture", &interpreter.BuiltinFunc{
		Name:  "LoadRenderTexture",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			w, err := interpreter.ArgInt(node, args, 0, "rl.LoadRenderTexture")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			h, err := interpreter.ArgInt(node, args, 1, "rl.LoadRenderTexture")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return &interpreter.StructValue{
				TypeName: TypeEnv["RenderTexture2D"].TypeInfo,
				Native:   rl.LoadRenderTexture(int32(w), int32(h)),
			}, nil
		},
	}, false)

	env.Define("UnloadRenderTexture", &interpreter.BuiltinFunc{
		Name:  "UnloadRenderTexture",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			ren, err := interpreter.ArgRenderTexture2D(node, i, TypeEnv, args, 0, "rl.UnloadRenderTexture")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.UnloadRenderTexture(ren)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("BeginTextureMode", &interpreter.BuiltinFunc{
		Name:  "BeginTextureMode",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rt, err := interpreter.ArgRenderTexture2D(node, i, TypeEnv, args, 0, "rl.BeginTextureMode")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.BeginTextureMode(rt)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("EndTextureMode", &interpreter.BuiltinFunc{
		Name:  "EndTextureMode",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rl.EndTextureMode()
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("GetTextureFromRender", &interpreter.BuiltinFunc{
		Name:  "GetTextureFromRender",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rt, err := interpreter.ArgRenderTexture2D(node, i, TypeEnv, args, 0, "rl.GetTextureFromRender")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return &interpreter.StructValue{
				TypeName: TypeEnv["RenderTexture2D"].TypeInfo,
				Native:   rt.Texture,
			}, nil
		},
	}, false)

	env.Define("LoadTexture", &interpreter.BuiltinFunc{
		Name:  "LoadTexture",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			path, err := interpreter.ArgString(node, args, 0, "rl.LoadTexture")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			return &interpreter.StructValue{
				TypeName: TypeEnv["Texture2D"].TypeInfo,
				Native:   rl.LoadTexture(path),
			}, nil
		},
	}, false)

	env.Define("UnloadTexture", &interpreter.BuiltinFunc{
		Name:  "UnloadTexture",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			tex, err := interpreter.ArgTexture2D(node, i, TypeEnv, args, 0, "rl.UnloadTexture")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.UnloadTexture(tex)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawTexture", &interpreter.BuiltinFunc{
		Name:  "DrawTexture",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			tex, err := interpreter.ArgTexture2D(node, i, TypeEnv, args, 0, "rl.DrawTexture")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			pos, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.DrawTexture")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			tint, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawTexture")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawTexture(tex, int32(pos.X), int32(pos.Y), tint)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawTextureRec", &interpreter.BuiltinFunc{
		Name:  "DrawTextureRec",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			tex, err := interpreter.ArgTexture2D(node, i, TypeEnv, args, 0, "rl.DrawTextureRec")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			src, err := interpreter.ArgRectangle(node, i, TypeEnv, args, 1, "rl.DrawTextureRec")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			pos, err := interpreter.ArgVector2(node, i, TypeEnv, args, 2, "rl.DrawTextureRec")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			tint, err := interpreter.ArgColor(node, TypeEnv, args, 3, "rl.DrawTectureRec")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawTextureRec(tex, src, pos, tint)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawTextureEx", &interpreter.BuiltinFunc{
		Name:  "DrawTextureEx",
		Arity: 5,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			tex, err := interpreter.ArgTexture2D(node, i, TypeEnv, args, 0, "rl.DrawTextureEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			pos, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.DrawTextureEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rot, err := interpreter.ArgFloat(node, args, 2, "rl.DrawTextureEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			scale, err := interpreter.ArgFloat(node, args, 3, "rl.DrawTextureEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			tint, err := interpreter.ArgColor(node, TypeEnv, args, 4, "rl.DrawTextureEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawTextureEx(tex, pos, float32(rot), float32(scale), tint)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawTexturePro", &interpreter.BuiltinFunc{
		Name:  "DrawTexturePro",
		Arity: 6,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			tex, err := interpreter.ArgTexture2D(node, i, TypeEnv, args, 0, "rl.DrawTexturePro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			src, err := interpreter.ArgRectangle(node, i, TypeEnv, args, 1, "rl.DrawTexturePro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			dest, err := interpreter.ArgRectangle(node, i, TypeEnv, args, 2, "rl.DrawTexturePro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			org, err := interpreter.ArgVector2(node, i, TypeEnv, args, 3, "rl.DrawTexturePro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rot, err := interpreter.ArgFloat(node, args, 4, "rl.DrawTexturePro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			tint, err := interpreter.ArgColor(node, TypeEnv, args, 5, "rl.DrawTexturePro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawTexturePro(tex, src, dest, org, float32(rot), tint)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawRectangle", &interpreter.BuiltinFunc{
		Name:  "DrawRect",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawRectangle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.DrawRectangle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawRectangle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawRectangleV(pv, sv, col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawRectangleRec", &interpreter.BuiltinFunc{
		Name:  "DrawRect",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rec, err := interpreter.ArgRectangle(node, i, TypeEnv, args, 0, "rl.DrawRectangleRec")

			col, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawRectangleRec")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawRectangleRec(rec, col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawRectanglePro", &interpreter.BuiltinFunc{
		Name:  "DrawRectanglePro",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rec, err := interpreter.ArgRectangle(node, i, TypeEnv, args, 0, "rl.DrawRectanglePro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			org, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.DrawRectanglePro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rot, err := interpreter.ArgFloat(node, args, 2, "rl.DrawRectanglePro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 3, "rl.DrawRectanglePro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawRectanglePro(rec, org, float32(rot), col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawRectangleLines", &interpreter.BuiltinFunc{
		Name:  "DrawRectLines",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawRectangleLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.DrawRectangleLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawRectangleLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawRectangleLines(int32(pv.X), int32(pv.Y), int32(sv.X), int32(sv.Y), col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawRectangleLinesRec", &interpreter.BuiltinFunc{
		Name:  "DrawRectLinesRec",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rec, err := interpreter.ArgRectangle(node, i, TypeEnv, args, 0, "rl.DrawRectangleLinesRec")

			col, err := interpreter.ArgColor(node, TypeEnv, args, 1, "rl.DrawRectangleLinesRec")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawRectangleLines(int32(rec.X), int32(rec.Y), int32(rec.X), int32(rec.Y), col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawRectangleLinesEx", &interpreter.BuiltinFunc{
		Name:  "DrawRectangleLinesEx",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rec, err := interpreter.ArgRectangle(node, i, TypeEnv, args, 0, "rl.DrawRectangleLinesEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			lineT, err := interpreter.ArgFloat(node, args, 1, "rl.DrawRectangleLinesEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawRectangleLinesEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawRectangleLinesEx(rec, float32(lineT), col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawRectangleRounded", &interpreter.BuiltinFunc{
		Name:  "DrawRectangleRounded",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rec, err := interpreter.ArgRectangle(node, i, TypeEnv, args, 0, "rl.DrawRectangleRounded")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			round, err := interpreter.ArgFloat(node, args, 1, "rl.DrawRectangleRounded")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			segs, err := interpreter.ArgInt(node, args, 2, "rl.DrawRectangleRounded")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 3, "rl.DrawRectangleRounded")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawRectangleRounded(rec, float32(round), int32(segs), col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawRectangleRoundedLines", &interpreter.BuiltinFunc{
		Name:  "DrawRectangleRoundedLines",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rec, err := interpreter.ArgRectangle(node, i, TypeEnv, args, 0, "rl.DrawRectangleRoundedLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			round, err := interpreter.ArgFloat(node, args, 1, "rl.DrawRectangleRoundedLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			segs, err := interpreter.ArgInt(node, args, 2, "rl.DrawRectangleRoundedLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 3, "rl.DrawRectangleRoundedLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawRectangleRoundedLines(rec, float32(round), int32(segs), col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawRectangleGradientH", &interpreter.BuiltinFunc{
		Name:  "rl.DrawRectangleGradientH",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rec, err := interpreter.ArgRectangle(node, i, TypeEnv, args, 0, "rl.DrawRectangleGradientH")

			lcol, err := interpreter.ArgColor(node, TypeEnv, args, 1, "rl.DrawRectangleGradientH")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rcol, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawRectangleGradientH")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawRectangleGradientH(int32(rec.X), int32(rec.Y), int32(rec.Width), int32(rec.Height), lcol, rcol)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawRectangleGradientV", &interpreter.BuiltinFunc{
		Name:  "rl.DrawRectangleGradientV",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rec, err := interpreter.ArgRectangle(node, i, TypeEnv, args, 0, "rl.DrawRectangleGradientV")

			lcol, err := interpreter.ArgColor(node, TypeEnv, args, 1, "rl.DrawRectangleGradientV")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rcol, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawRectangleGradientV")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawRectangleGradientH(int32(rec.X), int32(rec.Y), int32(rec.Width), int32(rec.Height), lcol, rcol)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawRectangleGradientEx", &interpreter.BuiltinFunc{
		Name:  "rl.DrawRectangleGradientEx",
		Arity: 5,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			rec, err := interpreter.ArgRectangle(node, i, TypeEnv, args, 0, "rl.DrawRectangleGradientEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			tlcol, err := interpreter.ArgColor(node, TypeEnv, args, 1, "rl.DrawRectangleGradientEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			blcol, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawRectangleGradientEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			trcol, err := interpreter.ArgColor(node, TypeEnv, args, 3, "rl.DrawRectangleGradientEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			brcol, err := interpreter.ArgColor(node, TypeEnv, args, 4, "rl.DrawRectangleGradientEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawRectangleGradientEx(rec, tlcol, blcol, trcol, brcol)
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

	env.Define("DrawCircleGradient", &interpreter.BuiltinFunc{
		Name:  "DrawCircleGradient",
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

			icol, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawCircle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			ocol, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawCircle")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			r := float32(rVal)

			rl.DrawCircleGradient(int32(pv.X), int32(pv.Y), r, icol, ocol)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawCircleSector", &interpreter.BuiltinFunc{
		Name:  "DrawCircleSector",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawCircleSector")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			r, err := interpreter.ArgFloat(node, args, 1, "rl.DrawCircleSector")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			startA, err := interpreter.ArgFloat(node, args, 2, "rl.DrawCircleSector")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			endA, err := interpreter.ArgFloat(node, args, 3, "rl.DrawCircleSector")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			segs, err := interpreter.ArgInt(node, args, 4, "rl.DrawCircleSector")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 5, "rl.DrawCircleSector")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawCircleSector(pv, float32(r), float32(startA), float32(endA), int32(segs), col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawCircleSectorLines", &interpreter.BuiltinFunc{
		Name:  "DrawCircleSectorLines",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawCircleSectorLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			r, err := interpreter.ArgFloat(node, args, 1, "rl.DrawCircleSectorLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			startA, err := interpreter.ArgFloat(node, args, 2, "rl.DrawCircleSectorLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			endA, err := interpreter.ArgFloat(node, args, 3, "rl.DrawCircleSectorLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			segs, err := interpreter.ArgInt(node, args, 4, "rl.DrawCircleSectorLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 5, "rl.DrawCircleSectorLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawCircleSectorLines(pv, float32(r), float32(startA), float32(endA), int32(segs), col)
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

	env.Define("DrawTextEx", &interpreter.BuiltinFunc{
		Name:  "DrawTextEx",
		Arity: 6,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			fontVal, err := interpreter.ArgFont(node, i, TypeEnv, args, 0, "rl.DrawTextEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			txtVal, err := interpreter.ArgString(node, args, 1, "rl.DrawTextEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 2, "rl.DrawTextEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sizeVal, err := interpreter.ArgInt(node, args, 3, "rl.DrawTextEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			spaceVal, err := interpreter.ArgInt(node, args, 4, "rl.DrawTextEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 5, "rl.DrawTextEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			font := fontVal
			txt := txtVal
			size := float32(sizeVal)
			space := float32(spaceVal)

			rl.DrawTextEx(font, txt, pv, size, space, col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawTextPro", &interpreter.BuiltinFunc{
		Name:  "DrawTextPro",
		Arity: 8,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			fontVal, err := interpreter.ArgFont(node, i, TypeEnv, args, 0, "rl.DrawTextPro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			txtVal, err := interpreter.ArgString(node, args, 1, "rl.DrawTextPro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 2, "rl.DrawTextPro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			ov, err := interpreter.ArgVector2(node, i, TypeEnv, args, 3, "rl.DrawTextPro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rotVal, err := interpreter.ArgFloat(node, args, 4, "rl.DrawTextPro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sizeVal, err := interpreter.ArgInt(node, args, 5, "rl.DrawTextPro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			spaceVal, err := interpreter.ArgInt(node, args, 6, "rl.DrawTextPro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 7, "rl.DrawTextPro")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			font := fontVal
			txt := txtVal
			rot := float32(rotVal)
			size := float32(sizeVal)
			space := float32(spaceVal)

			rl.DrawTextPro(font, txt, pv, ov, rot, size, space, col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("MeasureText", &interpreter.BuiltinFunc{
		Name:  "MeasureText",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			text, err := interpreter.ArgString(node, args, 0, "rl.MeasureText")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			size, err := interpreter.ArgInt(node, args, 1, "rl.MeasureText")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			res := rl.MeasureText(text, int32(size))
			return interpreter.IntValue{V: int(res)}, nil
		},
	}, false)

	env.Define("MeasureTextEx", &interpreter.BuiltinFunc{
		Name:  "MeasureTextEx",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			font, err := interpreter.ArgFont(node, i, TypeEnv, args, 0, "rl.MeasureTextEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			text, err := interpreter.ArgString(node, args, 1, "rl.MeasureTextEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			size, err := interpreter.ArgFloat(node, args, 2, "rl.MeasureTextEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			spacing, err := interpreter.ArgFloat(node, args, 3, "rl.MeasureTextEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			result := rl.MeasureTextEx(font, text, float32(size), float32(spacing))
			return interpreter.MakeVector2(result, TypeEnv), nil
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

	env.Define("DrawLineEx", &interpreter.BuiltinFunc{
		Name:  "DrawLineEx",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawLineEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.DrawLineEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			thick, err := interpreter.ArgFloat(node, args, 2, "rl.DrawLineEx")

			col, err := interpreter.ArgColor(node, TypeEnv, args, 3, "rl.DrawLineEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawLineEx(pv, sv, float32(thick), col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawLineBezier", &interpreter.BuiltinFunc{
		Name:  "DrawLineBezier",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			pv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawLineBezier")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sv, err := interpreter.ArgVector2(node, i, TypeEnv, args, 1, "rl.DrawLineBezier")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			thick, err := interpreter.ArgFloat(node, args, 2, "rl.DrawLineBezier")

			col, err := interpreter.ArgColor(node, TypeEnv, args, 3, "rl.DrawLineBezier")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawLineBezier(pv, sv, float32(thick), col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawLineStrip", &interpreter.BuiltinFunc{
		Name:  "DrawLineStrip",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			arr, err := interpreter.ArgArray(node, args, 0, "rl.DrawLineStrip", "rl.Vector2")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			vectors := []rl.Vector2{}

			for _, v := range arr.Elements {
				if _, ok := v.(*interpreter.StructValue); !ok {
					return interpreter.NilValue{}, interpreter.NewRuntimeError(node, "rl.DrawLineStrip: first argument must be a []rl.Vector2")
				}

				if !interpreter.TypesAssignable(v.(*interpreter.StructValue).TypeName, TypeEnv["Vector2"].TypeInfo) {
					return interpreter.NilValue{}, interpreter.NewRuntimeError(node, "rl.DrawLineStrip: first argument must be a []rl.Vector2")
				}

				vectors = append(vectors, rl.Vector2{
					X: float32(v.(*interpreter.StructValue).Fields["X"].(interpreter.FloatValue).V),
					Y: float32(v.(*interpreter.StructValue).Fields["Y"].(interpreter.FloatValue).V),
				})
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawLineStrip")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawLineStrip(vectors, col)
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

	env.Define("DrawTriangleStrip", &interpreter.BuiltinFunc{
		Name:  "DrawTriangleStrip",
		Arity: 3,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			arr, err := interpreter.ArgArray(node, args, 0, "rl.DrawTriangleStrip", "rl.Vector2")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			vectors := []rl.Vector2{}

			for _, v := range arr.Elements {
				if _, ok := v.(*interpreter.StructValue); !ok {
					return interpreter.NilValue{}, interpreter.NewRuntimeError(node, "rl.DrawTriangleStrip: first argument must be a []rl.Vector2")
				}

				if !interpreter.TypesAssignable(v.(*interpreter.StructValue).TypeName, TypeEnv["Vector2"].TypeInfo) {
					return interpreter.NilValue{}, interpreter.NewRuntimeError(node, "rl.DrawTriangleStrip: first argument must be a []rl.Vector2")
				}

				vectors = append(vectors, rl.Vector2{
					X: float32(v.(*interpreter.StructValue).Fields["X"].(interpreter.FloatValue).V),
					Y: float32(v.(*interpreter.StructValue).Fields["Y"].(interpreter.FloatValue).V),
				})
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 2, "rl.DrawTriangleStrip")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawTriangleStrip(vectors, col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawPoly", &interpreter.BuiltinFunc{
		Name:  "DrawPoly",
		Arity: 5,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			center, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawPoly")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sides, err := interpreter.ArgInt(node, args, 1, "rl.DrawPoly")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			radius, err := interpreter.ArgFloat(node, args, 2, "rl.DrawPoly")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rot, err := interpreter.ArgFloat(node, args, 3, "rl.DrawPoly")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 4, "rl.DrawPoly")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawPoly(center, int32(sides), float32(radius), float32(rot), col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawPolyLines", &interpreter.BuiltinFunc{
		Name:  "DrawPolyLines",
		Arity: 5,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			center, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawPolyLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sides, err := interpreter.ArgInt(node, args, 1, "rl.DrawPolyLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			radius, err := interpreter.ArgFloat(node, args, 2, "rl.DrawPolyLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rot, err := interpreter.ArgFloat(node, args, 3, "rl.DrawPolyLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 4, "rl.DrawPolyLines")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawPolyLines(center, int32(sides), float32(radius), float32(rot), col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("DrawPolyLinesEx", &interpreter.BuiltinFunc{
		Name:  "DrawPolyLinesEx",
		Arity: 6,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			center, err := interpreter.ArgVector2(node, i, TypeEnv, args, 0, "rl.DrawPolyLinesEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			sides, err := interpreter.ArgInt(node, args, 1, "rl.DrawPolyLinesEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			radius, err := interpreter.ArgFloat(node, args, 2, "rl.DrawPolyLinesEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rot, err := interpreter.ArgFloat(node, args, 3, "rl.DrawPolyLinesEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			thick, err := interpreter.ArgFloat(node, args, 4, "rl.DrawPolyLinesEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgColor(node, TypeEnv, args, 5, "rl.DrawPolyLinesEx")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.DrawPolyLinesEx(center, int32(sides), float32(radius), float32(rot), float32(thick), col)
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

	env.Define("SetMousePosition", &interpreter.BuiltinFunc{
		Name:  "SetMousePosition",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			x, err := interpreter.ArgInt(node, args, 0, "rl.SetMousePosition")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			y, err := interpreter.ArgInt(node, args, 1, "rl.SetMousePosition")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rl.SetMousePosition(x, y)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("GetMousePosition", &interpreter.BuiltinFunc{
		Name:  "GetMousePosition",
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

			rl.UnloadSound(*sound)
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

			rl.PlaySound(*sound)
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

			rl.StopSound(*sound)
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

			return interpreter.BoolValue{V: rl.IsSoundPlaying(*sound)}, nil
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

			rl.PauseSound(*sound)
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

			rl.ResumeSound(*sound)
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

			rl.SetSoundVolume(*sound, float32(volume))
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

			rl.SetSoundPitch(*sound, float32(pitch))
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
