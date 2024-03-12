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
	"time"
)

type Config struct {
	URI          string   `json:"URI"`
	SMSKey       string   `json:"SMS_KEY"`
	PhoneNumbers []string `json:"PHONE_NUMBERS"`
	IntervalMS   int      `json:"INTERVAL_MS"`
}

func loadConfig() Config {
	var config Config
	configFile, err := os.ReadFile("./config.json")

	if err != nil {
		log.Fatalf("Unable to read config file: %v", err)
	}

	json.Unmarshal(configFile, &config)

	return config
}

func sendText(number string, key string) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Disable SSL verification
		},
	}

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

	for _, number := range config.PhoneNumbers {
		go sendText(number, config.SMSKey)
	}

	time.Sleep(5 * time.Second)
	log.Fatal("Ending the process.")
}

func failure() {
	fmt.Println("no tikz found...")
}

func fetch(config Config) {
	resp, err := http.Get(config.URI)

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
