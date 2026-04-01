package term

import (
	"fmt"
	"os"
	"strings"

	"github.com/z-sk1/ayla-lang/interpreter"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/registry"
	"golang.org/x/term"
)

func init() {
	registry.Register("term", Load)
}

func Load(i *interpreter.Interpreter) (interpreter.ModuleValue, error) {
	env := interpreter.NewEnvironment(i.Env)

	env.Define("Clear", &interpreter.BuiltinFunc{
		Name:  "Clear",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			fmt.Print("\033[2J\033[H")
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("Reset", &interpreter.BuiltinFunc{
		Name:  "Reset",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			fmt.Print("\033[0m")
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("MoveTo", &interpreter.BuiltinFunc{
		Name:  "Move",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			row, err := interpreter.ArgInt(node, args, 0, "term.MoveTo")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			col, err := interpreter.ArgInt(node, args, 1, "term.MoveTo")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			fmt.Printf("\033[%d;%dH", row, col)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("MoveUp", &interpreter.BuiltinFunc{
		Name:  "MoveUp",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			n, err := interpreter.ArgInt(node, args, 0, "term.MoveUp")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			fmt.Printf("\x1b[%dA", n)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("MoveDown", &interpreter.BuiltinFunc{
		Name:  "MoveDown",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			n, err := interpreter.ArgInt(node, args, 0, "term.MoveDown")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			fmt.Printf("\x1b[%dB", n)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("MoveLeft", &interpreter.BuiltinFunc{
		Name:  "MoveLeft",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			n, err := interpreter.ArgInt(node, args, 0, "term.MoveLeft")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			fmt.Printf("\x1b[%dD", n)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("MoveRight", &interpreter.BuiltinFunc{
		Name:  "MoveRight",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			n, err := interpreter.ArgInt(node, args, 0, "term.MoveRight")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			fmt.Printf("\x1b[%dC", n)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("HideCursor", &interpreter.BuiltinFunc{
		Name:  "HideCursor",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			fmt.Print("\033[?25l")
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("ShowCursor", &interpreter.BuiltinFunc{
		Name:  "ShowCursor",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			fmt.Print("\033[?25h")
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("SetColor", &interpreter.BuiltinFunc{
		Name:  "SetColor",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			col, err := interpreter.ArgInt(node, args, 0, "term.SetColor")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			fmt.Printf("\033[%dm", col+30)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("SetBGColor", &interpreter.BuiltinFunc{
		Name:  "SetBGColor",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			col, err := interpreter.ArgInt(node, args, 0, "term.SetBGColor")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			fmt.Printf("\033[%dm", col+40)
			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("GetSize", &interpreter.BuiltinFunc{
		Name:  "GetSize",
		Arity: 0,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			w, h, err := term.GetSize(int(os.Stdin.Fd()))
			return interpreter.TupleValue{
				Values: []interpreter.Value{
					interpreter.IntValue{V: w},
					interpreter.IntValue{V: h},
					interpreter.Error{Message: err.Error()},
				},
			}, nil
		},
	}, false)

	env.Define("ProgressBar", &interpreter.BuiltinFunc{
		Name:  "ProgressBar",
		Arity: 4,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {

			text, err := interpreter.ArgString(node, args, 0, "term.ProgressBar")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			val, err := interpreter.ArgInt(node, args, 1, "term.ProgressBar")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			max, err := interpreter.ArgInt(node, args, 2, "term.ProgressBar")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			width, err := interpreter.ArgInt(node, args, 3, "term.ProgressBar")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			if val > max {
				val = max
			}

			filled := (val * width) / max
			percent := (val * 100) / max

			bar := "["
			for j := 0; j < width; j++ {
				if j < filled {
					bar += "█"
				} else {
					bar += "░"
				}
			}
			bar += "]"

			fmt.Printf("\r\033[2K%s %s %d%%", text, bar, percent)

			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("Spinner", &interpreter.BuiltinFunc{
		Name:  "Spinner",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {

			text, err := interpreter.ArgString(node, args, 0, "term.Spinner")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			frame, err := interpreter.ArgInt(node, args, 1, "term.Spinner")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

			spin := frames[frame%len(frames)]
			fmt.Printf("\r\033[2K%s %s", spin, text)

			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("Box", &interpreter.BuiltinFunc{
		Name:  "Box",
		Arity: 5,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {

			x, err := interpreter.ArgInt(node, args, 0, "term.Box")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			y, err := interpreter.ArgInt(node, args, 1, "term.Box")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			w, err := interpreter.ArgInt(node, args, 2, "term.Box")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			h, err := interpreter.ArgInt(node, args, 3, "term.Box")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			title, err := interpreter.ArgString(node, args, 4, "term.Box")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			move := func(px, py int) {
				fmt.Printf("\033[%d;%dH", py, px)
			}

			move(x, y)

			fmt.Print("+")
			for i := 0; i < w-2; i++ {
				fmt.Print("-")
			}
			fmt.Println("+")

			for row := 1; row < h-1; row++ {
				move(x, y+row)

				fmt.Print("|")

				for i := 0; i < w-2; i++ {
					fmt.Print(" ")
				}

				fmt.Println("|")
			}

			move(x, y+h-1)

			fmt.Print("+")
			for i := 0; i < w-2; i++ {
				fmt.Print("-")
			}
			fmt.Println("+")

			if title != "" && len(title)+4 < w {
				move(x+2, y)
				fmt.Printf(" %s ", title)
			}

			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("Table", &interpreter.BuiltinFunc{
		Name:  "Table",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			headersVal, err := interpreter.ArgArray(node, args, 0, "term.Table", "string")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			rowsVal, err := interpreter.ArgArray(node, args, 1, "term.Table", "string")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			headers := []string{}

			for _, h := range headersVal.Elements {
				str, ok := h.(interpreter.StringValue)
				if !ok {
					return interpreter.NilValue{}, interpreter.NewRuntimeError(node, "term.Table: first argument must be []string")
				}
				headers = append(headers, str.V)
			}

			rows := [][]string{}

			for _, r := range rowsVal.Elements {

				rowArr, ok := r.(interpreter.ArrayValue)
				if !ok {
					return interpreter.NilValue{}, fmt.Errorf("term.Table: second argument must be [][]string")
				}

				row := []string{}

				for _, cell := range rowArr.Elements {
					str, ok := cell.(interpreter.StringValue)
					if !ok {
						return interpreter.NilValue{}, fmt.Errorf("term.Table: second argument must be [][]string")
					}

					row = append(row, str.V)
				}

				rows = append(rows, row)
			}

			widths := make([]int, len(headers))

			for i, h := range headers {
				widths[i] = len(h)
			}

			for _, row := range rows {
				for i, cell := range row {
					if len(cell) > widths[i] {
						widths[i] = len(cell)
					}
				}
			}

			printSep := func() {
				fmt.Print("+")
				for _, w := range widths {
					fmt.Print(strings.Repeat("-", w+2) + "+")
				}
				fmt.Println()
			}

			printSep()

			fmt.Print("|")
			for i, h := range headers {
				fmt.Printf(" %-*s |", widths[i], h)
			}
			fmt.Println()

			printSep()

			for _, row := range rows {
				fmt.Print("|")
				for i, cell := range row {
					fmt.Printf(" %-*s |", widths[i], cell)
				}
				fmt.Println()
			}

			printSep()

			return interpreter.NilValue{}, nil
		},
	}, false)

	env.Define("Prompt", &interpreter.BuiltinFunc{
		Name:  "Prompt",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			ques, err := interpreter.ArgString(node, args, 0, "term.Prompt")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			fmt.Printf("%s: ", ques)

			var input string
			fmt.Scanln(&input)

			return interpreter.StringValue{V: input}, nil
		},
	}, false)

	env.Define("Confirm", &interpreter.BuiltinFunc{
		Name:  "Confirm",
		Arity: 1,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			ques, err := interpreter.ArgString(node, args, 0, "term.Confirm")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			fmt.Printf("%s (y/n): ", ques)

			var input string
			fmt.Scanln(&input)

			if input == "y" || input == "Y" || strings.ToLower(input) == "yes" {
				return interpreter.BoolValue{V: true}, nil
			}

			return interpreter.BoolValue{V: false}, nil
		},
	}, false)

	env.Define("Select", &interpreter.BuiltinFunc{
		Name:  "Select",
		Arity: 2,
		Fn: func(i *interpreter.Interpreter, node *parser.FuncCall, args []interpreter.Value) (interpreter.Value, error) {
			ques, err := interpreter.ArgString(node, args, 0, "term.Select")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			arr, err := interpreter.ArgArray(node, args, 1, "term.Select", "string")
			if err != nil {
				return interpreter.NilValue{}, err
			}

			oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
			if err != nil {
				return interpreter.NilValue{}, err
			}
			defer term.Restore(int(os.Stdin.Fd()), oldState)

			selected := 0

			draw := func() {
				fmt.Print("\033[2K\r")
				fmt.Println(ques)

				for idx, item := range arr.Elements {
					fmt.Print("\033[2K\r")

					str := item.(interpreter.StringValue).V

					if idx == selected {
						fmt.Printf("> %s\n", str)
					} else {
						fmt.Printf("  %s\n", str)
					}
				}
			}

			buf := make([]byte, 3)
			lines := len(arr.Elements) + 1

			draw()

			for {
				os.Stdin.Read(buf)

				if buf[0] == 27 && buf[1] == 91 {
					switch buf[2] {
					case 65:
						selected--
						if selected < 0 {
							selected = len(arr.Elements) - 1
						}

					case 66:
						selected++
						if selected >= len(arr.Elements) {
							selected = 0
						}
					}
				}

				if buf[0] == 13 {
					break
				}

				fmt.Printf("\033[%dA", lines)

				for range lines {
					fmt.Print("\033[2K\n")
				}

				fmt.Printf("\033[%dA", lines)

				draw()
			}

			return interpreter.IntValue{V: selected}, nil
		},
	}, false)

	module := interpreter.ModuleValue{
		Name: "term",
		Env:  env,
	}

	return module, nil
}
