package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gocolly/colly/v2"
)

var (
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	config Config
	wg     sync.WaitGroup
)

type Config struct {
	URL          string   `json:"URL"`
	SMSKey       string   `json:"SMS_KEY"`
	PhoneNumbers []string `json:"PHONE_NUMBERS"`
	IntervalMS   int      `json:"INTERVAL_MS"`
}

func loadConfig() {
	configFile, err := os.ReadFile("./config.json")
	if err != nil {
		log.Fatalf("Unable to read config file: %v", err)
	}

	err = json.Unmarshal(configFile, &config)
	if err != nil {
		log.Fatalf("Unable to parse config file: %v", err)
	}
}

func sendText(number, key string) {
	wg.Add(1)
	defer wg.Done()

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

func fetch(ctx context.Context) {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	c.WithTransport(&http.Transport{
		DialContext: (&net.Dialer{}).DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Referer", "https://www.google.com/")
		log.Println("Visiting", r.URL.String())
	})

	foundBuynow := false

	c.OnHTML("a[id='buynow']", func(e *colly.HTMLElement) {
		foundBuynow = true
		success()
	})

	c.OnScraped(func(r *colly.Response) {
		if !foundBuynow {
			failure()
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error fetching %s: %v\n", r.Request.URL, err)
	})

	if err := ctx.Err(); err != nil {
		log.Println("Operation canceled:", err)
		return
	}

	err := c.Visit(config.URL)
	if err != nil {
		log.Println("Error visiting:", err)
	}

	if err := ctx.Err(); err != nil {
		log.Println("Operation canceled after visit:", err)
	}
}

func success() {
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

func scrapeLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(config.IntervalMS) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				fetch(ctx)
			}()
		}
	}
}

func main() {
	fmt.Println("Polling for da tikz...")
	loadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signals
		fmt.Println("\nReceived an interrupt, stopping services...")
		cancel()
	}()

	scrapeLoop(ctx)

	wg.Wait()
	fmt.Println("Shutting down gracefully")
}
