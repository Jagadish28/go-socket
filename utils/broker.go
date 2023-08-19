package utils

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sensibull/config"
	"sensibull/model"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

var underLyingTokens = []int{} // variable to maintain current token list to subscribe / unsubscribe

// clearStocks will clear all current stocks in stocks table
func clearStocks(config *config.AppConfig, l *log.Logger) {
	underLyingTokens = underLyingTokens[:0]
	db, err := ConnectDB(config)
	if err != nil {
		l.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("TRUNCATE TABLE stocks")
	if err != nil {
		l.Fatal(err)
	}
	l.Printf("Stocks refreshed successfully at %s", time.Now())
}

// LoadInitialStocks hits the broker api and save it to DB and with the token list It'll trigger fetch derivatives
func LoadInitialStocks(conn *websocket.Conn, config *config.AppConfig, stopCh chan []int, l *log.Logger, done chan<- string) {
	clearStocks(config, l) // clears current stocks
	db, err := sql.Open(config.DatabaseHost, config.DataBaseConnectionString)
	if err != nil {
		l.Fatal(err)
	}
	defer db.Close()

	stocks := model.GetUnderLyingsFromBroker(config)
	for _, st := range stocks {
		stmt, err := db.Prepare("INSERT INTO stocks (symbol, underlying, token, instrument_type, expiry, strike,price) VALUES ($1, $2, $3, $4, $5, $6, $7)")
		if err != nil {
			l.Fatal(err)
		}
		defer stmt.Close()

		newStock := model.Stock{
			Symbol:          st.Symbol,
			Underlying:      st.Underlying,
			Token:           st.Token,
			Instrument_Type: st.Instrument_Type,
			Expiry:          st.Expiry,
			Strike:          st.Strike,
			Price:           st.Price,
		}
		_, err = stmt.Exec(newStock.Symbol, newStock.Underlying, newStock.Token, newStock.Instrument_Type, newStock.Expiry, newStock.Strike, newStock.Price)
		if err != nil {
			log.Fatal(err)
		}
		underLyingTokens = append(underLyingTokens, newStock.Token)

	}
	loadInitialDerivatives(conn, config, underLyingTokens, stopCh, l, done)

}

// ConnectDB connects to the DB and will return connection object
func ConnectDB(config *config.AppConfig) (*sql.DB, error) {
	db, err := sql.Open(config.DatabaseHost, config.DataBaseConnectionString)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// loadInitialDerivatives will load the list of derivatives from Broker API and sent token list for subscription
func loadInitialDerivatives(conn *websocket.Conn, config *config.AppConfig, tokens []int, stopCh chan []int, l *log.Logger, done chan<- string) {
	db, err := sql.Open(config.DatabaseHost, config.DataBaseConnectionString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	for _, token := range tokens {
		derivatives := model.GetDerivativessFromToken(config, strconv.Itoa(token))
		for _, dv := range derivatives {
			stmt, err := db.Prepare("INSERT INTO stocks (symbol, underlying, token, instrument_type, expiry, strike, price) VALUES ($1, $2, $3, $4, $5, $6, $7)")
			if err != nil {
				log.Fatal(err)
			}
			defer stmt.Close()

			newStock := model.Stock{
				Symbol:          dv.Symbol,
				Underlying:      dv.Underlying,
				Token:           dv.Token,
				Instrument_Type: dv.Instrument_Type,
				Expiry:          dv.Expiry,
				Strike:          dv.Strike,
				Price:           dv.Price,
			}
			_, err = stmt.Exec(newStock.Symbol, newStock.Underlying, newStock.Token, newStock.Instrument_Type, newStock.Expiry, newStock.Strike, newStock.Price)

			if err != nil {
				log.Fatal(err)
			}
			underLyingTokens = append(underLyingTokens, newStock.Token)

		}
	}
	// sends message to the channel with list of token
	getPrice(config, conn, underLyingTokens, l, done)
	// stopCh <- underLyingTokens
}

func saveStockPrice(config *config.AppConfig, token int, price float64, l *log.Logger) error {
	db, err := ConnectDB(config)
	if err != nil {
		l.Fatal(err)
		return err
	}
	defer db.Close()
	query := "UPDATE stocks SET price = $1 WHERE token = $2"
	result, err := db.Exec(query, price, token)
	if err != nil {
		l.Fatal(err)
		return err
	}

	// Check the number of rows affected by the update
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		l.Fatal(err)
		return err
	}

	l.Printf("Stock Price Update successfully. Rows affected: %d\n", rowsAffected)
	return nil
}

func SubscribeToWebsocket(conf *config.AppConfig, c *websocket.Conn, tokens []int, l *log.Logger) error {

	interval := time.Duration(conf.StockSubscribeSeconds) * time.Second // Interval for polling
	isUnSubscribed := false

	// Write data to the WebSocket connection every minute
	request := model.WebSocketRequest{
		Msg_Command: "subscribe",
		Data_Type:   "quote",
		Tokens:      tokens,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		l.Println("Error encoding JSON request:", err)
		return err
	}

	err = c.WriteMessage(websocket.TextMessage, requestJSON)
	if err != nil {
		l.Println("WebSocket write error:", err)
	} else {
		l.Println("Subscribe Message sent successfully!---->", requestJSON)
	}

	// Based on the configured unsubscription polling time, we are un-subscribing the current list of tokens
	DurationOfTime := time.Duration(conf.StockUnSubscribeSeconds) * time.Second
	Timer1 := time.AfterFunc(DurationOfTime, func() {
		isUnSubscribed = true
		err = UnSubscribeToWebsocket(c, conf, tokens, l)
		if err != nil {
			l.Println("WebSocket connection error:", err)
		}
	})

	defer Timer1.Stop()

	for {

		// once un-subscribed, breaking this infinite for loop
		if isUnSubscribed {
			break
		}
		// Read the response from the WebSocket server
		_, response, err := c.ReadMessage()
		if err != nil {
			l.Println("WebSocket read error:", err)
		}

		// Process the received response
		var websocketResponse model.WebSocketResponse

		err = json.Unmarshal([]byte(response), &websocketResponse)

		if err != nil {
			l.Println("Error parsing Websocket response:", err)
		} else {
			l.Println("Message Received!", websocketResponse)

		}

		// saveStockPrice writes every response from websocket
		err = saveStockPrice(conf, websocketResponse.Payload.Token, websocketResponse.Payload.Price, l)
		if err != nil {
			l.Println("Error while saving stock price:", err)
		}
		time.Sleep(interval)
	}
	return nil

}

// UnSubscribeToWebsocket will unsubscribe the current tokens and close the socket connection
func UnSubscribeToWebsocket(conn *websocket.Conn, config *config.AppConfig, tokens []int, l *log.Logger) error {
	l.Println("Unsubscribe Triggered.....")
	request := model.WebSocketRequest{
		Msg_Command: "unsubscribe",
		Data_Type:   "quote",
		Tokens:      tokens,
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		l.Println("Error encoding JSON request:", err)
		return err
	}
	err = conn.WriteMessage(websocket.TextMessage, requestJSON)
	if err != nil {
		l.Println("WebSocket write error:", err)

	} else {
		l.Println("Unsubscribe Message sent successfully!", string(requestJSON))
	}
	defer conn.Close()
	return nil

}

// DialTOWS will establist a new soscket connection and returns the connection object
func DialTOWS(config *config.AppConfig) *websocket.Conn {
	c, _, err := websocket.DefaultDialer.Dial(config.WebSocketUrl, nil)
	if err != nil {
		log.Fatal("WebSocket dial error:", err)
	}
	return c
}

func getPrice(config *config.AppConfig, conn *websocket.Conn, tokens []int, l *log.Logger, done chan<- string) error {

	for token := range tokens {

		fmt.Println("token---->", tokens[token])
		request := model.WebSocketRequest{
			Msg_Command: "subscribe",
			Data_Type:   "quote",
			Tokens:      []int{tokens[token]},
		}
		fmt.Println("request---->", request)

		requestJSON, err := json.Marshal(request)
		if err != nil {
			l.Println("Error encoding JSON request:", err)
			return err
		}

		err = conn.WriteMessage(websocket.TextMessage, requestJSON)
		if err != nil {
			l.Println("WebSocket write error:", err)
		} else {
			l.Println("Subscribe Message sent successfully!---->", string(requestJSON))
		}

		ReadMessage(conn, tokens[token], config, l)

	}
	defer conn.Close()
	done <- "initial stocks fetched"

	return nil

}

func ReadMessage(conn *websocket.Conn, token int, config *config.AppConfig, l *log.Logger) {
	_, response, err := conn.ReadMessage()
	if err != nil {
		l.Println("WebSocket read error:", err)
	}

	// Process the received response
	var websocketResponse model.WebSocketResponse
	var websocketPingResponse model.WebSocketPingResponse

	err = json.Unmarshal([]byte(response), &websocketResponse)
	fmt.Println("response--->", string(response))

	if err != nil {
		err = json.Unmarshal([]byte(response), &websocketPingResponse)
		if err != nil {
			l.Println("Error parsing Websocket response:", err)
		}
		if websocketPingResponse.Data_type == "ping" {
			l.Println("ping message received")
		}

		// when the ping message received call the same method again for actual subscription price
		ReadMessage(conn, token, config, l)

	} else {
		l.Println("Message Received!", websocketResponse)

		err := saveStockPrice(config, websocketResponse.Payload.Token, websocketResponse.Payload.Price, l)
		if err != nil {
			l.Println("Error while saving stock price:", err)
		}
		err = unSubscribePrices(conn, token, l)
		if err != nil {
			l.Println("Error while unsubscribe:", err)
		}
		// readResponseAndSave(conn, config, websocketResponse, l, tokens[token])

	}
}

// func readResponseAndSave(conn *websocket.Conn, config *config.AppConfig, websocketResponse model.WebSocketResponse, l *log.Logger, token int) {
// 	l.Println("Message Received!", websocketResponse)

// 	err := saveStockPrice(config, websocketResponse.Payload.Token, websocketResponse.Payload.Price, l)
// 	if err != nil {
// 		l.Println("Error while saving stock price:", err)
// 	}
// 	err = unSubscribePrices(conn, token, l)
// 	if err != nil {
// 		l.Println("Error while unsubscribe:", err)
// 	}
// }

func unSubscribePrices(conn *websocket.Conn, token int, l *log.Logger) error {
	l.Println("Unsubscribe Triggered.....")
	request := model.WebSocketRequest{
		Msg_Command: "unsubscribe",
		Data_Type:   "quote",
		Tokens:      []int{token},
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		l.Println("Error encoding JSON request:", err)
		return err
	}
	err = conn.WriteMessage(websocket.TextMessage, requestJSON)
	if err != nil {
		l.Println("WebSocket write error:", err)

	} else {
		l.Println("Unsubscribe Message sent successfully!", string(requestJSON))
	}
	// defer conn.Close()
	return nil

}

func GetDerivativeTokens(config *config.AppConfig, l *log.Logger) (error, []int) {
	db, err := ConnectDB(config)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	query := "SELECT ARRAY(SELECT token from stocks where instrument_type <> 'EQ') AS tokens; "
	rows, err := db.Query(query)

	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()
	fmt.Println("rows-->", rows)

	return nil, nil
}
