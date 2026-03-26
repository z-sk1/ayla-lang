package registry

import "github.com/z-sk1/ayla-lang/interpreter"

func Register(name string, loader interpreter.NativeLoader) {
	interpreter.NativeModules[name] = loader
}
