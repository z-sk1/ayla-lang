package math

import (
	"math"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/registry"
)

func init() {
	registry.Register("math", LoadMathModule)
}

func LoadMathModule(i *interpreter.Interpreter) (interpreter.ModuleValue, error) {
	env := interpreter.NewEnvironment(i.Env)

	// functions
	env.Define("Abs", interpreter.WrapFloat1("Abs", math.Abs), false)
	env.Define("Sin", interpreter.WrapFloat1("Sin", math.Sin), false)
	env.Define("Asin", interpreter.WrapFloat1("Asin", math.Asin), false)
	env.Define("Cos", interpreter.WrapFloat1("Cos", math.Cos), false)
	env.Define("Acos", interpreter.WrapFloat1("Acos", math.Acos), false)
	env.Define("Tan", interpreter.WrapFloat1("Tan", math.Tan), false)
	env.Define("Atan", interpreter.WrapFloat1("Atan", math.Atan), false)
	env.Define("Sqrt", interpreter.WrapFloat1("Sqrt", math.Sqrt), false)
	env.Define("Log", interpreter.WrapFloat1("Log", math.Log), false)
	env.Define("Exp", interpreter.WrapFloat1("Exp", math.Exp), false)
	env.Define("Floor", interpreter.WrapFloat1("Floor", math.Floor), false)
	env.Define("Ceil", interpreter.WrapFloat1("Ceil", math.Ceil), false)
	env.Define("Round", interpreter.WrapFloat1("Round", math.Round), false)
	env.Define("RoundToEven", interpreter.WrapFloat1("RoundToEven", math.RoundToEven), false)
	env.Define("Trunc", interpreter.WrapFloat1("Trunc", math.Trunc), false)

	env.Define("Max", interpreter.WrapFloat2("Max", math.Max), false)
	env.Define("Min", interpreter.WrapFloat2("Min", math.Min), false)
	env.Define("Pow", interpreter.WrapFloat2("Pow", math.Pow), false)
	env.Define("Remainder", interpreter.WrapFloat2("Remainder", math.Remainder), false)
	env.Define("Atan2", interpreter.WrapFloat2("Atan2", math.Atan2), false)

	// constants
	env.Define("Pi", interpreter.FloatValue{V: math.Pi}, true)
	env.Define("Phi", interpreter.FloatValue{V: math.Phi}, true)
	env.Define("E", interpreter.FloatValue{V: math.E}, true)
	env.Define("Sqrt2", interpreter.FloatValue{V: math.Sqrt2}, true)
	env.Define("SqrtPi", interpreter.FloatValue{V: math.SqrtPi}, true)
	env.Define("SqrtPhi", interpreter.FloatValue{V: math.SqrtPhi}, true)
	env.Define("SqrtE", interpreter.FloatValue{V: math.SqrtE}, true)

	module := interpreter.ModuleValue{
		Name: "math",
		Env:  env,
	}

	return module, nil
}
