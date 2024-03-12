package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Config struct {
	URI          string   `json:"URI"`
	SMSKey       string   `json:"SMS_KEY"`
	PhoneNumbers []string `json:"PHONE_NUMBERS"`
	IntervalMS   int      `json:"INTERVAL_MS"`
}

var client = &http.Client{
	Timeout: time.Second * 30,
	Transport: &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: false}, // It's recommended to not disable SSL verification
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

func loadConfig() Config {
	var config Config
	configFile, err := os.ReadFile("./config.json")

	if err != nil {
		log.Fatalf("Unable to read config file: %v", err)
	}

	err = json.Unmarshal(configFile, &config)

	if err != nil {
		log.Fatalf("Unable to parse config file: %v", err)
	}

	return config
}

func sendText(number string, key string) {
	message := `BUY DI TIKITZ!!!`
	reqBody := strings.NewReader(fmt.Sprintf(`{"phone": "%s", "message": "%s", "key": "%s"}`, number, message, key))

	req, err := http.NewRequest("POST", "https://textbelt.com/text", reqBody)

	if err != nil {
		log.Println(err)
		return
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		log.Println(err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("SMS sent: %s\n", number)
	} else {
		log.Printf("Failed to send SMS to %s\n", number)
	}
}

func parsePage(html string, config Config) {
	match := `<a id="buynow" href="#" title="Buy tickets">Buy tickets</a>`

	if strings.Contains(html, match) {
		success(config)
	} else {
		failure()
	}
}

func success(config Config) {
	fmt.Println("BUY DI TIKITZ!!!")

	var wg sync.WaitGroup

	for _, number := range config.PhoneNumbers {
		wg.Add(1)
		go func(num string) {
			defer wg.Done()
			sendText(num, config.SMSKey)
		}(number)
	}

	wg.Wait() // Wait for all SMS sending goroutines to finish
	log.Fatal("Ending the process.")
}

func failure() {
	fmt.Println("no tikz found...")
}

func fetch(config Config) {
	resp, err := client.Get(config.URI)

	if err != nil {
		log.Println(err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)

		if err != nil {
			log.Println(err)
			return
		}

		bodyString := string(bodyBytes)
		parsePage(bodyString, config)
	}
}

func scrapeLoop(config Config) {
	for {
		go fetch(config)
		time.Sleep(time.Duration(config.IntervalMS) * time.Millisecond)
	}
}

func main() {
	fmt.Println("Polling for da tikz...")
	config := loadConfig()
	scrapeLoop(config)
}
