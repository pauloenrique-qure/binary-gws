package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	requestCount uint64
	failCount    int = 2 // Primeros 2 requests fallarán
)

func handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	count := atomic.AddUint64(&requestCount, 1)

	log.Printf("[Request #%d] Method: %s, Path: %s", count, r.Method, r.URL.Path)
	log.Printf("[Request #%d] Authorization: %s", count, r.Header.Get("Authorization"))

	// Simular fallos en los primeros N requests
	if count <= uint64(failCount) {
		log.Printf("[Request #%d] 🔴 SIMULATING SERVER ERROR (HTTP 500)", count)
		http.Error(w, "Internal Server Error - Simulated", http.StatusInternalServerError)
		return
	}

	// Leer body
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("[Request #%d] Error parsing JSON: %v", count, err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Pretty print
	prettyJSON, _ := json.MarshalIndent(payload, "", "  ")
	log.Printf("[Request #%d] 🟢 SUCCESS - Payload received:\n%s", count, string(prettyJSON))

	// Success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status":      "success",
		"message":     "Heartbeat received",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"request_id":  count,
		"retry_count": count - 1,
	}
	json.NewEncoder(w).Encode(response)

	log.Printf("[Request #%d] ✅ Response sent successfully (after %d retries)\n", count, count-1)
}

func main() {
	http.HandleFunc("/heartbeat", handleHeartbeat)

	addr := ":8080"
	log.Printf("🚀 Retry Test Server starting on %s", addr)
	log.Printf("📡 Endpoint: http://localhost:8080/heartbeat")
	log.Printf("⚠️  First %d requests will return HTTP 500 to test retry logic", failCount)
	log.Println("Waiting for heartbeats...")
	log.Println("")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
