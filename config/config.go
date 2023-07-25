package config

import (
	"encoding/json"
	"log"
	"os"
)

type AppConfig struct {
	DatabaseHost                  string `json:"database_host"`
	DatabasePort                  int    `json:"database_port"`
	DatabaseName                  string `json:"database_name"`
	APIKey                        string `json:"api_key"`
	AppPort                       string `json:"app_port"`
	DatabaseUserName              string `json:"database_username"`
	DatabasePassword              string `json:"database_password"`
	DataBaseConnectionString      string `json:"database_connection_string"`
	DataBaseCheckConnectionString string `json:"database_check_connection_string"`
	UnderlyingsApiUrl             string `json:"underlyings_api_url"`
	DerivativesApiUrl             string `json:"derivatives_api_url"`
	WebSocketUrl                  string `json:"web_socket_url"`
	StockRefreshSeconds           int    `json:"stock_refresh_poll_seconds"`
	StockSubscribeSeconds         int    `json:"socket_subscription_poll_seconds"`
	StockUnSubscribeSeconds       int    `json:"socket_un_subscription_poll_seconds"`
}

func LoadConfig(l *log.Logger) AppConfig {
	configFile := "./config.json"
	file, err := os.Open(configFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var config AppConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		panic(err)
	}

	l.Println("Config loaded Successfully")
	return config
}
