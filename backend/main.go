package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type ContainerStatus struct {
	IP          string    `json:"ip"`
	PingTime    int       `json:"ping_time"`
	LastSuccess time.Time `json:"last_success"`
}

var db *sql.DB

func initDB() {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("Cannot ping DB: %v", err)
	}
	log.Println("Connected to database")

	// Создаем таблицу, если ее нет
	query := `
	CREATE TABLE IF NOT EXISTS container_status (
		ip TEXT PRIMARY KEY,
		ping_time INT,
		last_success TIMESTAMP
	);
	`
	if _, err = db.Exec(query); err != nil {
		log.Fatalf("Error creating table: %v", err)
	}
}

func getStatuses(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT ip, ping_time, last_success FROM container_status")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var statuses []ContainerStatus
	for rows.Next() {
		var cs ContainerStatus
		if err := rows.Scan(&cs.IP, &cs.PingTime, &cs.LastSuccess); err != nil {
			http.Error(w, "Error scanning row", http.StatusInternalServerError)
			return
		}
		statuses = append(statuses, cs)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

func updateStatus(w http.ResponseWriter, r *http.Request) {
	var cs ContainerStatus
	if err := json.NewDecoder(r.Body).Decode(&cs); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	// Upsert: если запись с таким IP уже есть, обновляем её
	query := `
	INSERT INTO container_status (ip, ping_time, last_success)
	VALUES ($1, $2, $3)
	ON CONFLICT (ip) DO UPDATE SET
		ping_time = EXCLUDED.ping_time,
		last_success = EXCLUDED.last_success;
	`
	if _, err := db.Exec(query, cs.IP, cs.PingTime, cs.LastSuccess); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func main() {
	initDB()
	r := mux.NewRouter()
	r.HandleFunc("/status", getStatuses).Methods("GET")
	r.HandleFunc("/status", updateStatus).Methods("POST")

	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Backend service listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}