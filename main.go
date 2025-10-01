// fixed version
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"runtime/debug"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database struct {
		Host     string `yaml:"host"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"database"`
	API struct {
		Key string `yaml:"key"`
	} `yaml:"api"`
}

var cfg Config

func loadConfig() {
	// Load config.yaml if it exists (it should not contain secrets in the repo).
	if _, err := os.Stat("config.yaml"); os.IsNotExist(err) {
		return
	}
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Printf("failed to read config.yaml: %v", err)
		return
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Printf("failed to parse config.yaml: %v", err)
	}
}

func main() {
	loadConfig()

	// Read secrets from environment ( use a secrets manager in production).
	dbConn := os.Getenv("DB_CONNECTION")
	apiKey := os.Getenv("EXTERNAL_API_KEY")

	// Read port from environment, default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	// Optional debug flag for server-side log verbosity (does NOT affect what we return to clients).
	debugMode := os.Getenv("DEBUG") == "1"

	// build our mux with routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)

	// Pass secrets into the handler via closure instead of globals and never expose them in responses.
	mux.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		dataHandler(w, r, dbConn, apiKey)
	})

	mux.HandleFunc("/cause_error", causeErrorHandler)

	// wrap mux with recovery middleware
	handler := recoveryMiddleware(mux)

	log.Printf("Starting server on %s (debug=%v)", addr, debugMode)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "Hello, world!"})
}

// dataHandler receives secrets as arguments but does NOT include them in responses. Non-sensitive metadata only
func dataHandler(w http.ResponseWriter, r *http.Request, dbConn, apiKey string) {
	resp := map[string]interface{}{
		"status":            "ok",
		"db_connection_set": dbConn != "",
		"api_key_set":       apiKey != "",
		"config_db_host":    cfg.Database.Host,
		"config_db_user":    cfg.Database.User,
		"note":              "secrets are not returned in responses",
	}
	writeJSON(w, http.StatusOK, resp)
}

func causeErrorHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate a panic recovery middleware will handle it safely.
	panic("simulated panic for demo")
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// If encoding fails, there's not much the handler can do log server-side and send a minimal message.
		log.Printf("failed to write json response: %v", err)
		http.Error(w, `{"error":"internal_server_error"}`, http.StatusInternalServerError)
	}
}

// recoveryMiddleware logs the full stack on the server but returns a general error to the client.
// IMPORTANT: stack traces and panic details are never returned to clients.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				// Server-side logging includes the recovered value and stack trace for diagnostics.
				log.Printf("panic recovered: %v\n%s", rec, stack)

				// Return a general error to the client without any internal details.
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "internal_server_error",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
