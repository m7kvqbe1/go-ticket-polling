package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/joho/godotenv"
)

var (
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Ignore SSL errors
		},
	}
	wg   sync.WaitGroup
	done = make(chan struct{})
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
}

func sendText(number string) {
	wg.Add(1)
	defer wg.Done()

	key := os.Getenv("SMS_KEY")
	message := "BUY DI TIKITZ!!!"

	reqJSON, err := json.Marshal(map[string]string{
		"phone":   number,
		"message": message,
		"key":     key,
	})
	if err != nil {
		log.Println("Error encoding request body:", err)
		return
	}

	req, err := http.NewRequest(
		"POST",
		"https://textbelt.com/text",
		strings.NewReader(string(reqJSON)),
	)
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
		log.Printf("SMS sent: %s\n", number)
	} else {
		log.Printf("Failed to send SMS to %s\n", number)
	}
}

func fetch(ctx context.Context) {
	c := colly.NewCollector(
		// Spoof user agent to avoid bot detection
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/91.0.4472.124 Safari/537.36"),
	)

	c.WithTransport(&http.Transport{
		DialContext: (&net.Dialer{}).DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9")
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

	err := c.Visit(os.Getenv("URL"))
	if err != nil {
		log.Println("Error visiting:", err)
	}

	if err := ctx.Err(); err != nil {
		log.Println("Operation canceled after visit:", err)
	}
}

func success() {
	log.Println("BUY DI TIKITZ!!!")

	phoneNumbers := strings.Split(os.Getenv("PHONE_NUMBERS"), ",")
	for _, number := range phoneNumbers {
		go sendText(number)
	}

	close(done)
}

func failure() {
	log.Println("no tikz found...")
}

func scrapeLoop(ctx context.Context) {
	intervalMS, err := strconv.Atoi(os.Getenv("INTERVAL_MS"))
	if err != nil {
		log.Fatalf("Error parsing INTERVAL_MS: %v", err)
	}

	ticker := time.NewTicker(time.Duration(intervalMS) * time.Millisecond)
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
	log.Println("Polling for da tikz...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go scrapeLoop(ctx)

	select {
	case <-signals:
		log.Println("Received an interrupt, stopping service...")
	case <-done:
		log.Println("Success! Terminating gracefully...")
	}

	wg.Wait()
	log.Println("Shutting down gracefully")
}
