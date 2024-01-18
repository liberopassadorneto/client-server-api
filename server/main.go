package main

import (
	"context"
	"database/sql"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const quoteURL = "https://economia.awesomeapi.com.br/json/last/USD-BRL"

const dbOperationTimeout = 100 * time.Millisecond // I need to set 100ms timeout because my CPU is slow
const apiTimeout = 200 * time.Millisecond

type App struct {
	DB *sql.DB
}

type ExchangeRate struct {
	ID         int    `json:"id"`
	Code       string `json:"code"`
	CodeIn     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

type ResponseData struct {
	USDBRL ExchangeRate `json:"USDBRL"`
}

type ResponsePresenter struct {
	Bid string `json:"bid"`
}

func main() {
	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = CreateExchangeRatesTable(db)
	if err != nil {
		panic(err)
	}

	app := &App{DB: db}
	http.HandleFunc("/cotacao", app.USDBRLExchangeRateHandler)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

func (app *App) USDBRLExchangeRateHandler(w http.ResponseWriter, r *http.Request) {
	dbCTX, dbCancel := context.WithTimeout(context.Background(), dbOperationTimeout)
	defer dbCancel()

	apiCTX, apiCancel := context.WithTimeout(context.Background(), apiTimeout)
	defer apiCancel()

	rate, err := FetchUSDBRLExchangeRate(apiCTX)
	if err != nil {
		LogError(dbCTX, "FetchUSDBRLExchangeRate", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	responsePresenter := ResponsePresenter{
		Bid: rate.USDBRL.Bid,
	}
	err = json.NewEncoder(w).Encode(responsePresenter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = InsertUSDBRLExchangeRate(dbCTX, app.DB, &rate.USDBRL)
	if err != nil {
		LogError(dbCTX, "InsertUSDBRLExchangeRate", err)
		return
	}
}

func LogError(ctx context.Context, operation string, err error) {
	if ctx.Err() != nil {
		log.Printf("operation: %s, error: %s", operation, ctx.Err())
	} else {
		log.Printf("operation: %s, error: %s", operation, err)
	}
}

func FetchUSDBRLExchangeRate(ctx context.Context) (*ResponseData, error) {
	// Create a new HTTP request with the context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, quoteURL, nil)
	if err != nil {
		return nil, err
	}

	// Execute the HTTP request
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON response into the ResponseData struct
	var responseData ResponseData
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return nil, err
	}

	return &responseData, nil
}

func CreateExchangeRatesTable(db *sql.DB) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS exchange_rates (
	    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL,
		code_in TEXT NOT NULL,
		name TEXT NOT NULL,
		high TEXT NOT NULL,
		low TEXT NOT NULL,
		var_bid TEXT NOT NULL,
		pct_change TEXT NOT NULL,
		bid TEXT NOT NULL,
		ask TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		create_date TEXT NOT NULL
	);
	`
	_, err := db.Exec(createTableSQL)
	if err != nil {
		return err
	}
	return nil
}

func InsertUSDBRLExchangeRate(ctx context.Context, db *sql.DB, exchangeRate *ExchangeRate) error {
	// SQL statement to insert the exchange rate data into the database
	insertSQL := `
    INSERT INTO exchange_rates (
        code,
        code_in,
        name,
        high,
        low,
        var_bid,
        pct_change,
        bid,
        ask,
        timestamp,
        create_date
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
    `

	// Prepare the SQL statement with the provided context
	stmt, err := db.PrepareContext(ctx, insertSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Execute the SQL statement with the provided context
	_, err = stmt.ExecContext(ctx,
		exchangeRate.Code,
		exchangeRate.CodeIn,
		exchangeRate.Name,
		exchangeRate.High,
		exchangeRate.Low,
		exchangeRate.VarBid,
		exchangeRate.PctChange,
		exchangeRate.Bid,
		exchangeRate.Ask,
		exchangeRate.Timestamp,
		exchangeRate.CreateDate)

	if err != nil {
		return err
	}

	return nil
}
