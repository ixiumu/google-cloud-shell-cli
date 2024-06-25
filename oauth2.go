package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type StateResponse struct {
	Error struct {
		Code  int    `json:"code"`
		State string `json:"state"`
	} `json:"error"`
	State       string `json:"state"`
	SSHUsername string `json:"sshUsername"`
	SSHPort     int    `json:"sshPort"`
	SSHHost     string `json:"sshHost"`
}

const (
	listenAddress      = "127.0.0.1:8086"
)

func getSSHConfigLocal() (*SSHConfig, error) {
	log.Println("Getting Google Cloud Shell status...")

	for {
		status, err := status()
		if err != nil {
			return nil, err
		}

		// debug
		//log.Println(status)

		if status.Error.Code == 401 {
			log.Println("Unauthorized or token has expired, please reauthorize.")
		} else if status.State == "RUNNING" {
			log.Println("Got host information success, connecting...")

			return &SSHConfig{
				Host:     status.SSHHost,
				Port:     status.SSHPort,
				Username: status.SSHUsername,
			}, nil
		} else if status.State == "SUSPENDED" {
			log.Println("Waiting for Booting...")

			_, err = start()
			if err != nil {
				return nil, err
			}
		} else if status.State == "STARTING" {
			log.Println("Waiting for startup...")
		} else if status.State == "PENDING" {
			log.Println("Waiting for startup...")
		} else {
			log.Printf("State: %s", status)
		}

		time.Sleep(3 * time.Second)
	}
}

func getClient() (*http.Client, error) {
	// Load client credentials from a local file
	credentialsFile := filepath.Join(getSSHDirectory(), "gcs_credentials.json")
	credentials, err := readConfigFile(credentialsFile)
	if err != nil {
		log.Printf("Failed to read client credentials file: %v\n", err)
		return nil, err
	}

	// Parse client credentials
	config, err := google.ConfigFromJSON(credentials, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		log.Printf("Failed to parse client credentials: %v\n", err)
		return nil, err
	}

	// Load token from a local file
	tokenFile := filepath.Join(getSSHDirectory(), "gcs_token.json")
	token, err := loadTokenFromFile(tokenFile)
	if err != nil {
		// If token doesn't exist, initiate the authorization flow
		token, err = getTokenFromWeb(config)
		if err != nil {
			log.Printf("Failed to obtain token: %v\n", err)
			return nil, err
		}
		// Save the token to a file
		err = saveTokenToFile(tokenFile, token)
		if err != nil {
			log.Printf("Failed to save token to file: %v\n", err)
		}
	}

	// Create an HTTP client using the token
	client := config.Client(context.Background(), token)

	return client, nil
}

func status() (*StateResponse, error) {
	client, err := getClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.Get("https://cloudshell.googleapis.com/v1/users/me/environments/default")
	if err != nil {
		log.Printf("API request failed: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	var result *StateResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Fatal(err)
	}

	return result, err
}

func start() (bool, error) {
	client, err := getClient()
	if err != nil {
		return false, err
	}

	resp, err := client.Post("https://cloudshell.googleapis.com/v1/users/me/environments/default:start", "", nil)
	if err != nil {
		log.Printf("API request failed: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	return true, nil
}

func readConfigFile(file string) ([]byte, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		log.Printf("Failed to read file: %v\n", err)
		return nil, err
	}

	// log.Print(string(content))
	return content, nil
}

// Load token from a local file
func loadTokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	token := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(token)
	return token, err
}

// Save token to a local file
func saveTokenToFile(file string, token *oauth2.Token) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}

// Get token from the web flow
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	config.RedirectURL = fmt.Sprintf("http://%s/callback", listenAddress)
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	// log.Print(authURL)

	err := openBrowser(authURL)
	if err != nil {
		fmt.Printf("Please manually open the following URL in your browser to authorize the access:\n%v\n", authURL)
		fmt.Print("Enter the authorization code: ")
		var code string
		fmt.Scan(&code)

		return config.Exchange(context.Background(), code)
	} else {
		codeCh := make(chan string)
		go listenForCode(codeCh)
	
		fmt.Println("Waiting for authorization code...")
		code := <-codeCh

		return config.Exchange(context.Background(), code)
	}
}

func listenForCode(codeCh chan<- string) {
	srv := http.Server{Addr: listenAddress}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			codeCh <- code
			fmt.Fprint(w, "Authorization code received. You can now close this page.")

			go func() {
				time.Sleep(3 * time.Second)
				srv.Shutdown(context.Background())
			}()
		} else {
			fmt.Fprint(w, "Invalid authorization code.")
		}
	})

	srv.ListenAndServe()
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

func getSSHDirectory() (string) {
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
