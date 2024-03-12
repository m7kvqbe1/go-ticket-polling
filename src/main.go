package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

type Config struct {
	URL          string   `json:"URL"`
	SMSKey       string   `json:"SMS_KEY"`
	PhoneNumbers []string `json:"PHONE_NUMBERS"`
	IntervalMS   int      `json:"INTERVAL_MS"`
}

var (
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
)

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

func sendText(number, key string) {
	message := `BUY DI TIKITZ!!!`
	reqBody := strings.NewReader(fmt.Sprintf(`{"phone": "%s", "message": "%s", "key": "%s"}`, number, message, key))

	req, err := http.NewRequest("POST", "https://textbelt.com/text", reqBody)

	if err != nil {
		log.Println("Error creating request:", err)
		return
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := httpClient.Do(req)

	if err != nil {
		log.Println("Error sending SMS:", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("SMS sent: %s\n", number)
	} else {
		log.Printf("Failed to send SMS to %s\n", number)
	}
}

func fetch(config Config) {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Referer", "https://www.google.com/")
		log.Println("Visiting", r.URL.String())
	})

	foundBuynow := false

	c.OnHTML("a[id='buynow']", func(e *colly.HTMLElement) {
		foundBuynow = true
		success(config)
	})

	c.OnScraped(func(r *colly.Response) {
		if !foundBuynow {
			failure()
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error fetching %s: %v\n", r.Request.URL, err)
	})

	err := c.Visit(config.URL)
	if err != nil {
		log.Println("Error visiting:", err)
	}
}

func success(config Config) {
	fmt.Println("BUY DI TIKITZ!!!")

	for _, number := range config.PhoneNumbers {
		go sendText(number, config.SMSKey)
	}

	time.Sleep(5 * time.Second)
	log.Fatal("Ending the process")
}

func failure() {
	fmt.Println("no tikz found...")
}

func scrapeLoop(config Config) {
	ticker := time.NewTicker(time.Duration(config.IntervalMS) * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		fetch(config)
	}
}

func main() {
	fmt.Println("Polling for da tikz...")
	config := loadConfig()
	scrapeLoop(config)
}
