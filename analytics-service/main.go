// Skill Broker — Analytics Service
//
// Receives opt-in usage events from the broker hook.
// Only accepts skill_id — never prompt text.
//
// Endpoints:
//   POST /events   { "event": "inject|pin|install|flag", "skill_id": "..." }
//   GET  /health
//
// Deploy to Railway: set PORT env var (Railway sets this automatically).
// Events are appended to EVENTS_LOG (default: events.log).

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	logPath = envOr("EVENTS_LOG", "events.log")
	mu      sync.Mutex
)

type Event struct {
	Event   string `json:"event"`
	SkillID string `json:"skill_id"`
}

type LogEntry struct {
	Timestamp string `json:"ts"`
	Event     string `json:"event"`
	SkillID   string `json:"skill_id"`
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func writeEvent(e Event) error {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Event:     e.Event,
		SkillID:   e.SkillID,
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var e Event
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// sanitise — accept only known event types and non-empty skill IDs
	validEvents := map[string]bool{"inject": true, "pin": true, "install": true, "flag": true}
	e.Event = strings.ToLower(strings.TrimSpace(e.Event))
	e.SkillID = strings.TrimSpace(e.SkillID)

	if !validEvents[e.Event] || e.SkillID == "" {
		http.Error(w, "invalid event or skill_id", http.StatusBadRequest)
		return
	}

	if err := writeEvent(e); err != nil {
		log.Printf("error writing event: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func main() {
	port := envOr("PORT", "8788")

	mux := http.NewServeMux()
	mux.HandleFunc("/events", handleEvents)
	mux.HandleFunc("/health", handleHealth)

	log.Printf("analytics service listening on :%s", port)
	log.Printf("events log: %s", logPath)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
