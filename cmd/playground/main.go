package main

import (
	"syscall/js"

	"github.com/z-sk1/ayla-lang/runner"
	_ "github.com/z-sk1/ayla-lang/stdlib"
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
