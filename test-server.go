package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

var requestCount int

func handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	requestCount++

	// Log request details
	log.Printf("[Request #%d] Method: %s, Path: %s", requestCount, r.Method, r.URL.Path)
	log.Printf("[Request #%d] Authorization: %s", requestCount, r.Header.Get("Authorization"))

	// Read and parse body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Request #%d] Error reading body: %v", requestCount, err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse JSON
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[Request #%d] Error parsing JSON: %v", requestCount, err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Pretty print the payload
	prettyJSON, _ := json.MarshalIndent(payload, "", "  ")
	log.Printf("[Request #%d] Payload received:\n%s", requestCount, string(prettyJSON))

	// Validate required fields
	requiredFields := []string{"uuid", "client_id", "site_id", "payload_version"}
	for _, field := range requiredFields {
		if _, ok := payload[field]; !ok {
			log.Printf("[Request #%d] Missing required field: %s", requestCount, field)
			http.Error(w, fmt.Sprintf("Missing field: %s", field), http.StatusBadRequest)
			return
		}
	}

	// Check stats
	if stats, ok := payload["stats"].(map[string]interface{}); ok {
		if status, ok := stats["system_status"]; ok {
			log.Printf("[Request #%d] ✅ System Status: %v", requestCount, status)
		}
		if compute, ok := stats["compute"].(map[string]interface{}); ok {
			log.Printf("[Request #%d] ✅ Compute metrics present: %v", requestCount, compute != nil)
		}
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status":     "success",
		"message":    "Heartbeat received",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"request_id": requestCount,
	}
	json.NewEncoder(w).Encode(response)

	log.Printf("[Request #%d] ✅ Response sent successfully\n", requestCount)
}

func main() {
	http.HandleFunc("/heartbeat", handleHeartbeat)

	addr := ":8080"
	log.Printf("🚀 Test server starting on %s", addr)
	log.Printf("📡 Endpoint: http://localhost:8080/heartbeat")
	log.Println("Waiting for heartbeats...")
	log.Println("")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
