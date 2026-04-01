package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Weight struct {
	ID         int       `json:"id"`
	Weight     float64   `json:"weight"`
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
	http.HandleFunc("/api/weights/export", handleExport)
	http.HandleFunc("/api/weights/import", handleImport)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	port := ":8080"
	log.Printf("Server starting on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func initDB() (*sql.DB, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "weights.db"
	}
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

func handleExport(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT weight, recorded_date FROM weights ORDER BY recorded_date")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch weights: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=weights.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"date", "weight"})
	for rows.Next() {
		var weight float64
		var date string
		if err := rows.Scan(&weight, &date); err != nil {
			continue
		}
		writer.Write([]string{date, strconv.FormatFloat(weight, 'f', 1, 64)})
	}
}

func handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(1 << 20) // 1MB max
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Skip header row
	if _, err := reader.Read(); err != nil {
		http.Error(w, "Failed to read CSV header", http.StatusBadRequest)
		return
	}

	imported := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 2 {
			continue
		}

		date := record[0]
		weight, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			continue
		}

		_, err = db.Exec(
			"INSERT INTO weights (weight, recorded_date) VALUES (?, ?) ON CONFLICT(recorded_date) DO UPDATE SET weight=excluded.weight",
			weight, date,
		)
		if err == nil {
			imported++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "imported": imported})
}
