package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	addr := fmt.Sprintf("%s:%s", host, port)

	ATBBaseURL := os.Getenv("ATB_BASE_URL")
	if ATBBaseURL == "" {
		ATBBaseURL = "https://www.atbmarket.com"
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}

	app := Server{HttpClient: httpClient, ATBBaseURL: ATBBaseURL}
	log.Fatal(app.Run(context.Background(), addr))
}
