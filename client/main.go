package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

const apiURL = "http://localhost:8080/cotacao"
const clientTimeout = 300 * time.Millisecond

type ResponseStruct struct {
	Bid string `json:"bid"`
}

func main() {
	clientCTX, clientCancel := context.WithTimeout(context.Background(), clientTimeout)
	defer clientCancel()

	req, err := http.NewRequestWithContext(clientCTX, http.MethodGet, apiURL, nil)
	if err != nil {
		LogError(clientCTX, "http.NewRequestWithContext", err)
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		LogError(clientCTX, "http.DefaultClient.Do", err)
		return
	}
	defer res.Body.Close()

	responseObject := ResponseStruct{}
	err = json.NewDecoder(res.Body).Decode(&responseObject)
	if err != nil {
		panic(err)
	}

	SaveFile(responseObject.Bid)
}

func SaveFile(bid string) {
	// Open the file in append mode, create it if it doesn't exist
	file, err := os.OpenFile("cotacao.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Write the new data on a new line
	_, err = file.WriteString("Dólar: " + bid + "\n")
	if err != nil {
		panic(err)
	}
}

func LogError(ctx context.Context, operation string, err error) {
	if ctx.Err() != nil {
		log.Printf("operation: %s, error: %s", operation, ctx.Err())
	} else {
		log.Printf("operation: %s, error: %s", operation, err)
	}
}
