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
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "ssh":
		ssh()
	case "state", "status":
		state()
	case "start":
		status, err := status()
		if err != nil {
			return
		}
		if status.State == "SUSPENDED" {
			_, err := start()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Start cloud shell success")
		} else if status.State == "RUNNING" {
			fmt.Println("Cloud shell is running")
		} else {
			fmt.Println("Unknow state:", status)
		}
	case "addPublicKey":
		if len(os.Args) < 3 {
			usage()
			return
		}

		key := os.Args[2]
		fmt.Println("Public key:", key)
		err := addPublicKey(key)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Public Key added successfully")
	default:
		usage()
	}
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("gcs ssh")
	fmt.Println("gcs status")
	fmt.Println("gcs start")
	fmt.Println("gcs addPublicKey <ssh-dss|ssh-rsa|ecdsa-sha2>")
}

func state() {
	status, err := status()
	if err != nil {
		return
	}
	if status.Error.Code == 401 {
		fmt.Println("Unauthorized or token has expired, please reauthorize.")
	} else if status.State == "RUNNING" {
		fmt.Println("State:", status.State)
		fmt.Println("Username:", status.SSHUsername)
		fmt.Println("Host:", status.SSHHost)
		fmt.Println("Port:", status.SSHPort)
		fmt.Println("PublicKeys:", status.PublicKeys)
	} else if status.State != "" {
		fmt.Println("State:", status.State)
	} else {
		fmt.Println("Unknow state:", status)
	}
}

func ssh() {
	sshConfig, err := getSSHConfigLocal()
	if err != nil {
		log.Println("Error get SSH config:", err)
		return
	}

	if sshConfig.Host != "" {
		arg := os.Args[2:]
		launchSSHCommand(sshConfig, arg)
	} else {
		log.Println("Error get SSH config")
	}
}

func launchSSHCommand(config *SSHConfig, arg []string) {
	var cmd *exec.Cmd

	parm := append([]string{"-p",
		fmt.Sprintf("%d", config.Port),
		fmt.Sprintf("%s@%s", config.Username, config.Host),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null"}, arg...)

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
