package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sensibull/handler"
	"sensibull/utils"
	"syscall"
	"time"

	"sensibull/config"

	"github.com/gorilla/mux"
)

func main() {

	l := log.New(os.Stdout, "sensibull-api:", log.LstdFlags) // log object to write application logs
	var AppConfig = config.LoadConfig(l)
	var ticker = time.NewTicker(time.Second * time.Duration(AppConfig.StockRefreshSeconds))

	utils.DataSetup(&AppConfig, l)  // creates DB if not exists, checks for connection, runs migration to create tables and store initial data
	c := utils.DialTOWS(&AppConfig) // connect to websocket and returns connection object
	utils.StockRefresh(&AppConfig, c, l)
	go func() {
		// As per the stock_refresh_poll_seconds mentioned in config.json, this function will execute for every mentioned seconds
		for range ticker.C {
			l.Println("Refreshing Stocks...")
			c := utils.DialTOWS(&AppConfig)
			utils.StockRefresh(&AppConfig, c, l)
		}
	}()

	ph := handler.NewRequestHandler(l, &AppConfig)
	sm := mux.NewRouter()

	getUnderLyingRouter := sm.Methods(http.MethodGet).Subrouter()
	getUnderLyingRouter.HandleFunc("/api/underlying-prices", ph.GetUnderLyings)
	getUnderLyingRouter.HandleFunc("/api/derivative-prices/{underlying:[A-Z]+}", ph.GetDerivatives)

	s := &http.Server{
		Addr:         AppConfig.AppPort,
		Handler:      sm,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	l.Println("Server listening on", AppConfig.AppPort)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signals

	l.Println("Received terminate, graceful shutdown", sig)

	tc, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	l.Println("Timeout error occured")

	s.Shutdown(tc)

}
