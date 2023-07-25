package model

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sensibull/config"
)

type Stock struct {
	Symbol          string  `json:"symbol"`
	Underlying      string  `json:"underlying"`
	Token           int     `json:"token"`
	Instrument_Type string  `json:"instrument_type"`
	Expiry          string  `json:"expiry"`
	Strike          float32 `json:"strike"`
	Price           float64 `json:"price"`
}
type Response struct {
	Success bool   `json:"success"`
	Payload Stocks `json:"payload"`
}
type WebSocketRequest struct {
	Msg_Command string `json:"msg_command"`
	Data_Type   string `json:"data_type"`
	Tokens      []int  `json:"tokens"`
}

type WebSocketResponse struct {
	Data_type string         `json:"data_type"`
	Payload   WebSocketPrice `json:"payload"`
}

type WebSocketPrice struct {
	Token int     `json:"token"`
	Price float64 `json:"price"`
}

type Stocks []*Stock

// GetUnderLyingsFromBroker fetches all the current underlyings from broker API
func GetUnderLyingsFromBroker(config *config.AppConfig) Stocks {
	response, err := http.Get(config.UnderlyingsApiUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var responseData Response
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		log.Fatal(err)
	}
	return responseData.Payload
}

// GetDerivativessFromToken fetches the list of derivatives from broker API
func GetDerivativessFromToken(config *config.AppConfig, id string) Stocks {
	response, err := http.Get(config.DerivativesApiUrl + id)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var responseData Response
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		log.Fatal(err)
	}
	return responseData.Payload
}
