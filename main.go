package main

import (
	"log"
	"net/http"
)

func healthCheck(w http.ResponseWriter, req *http.Request){
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}	
func main() {
	const port = "8080"

	mux := http.NewServeMux()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)

	mux.HandleFunc("/healthz", healthCheck)
	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	log.Fatal(srv.ListenAndServe())
}
