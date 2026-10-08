package main

import (
	"syscall/js"

	"github.com/z-sk1/ayla-lang/runner"
)

func main() {
	js.Global().Set("runAyla", js.FuncOf(func(this js.Value, args []js.Value) any {
		code := args[0].String()

		result, err := runner.Execute(code)
		if err != nil {
			return err.Error()
		}

		return result
	}))

	select {}
}
