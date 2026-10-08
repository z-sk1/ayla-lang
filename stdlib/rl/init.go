//go:build rl

package rl

import "github.com/z-sk1/ayla-lang/registry"

func init() {
	registry.Register("rl", Load)
}
