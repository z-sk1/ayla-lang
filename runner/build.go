package runner

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Build() {
	args := os.Args[2:]

	filename := ""
	output := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "-o":
			if i+1 >= len(args) {
				fmt.Println("Expected filename after -o")
				return
			}
			output = args[i+1]
			i++

		default:
			filename = arg
		}
	}

	if filename == "" {
		fmt.Println("No input file provided")
		return
	}

	if output == "" {
		base := filepath.Base(filename)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		output = name + ".exe"
	}

	src, _, err := readSourceFile(filename)
	if err != nil {
		fmt.Println(err)
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		fmt.Println(err)
		return
	}

	data, err := os.ReadFile(exePath)
	if err != nil {
		fmt.Println(err)
		return
	}

	startMarker := []byte("\n__AYLA_SCRIPT_START__\n")
	endMarker := []byte("\n__AYLA_SCRIPT_END__\n")

	start := bytes.LastIndex(data, startMarker)
	end := bytes.LastIndex(data, endMarker)

	if start != -1 && end != -1 && end > start {
		data = data[:start]
	}

	out, err := os.Create(output)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer out.Close()

	out.Write(data)
	out.Write(startMarker)
	out.Write([]byte(src))
	out.Write(endMarker)

	fmt.Println("built executable:", output)
}
