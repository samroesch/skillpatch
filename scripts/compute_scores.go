// Nightly trust score computation.
// Reads storage/events.log, recomputes trust scores, updates storage/index.json.
//
// Run: go run scripts/compute_scores.go
// Or via GitHub Action: .github/workflows/update-scores.yml

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// event weights for trust score contribution
const (
	weightInject  = 1.0
	weightPin     = 5.0
	weightInstall = 10.0
	weightFlag    = -20.0
)

type LogEntry struct {
	Timestamp string `json:"ts"`
	Event     string `json:"event"`
	SkillID   string `json:"skill_id"`
}

type Skill struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Summary    string   `json:"summary"`
	Tags       []string `json:"tags"`
	Keywords   []string `json:"keywords"`
	Verified   bool     `json:"verified"`
	TrustScore float64  `json:"trust_score"`
	UsageCount int      `json:"usage_count"`
	ContentURL string   `json:"content_url"`
	Version    string   `json:"version"`
	SHA        string   `json:"sha,omitempty"`
	Install    string   `json:"install,omitempty"`
}

func main() {
	indexPath := "storage/index.json"
	eventsPath := "storage/events.log"

	// load index
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading index: %v\n", err)
		os.Exit(1)
	}
	var skills []Skill
	if err := json.Unmarshal(indexData, &skills); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing index: %v\n", err)
		os.Exit(1)
	}

	// tally events per skill
	type tally struct {
		injects  float64
		pins     float64
		installs float64
		flags    float64
	}
	tallies := map[string]*tally{}
	for _, s := range skills {
		tallies[s.ID] = &tally{}
	}

	f, err := os.Open(eventsPath)
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error reading events: %v\n", err)
		os.Exit(1)
	}
	if f != nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var e LogEntry
			if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
				continue
			}
			t, ok := tallies[e.SkillID]
			if !ok {
				continue
			}
			switch e.Event {
			case "inject":
				t.injects++
			case "pin":
				t.pins++
			case "install":
				t.installs++
			case "flag":
				t.flags++
			}
		}
	}

	// recompute scores
	for i, s := range skills {
		t := tallies[s.ID]
		if t == nil {
			continue
		}
		raw := t.injects*weightInject +
			t.pins*weightPin +
			t.installs*weightInstall +
			t.flags*weightFlag

		// log-scale the raw signal so early usage matters but
		// doesn't dominate verified/quality signals
		signal := math.Log1p(math.Max(0, raw))

		// base score: verified gets a floor of 3.0
		base := 2.0
		if s.Verified {
			base = 3.0
		}

		// cap at 5.0
		score := math.Min(5.0, base+signal*0.5)
		skills[i].TrustScore = math.Round(score*10) / 10

		// usage_count = total inject events (install + pin contribute more but
		// we keep usage_count as raw exposure metric)
		skills[i].UsageCount = int(t.injects) + int(t.pins)*5 + int(t.installs)*10
	}

	// write updated index
	out, err := json.MarshalIndent(skills, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(indexPath, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing index: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("updated %d skills in %s\n", len(skills), indexPath)
	for _, s := range skills {
		t := tallies[s.ID]
		fmt.Printf("  %-30s score=%.1f  inj=%.0f pin=%.0f inst=%.0f flag=%.0f\n",
			s.ID, s.TrustScore, t.injects, t.pins, t.installs, t.flags)
	}
}
