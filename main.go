package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/trugamr/borque/internal/repository"

	_ "modernc.org/sqlite"
)

//go:embed db/schema.sql
var ddl string

func main() {
	ctx := context.Background()

	// Connect to the database
	db, err := sql.Open("sqlite", "./borque.db")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Create the tables
	_, err = db.ExecContext(ctx, ddl)
	if err != nil {
		log.Fatalf("Error creating tables: %v", err)
	}

	// Create a new Querier
	queries := repository.New(db)

	// Start the server
	mux := http.NewServeMux()
	indexHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		indexRoute(w, r, queries)
	})
	mux.Handle("/", apiKeyAuthMiddleware(indexHandler))

	addr := net.TCPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: 8080,
	}
	log.Printf("Server started! Listening at http://%s", addr.String())
	err = http.ListenAndServe(addr.String(), mux)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func apiKeyAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := os.Getenv("API_KEY")
		if apiKey != "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Printf("Missing Authorization header")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				log.Printf("Invalid Authorization header format")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if parts[1] != apiKey {
				log.Printf("Invalid API key")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func indexRoute(w http.ResponseWriter, r *http.Request, queries *repository.Queries) {
	log.Printf("Incoming request: path=%s, query=%s", r.URL.Path, r.URL.Query().Encode())

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Marshal headers to JSON
	headers, err := json.Marshal(r.Header)
	if err != nil {
		log.Printf("Error marshalling headers to JSON: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Insert request into the database
	err = queries.InsertRequest(r.Context(), repository.InsertRequestParams{
		Method:  sql.NullString{String: r.Method, Valid: r.Method != ""},
		Path:    r.URL.Path,
		Headers: string(headers),
		Query:   r.URL.Query().Encode(),
		Body:    string(body),
	})
	if err != nil {
		log.Printf("Error inserting request into database: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "OK!")
}
