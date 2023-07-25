package repo

import (
	"log"
	"net/http"
	"sensibull/config"
	"sensibull/utils"
)

// GetStocks fetches the list of stocks and its prices from the DB
func GetStocks(config *config.AppConfig, rw http.ResponseWriter, r *http.Request) []map[string]interface{} {
	db, err := utils.ConnectDB(config)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT symbol, underlying, instrument_type, strike, price, token from stocks where instrument_type = 'EQ'`)

	if err != nil {
		http.Error(rw, "unable to fetch data from DB", http.StatusBadRequest)
	}

	defer rows.Close()

	results := []map[string]interface{}{}
	columns, err := rows.Columns()
	if err != nil {
		http.Error(rw, "unable to fetch data from DB", http.StatusBadRequest)
	}
	for rows.Next() {
		rowData := make(map[string]interface{})
		columnPointers := make([]interface{}, len(columns))
		columnValues := make([]interface{}, len(columns))
		for i := range columns {
			columnPointers[i] = &columnValues[i]
		}
		err := rows.Scan(columnPointers...)
		if err != nil {
			http.Error(rw, "unable to fetch data from DB", http.StatusBadRequest)
		}
		for i, column := range columns {
			rowData[column] = columnValues[i]
		}
		results = append(results, rowData)
	}
	return results
}

// GetDerivativesPrice fetches the list of derivative stocks and its prices from the DB
func GetDerivativesPrice(id string, config *config.AppConfig, rw http.ResponseWriter, r *http.Request) []map[string]interface{} {
	db, err := utils.ConnectDB(config)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	query := "SELECT symbol, underlying, instrument_type, strike, price, token,expiry from stocks where underlying = $1 and instrument_type <> 'EQ' "
	rows, err := db.Query(query, id)

	if err != nil {
		http.Error(rw, "unable to fetch data from DB", http.StatusInternalServerError)
	}

	defer rows.Close()

	results := []map[string]interface{}{}
	columns, err := rows.Columns()
	if err != nil {
		http.Error(rw, "unable to fetch data from DB", http.StatusBadRequest)

	}
	for rows.Next() {
		rowData := make(map[string]interface{})
		columnPointers := make([]interface{}, len(columns))
		columnValues := make([]interface{}, len(columns))
		for i := range columns {
			columnPointers[i] = &columnValues[i]
		}
		err := rows.Scan(columnPointers...)
		if err != nil {
			http.Error(rw, "unable to fetch data from DB", http.StatusBadRequest)
		}
		for i, column := range columns {
			rowData[column] = columnValues[i]
		}
		results = append(results, rowData)
	}
	return results
}
