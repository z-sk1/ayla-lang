package main

import (
	"math/rand"
	"time"

	"github.com/z-sk1/ayla-lang/runner"

	_ "github.com/z-sk1/ayla-lang/stdlib"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	runner.Main()
}
