package main

import (
	"fmt"
	"log"

	"os"
	"os/exec"
	"runtime"
)

type SSHConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
}

func main() {
	sshConfig, err := getSSHConfig()
	if err != nil {
		log.Println("Error get SSH config:", err)
		return
	}

	if sshConfig.Host != "" {
		launchSSHCommand(sshConfig)
	} else {
		log.Println("Error get SSH config")
	}
}

func launchSSHCommand(config SSHConfig) {
	var cmd *exec.Cmd

	var parm = []string{"-p", fmt.Sprintf("%d", config.Port), fmt.Sprintf("%s@%s", config.Username, config.Host)}
	parm = append(parm, os.Args[1:]...)

	if runtime.GOOS == "windows" {
		cmd = exec.Command("ssh", parm...)
	} else {
		cmd = exec.Command("ssh", parm...)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Println("Error launching SSH command:", err)
	}
}
