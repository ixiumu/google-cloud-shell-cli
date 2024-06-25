package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type StateResponse struct {
	Error struct {
		Code  int    `json:"code"`
		State string `json:"state"`
	} `json:"error"`
	State       string   `json:"state"`
	SSHUsername string   `json:"sshUsername"`
	SSHPort     int      `json:"sshPort"`
	SSHHost     string   `json:"sshHost"`
	PublicKeys  []string `json:"publicKeys"`
}

const (
	listenAddress = "127.0.0.1:8086"
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
			log.Println("Waiting for booting...")

			_, err = start()
			if err != nil {
				return nil, err
			}
		} else if status.State == "STARTING" {
			log.Println("Waiting for starting...")
		} else if status.State == "PENDING" {
			log.Println("Waiting for pending...")
		} else {
			log.Println("Status:", status)
		}

		time.Sleep(3 * time.Second)
	}
}

func getClient() (*http.Client, error) {
	credentialsFile := filepath.Join(getSSHDirectory(), "gcs_credentials.json")
	credentials, err := ReadFile(credentialsFile)
	if err != nil {
		log.Printf("Failed to read client credentials file: %v\n", err)
		return nil, err
	}

	config, err := google.ConfigFromJSON(credentials, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		log.Printf("Failed to parse client credentials: %v\n", err)
		return nil, err
	}

	tokenFile := filepath.Join(getSSHDirectory(), "gcs_token.json")
	token, err := loadTokenFromFile(tokenFile)
	if err != nil {
		token, err = getTokenFromWeb(config)
		if err != nil {
			log.Printf("Failed to obtain token: %v\n", err)
			return nil, err
		}
		err = saveTokenToFile(tokenFile, token)
		if err != nil {
			log.Printf("Failed to save token to file: %v\n", err)
		}
	}

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

func addPublicKey(key string) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	requestBody := map[string]string{"key": key}
	jsonBody, _ := json.Marshal(requestBody)

	resp, err := client.Post("https://cloudshell.googleapis.com/v1/users/me/environments/default:addPublicKey", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("API request failed: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response body: %v\n", err)
			return err
		}

		fmt.Println("Response Body:", string(body))

		return fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	return nil
}

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

func saveTokenToFile(file string, token *oauth2.Token) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}

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
