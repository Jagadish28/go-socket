package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sensibull/config"
	"sensibull/repo"

	"github.com/gorilla/mux"
)

type RequestHandler struct {
	l      *log.Logger
	config *config.AppConfig
}

func NewRequestHandler(l *log.Logger, config *config.AppConfig) *RequestHandler {
	return &RequestHandler{l, config}
}

func (p *RequestHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	p.l.Println("ServeHTTP is for Handler interface")
}

// GetUnderLyings get the list of stocks from the function in model package
func (p *RequestHandler) GetUnderLyings(rw http.ResponseWriter, r *http.Request) {
	results := repo.GetStocks(p.config, rw, r)
	jsonData, err := json.Marshal(results)
	if err != nil {
		p.l.Println("Failed to convert results to JSON:", err)
		return
	}

	rw.Write(jsonData)

}

// GetDerivatives get the list of derivatives from the function in model package
func (p *RequestHandler) GetDerivatives(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["underlying"]

	results := repo.GetDerivativesPrice(id, p.config, rw, r)
	jsonData, err := json.Marshal(results)
	if err != nil {
		p.l.Println("Failed to convert results to JSON:", err)
		return
	}

	rw.Write(jsonData)
}
