package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"example.com/username/bootdev-chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, _ := sql.Open("postgres", dbURL)
	dbQueries := database.New(db)
	fmt.Println(dbQueries)

	const port = "8080"

	mux := http.NewServeMux()

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	apiCnfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             dbQueries,
		platform:       os.Getenv("PLATFORM"),
		secret:         os.Getenv("SECRET"),
		polkaKey:       os.Getenv("POLKA_KEY"),
	}

	log.Printf("Serving on port: %s\n", port)
	/*Admin stuuf */
	mux.HandleFunc("GET /admin/metrics", apiCnfg.metrics)
	mux.HandleFunc("POST /admin/reset", apiCnfg.reset)

	/* API stuff */
	mux.HandleFunc("GET /api/healthz", healthCheck)
	/*mux.HandleFunc("POST /api/validate_chirp", validateChirp)*/
	mux.HandleFunc("POST /api/users", apiCnfg.createUser)
	mux.HandleFunc("PUT /api/users", apiCnfg.updateUser)

	mux.HandleFunc("POST /api/chirps", apiCnfg.createChirp)
	mux.HandleFunc("GET /api/chirps", apiCnfg.getChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCnfg.getChirp)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCnfg.deleteChirp)

	mux.HandleFunc("POST /api/login", apiCnfg.login)
	mux.HandleFunc("POST /api/refresh", apiCnfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", apiCnfg.handlerRevoke)
	mux.HandleFunc("POST /api/polka/webhooks", apiCnfg.upgradeUser)

	/* App stuff */
	mux.Handle("/app/", http.StripPrefix("/app", apiCnfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	log.Fatal(srv.ListenAndServe())
}
