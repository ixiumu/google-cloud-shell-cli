package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

type Response struct {
	Error struct {
		Code  int    `json:"code"`
		State string `json:"state"`
	} `json:"error"`
	State       string `json:"state"`
	SSHUsername string `json:"sshUsername"`
	SSHPort     int    `json:"sshPort"`
	SSHHost     string `json:"sshHost"`
}

func request(url string) (Response, error) {
	// response, err := http.Get(url)
	response, err := http.Post(url, "application/json", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	var result Response
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		log.Fatal(err)
	}

	return result, err
}

func getSSHConfig() (SSHConfig, error) {
	log.Println("Getting Google Cloud Shell status...")

	connectUrl := "https://google-cloud-shell.add.workers.dev/connect"
	result, err := request(connectUrl)
	if err != nil {
		log.Fatal(err)
	}

	if result.Error.Code == 401 {
		log.Println("Unauthorized or token has expired, please reauthorize.")

		authUrl := "https://google-cloud-shell.add.workers.dev/auth"
		err := openBrowser(authUrl)
		if err != nil {
			log.Fatal(err)
			log.Printf("Please manually open the website %s and authorize using your Google Cloud account.", authUrl)
		}

		for {
			time.Sleep(3 * time.Second)

			result, err = request(connectUrl)
			// log.Println(result)
			if err != nil {
				log.Fatal(err)
			} else if result.Error.Code == 401 {
				log.Println("Waiting for authorization...")
			} else if result.Error.Code != 0 {
				log.Printf("Error Code: %v", result.Error.Code)
			} else if result.State == "STARTING" {
				log.Println("Waiting for startup...")
				time.Sleep(1 * time.Second)
				break
			} else if result.State == "RUNNING" {
				break
			} else if result.State != "" {
				log.Println("State: ", result.State)
			} else {
				log.Println("Waiting...")
			}
		}
	}

	if result.State == "RUNNING" {
		log.Println("Got host information success, connecting...")

		return SSHConfig{
			Host:     result.SSHHost,
			Port:     result.SSHPort,
			Username: result.SSHUsername,
		}, nil
	} else {
		log.Println(result)
		return SSHConfig{}, errors.New("Unknown")
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
