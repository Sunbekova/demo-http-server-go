package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime/debug"

	"gopkg.in/yaml.v3"
)

// hard-coded secrets
var (
	DB_CONNECTION    = "postgres://admin:HardCodedPassword@localhost:5432/demo_db"
	EXTERNAL_API_KEY = "EXTERNAL_API_KEY_123456789"
)

// unsafe flag: если true — отправляем стек-трейсы в ответе
var DEBUG = true

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
	data, err := ioutil.ReadFile("config.yaml")
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

	// build our mux with routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/data", dataHandler)
	mux.HandleFunc("/cause_error", causeErrorHandler)

	// wrap mux with recovery middleware
	handler := recoveryMiddleware(mux)

	addr := ":8080"
	log.Printf("Starting server on %s (DEBUG=%v)", addr, DEBUG)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "Hello, world!"})
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"status":          "ok",
		"db_conn_preview": DB_CONNECTION,
		"used_api_key":    fmt.Sprintf("%s...", EXTERNAL_API_KEY[:8]),
		"config_password": cfg.Database.Password,
	}
	writeJSON(w, http.StatusOK, resp)
}

func causeErrorHandler(w http.ResponseWriter, r *http.Request) {
	panic("simulated panic for demo")
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Recovery middleware
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				log.Printf("panic recovered: %v\n%s", rec, stack)
				if DEBUG {
					writeJSON(w, http.StatusInternalServerError, map[string]string{
						"error": fmt.Sprintf("%v", rec),
						"stack": string(stack),
					})
				} else {
					writeJSON(w, http.StatusInternalServerError, map[string]string{
						"error": "internal_server_error",
					})
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}
