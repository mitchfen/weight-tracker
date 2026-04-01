package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Weight struct {
	ID        int       `json:"id"`
	Weight    float64   `json:"weight"`
	RecordedAt time.Time `json:"recorded_at"`
}

func main() {
	var err error
	db, err = initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/", serveIndex)
	http.HandleFunc("/api/weight", handleWeight)
	http.HandleFunc("/api/weights", handleWeights)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	port := ":8080"
	log.Printf("Server starting on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func initDB() (*sql.DB, error) {
	dbPath := "weights.db"
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create table if it doesn't exist
	createTable := `
	CREATE TABLE IF NOT EXISTS weights (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		weight REAL NOT NULL,
		recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		recorded_date DATE UNIQUE NOT NULL
	);
	`
	_, err = database.Exec(createTable)
	if err != nil {
		return nil, err
	}

	return database, nil
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "templates/index.html")
}

func handleWeight(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		recordWeight(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func recordWeight(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	weightStr := r.FormValue("weight")
	if weightStr == "" {
		http.Error(w, "Weight is required", http.StatusBadRequest)
		return
	}

	var weight float64
	_, err = fmt.Sscanf(weightStr, "%f", &weight)
	if err != nil {
		http.Error(w, "Invalid weight format", http.StatusBadRequest)
		return
	}

	// Record only for today
	today := time.Now().Format("2006-01-02")
	_, err = db.Exec(
		"INSERT INTO weights (weight, recorded_date) VALUES (?, ?) ON CONFLICT(recorded_date) DO UPDATE SET weight=excluded.weight",
		weight,
		today,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to record weight: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Weight recorded"})
}

func handleWeights(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, weight, recorded_at FROM weights ORDER BY recorded_at")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch weights: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	weights := []Weight{}
	for rows.Next() {
		var wt Weight
		err := rows.Scan(&wt.ID, &wt.Weight, &wt.RecordedAt)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan weight: %v", err), http.StatusInternalServerError)
			return
		}
		weights = append(weights, wt)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weights)
}
