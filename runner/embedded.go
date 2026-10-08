package runner

import (
	"bytes"
	"os"
)

func TryRunEmbedded() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}

	data, err := os.ReadFile(exe)
	if err != nil {
		return false
	}

	startMarker := []byte("\n__AYLA_SCRIPT_START__\n")
	endMarker := []byte("\n__AYLA_SCRIPT_END__\n")

	start := bytes.LastIndex(data, startMarker)
	end := bytes.LastIndex(data, endMarker)

	if start == -1 || end == -1 || end <= start {
		return false
	}

	start += len(startMarker)
	runEmbedded(string(data[start:end]))
	return true
}
