package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test-weights.db")

	if err := os.Setenv("DB_PATH", dbPath); err != nil {
		t.Fatalf("failed to set DB_PATH: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("DB_PATH")
	})

	var err error
	db, err = initDB()
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})
}

func TestRecordWeightUpsertsDailyEntry(t *testing.T) {
	setupTestDB(t)

	firstBody := url.Values{"weight": {"180.2"}}.Encode()
	firstReq := httptest.NewRequest(http.MethodPost, "/api/weight", strings.NewReader(firstBody))
	firstReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	firstRes := httptest.NewRecorder()

	recordWeight(firstRes, firstReq)

	if firstRes.Code != http.StatusOK {
		t.Fatalf("expected first record status 200, got %d", firstRes.Code)
	}

	secondBody := url.Values{"weight": {"181.4"}}.Encode()
	secondReq := httptest.NewRequest(http.MethodPost, "/api/weight", strings.NewReader(secondBody))
	secondReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	secondRes := httptest.NewRecorder()

	recordWeight(secondRes, secondReq)

	if secondRes.Code != http.StatusOK {
		t.Fatalf("expected second record status 200, got %d", secondRes.Code)
	}

	var rowCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM weights").Scan(&rowCount); err != nil {
		t.Fatalf("failed to count rows: %v", err)
	}

	if rowCount != 1 {
		t.Fatalf("expected 1 row after upsert, got %d", rowCount)
	}

	var savedWeight float64
	if err := db.QueryRow("SELECT weight FROM weights LIMIT 1").Scan(&savedWeight); err != nil {
		t.Fatalf("failed to fetch saved weight: %v", err)
	}

	if savedWeight != 181.4 {
		t.Fatalf("expected updated weight 181.4, got %.1f", savedWeight)
	}
}

func TestHandleWeightsReturnsChronologicalJSON(t *testing.T) {
	setupTestDB(t)

	rows := []struct {
		weight       float64
		recordedDate string
		recordedAt   string
	}{
		{weight: 183.0, recordedDate: "2026-01-01", recordedAt: "2026-01-01 08:00:00"},
		{weight: 182.5, recordedDate: "2026-01-02", recordedAt: "2026-01-02 08:00:00"},
		{weight: 182.1, recordedDate: "2026-01-03", recordedAt: "2026-01-03 08:00:00"},
	}

	for _, row := range rows {
		_, err := db.Exec(
			"INSERT INTO weights (weight, recorded_date, recorded_at) VALUES (?, ?, ?)",
			row.weight,
			row.recordedDate,
			row.recordedAt,
		)
		if err != nil {
			t.Fatalf("failed inserting seed data: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/weights", nil)
	res := httptest.NewRecorder()

	handleWeights(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Code)
	}

	var weights []Weight
	if err := json.Unmarshal(res.Body.Bytes(), &weights); err != nil {
		t.Fatalf("failed to decode json response: %v", err)
	}

	if len(weights) != 3 {
		t.Fatalf("expected 3 weights, got %d", len(weights))
	}

	for i := 1; i < len(weights); i++ {
		if weights[i].RecordedAt.Before(weights[i-1].RecordedAt) {
			t.Fatalf("weights were not returned chronologically")
		}
	}

	if weights[0].Weight != 183.0 || weights[2].Weight != 182.1 {
		t.Fatalf("unexpected response weights: first=%.1f last=%.1f", weights[0].Weight, weights[2].Weight)
	}

	if !weights[0].RecordedAt.Equal(time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected first timestamp to parse correctly, got %s", weights[0].RecordedAt)
	}
}