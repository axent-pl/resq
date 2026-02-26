// main.go (skrótowo)
package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", listReportsHandler)
	mux.HandleFunc("/reports", listReportsHandler)
	mux.HandleFunc("/reports/new", newReportHandler)
	mux.HandleFunc("/reports/view/{id}", readReportHandler)
	mux.HandleFunc("/reports/edit/{id}", editReportHandler)
	mux.HandleFunc("/reports/version/{id}", newVersionHandler)
	log.Fatal(http.ListenAndServe(":1234", mux))
}
