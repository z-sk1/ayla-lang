package runner

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func Install() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		return
	}

	libDir := filepath.Join(home, ".ayla", "lib")
	os.MkdirAll(libDir, 0755)

	url := normalizeGitHubURL(os.Args[2])

	fmt.Println("downloading:", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("failed to download module")
		return
	}

	fileName := filepath.Base(url)

	if !strings.HasSuffix(fileName, ".ayla") && !strings.HasSuffix(fileName, ".ayl") {
		fileName += ".ayla"
	}

	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	moduleDir := filepath.Join(libDir, name)

	os.MkdirAll(moduleDir, 0755)

	dest := filepath.Join(moduleDir, fileName)

	out, err := os.Create(dest)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("installed module:", fileName)
}
