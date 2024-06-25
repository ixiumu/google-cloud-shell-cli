package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

func ReadFile(file string) ([]byte, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		log.Printf("Failed to read file: %v\n", err)
		return nil, err
	}

	// log.Print(string(content))
	return content, nil
}

func getSSHDirectory() string {
	currentUser, err := user.Current()
	if err != nil {
		return ""
	}

	sshDir := filepath.Join(currentUser.HomeDir, ".ssh")

	_, err = os.Stat(sshDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(sshDir, 0700)
		if err != nil {
			return ""
		}
	} else if err != nil {
		return ""
	}

	return sshDir
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", strings.Replace(url, "&", "^&", -1))
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
