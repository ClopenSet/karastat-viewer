package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed static/*
var embeddedFiles embed.FS

type KeyCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type HeatmapColor struct {
	ID    string `json:"id"`
	Color string `json:"fill"`
	Count int    `json:"count"`
}

func main() {
	// static directory
	staticFS, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		log.Fatalf("embed failure: %v", err)
	}
	http.Handle("/", http.FileServer(http.FS(staticFS)))

	// SQLite
	dbPath := filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "KaraStat", "key_stats.sqlite")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failure in opening database: %v", err)
	}
	defer db.Close()

	// SSE 
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				rows, err := db.Query("SELECT key, count FROM key_counts")
				if err != nil {
					continue
				}
				var data []KeyCount
				for rows.Next() {
					var k KeyCount
					if err := rows.Scan(&k.Key, &k.Count); err == nil {
						data = append(data, k)
					}
				}
				rows.Close()

				counts := make([]int, len(data))
				for i, d := range data {
					counts[i] = d.Count
				}
				getRatio := PercentileClip(counts, 0.95)
				getColor := RedGreenColor

				var results []HeatmapColor
				for _, item := range data {
					ratio := getRatio(item.Count)
					color := getColor(ratio)
					results = append(results, HeatmapColor{
						ID:    item.Key + "-inner",
						Color: color,
						Count: item.Count,
					})
				}

				b, _ := json.Marshal(results)
				fmt.Fprintf(w, "data: %s\n\n", b)
				flusher.Flush()
			}
		}
	})

	fmt.Println("🔥 Visit http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func PercentileClip(counts []int, percentile float64) func(int) float64 {
	sorted := append([]int{}, counts...)
	sort.Ints(sorted)
	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	max := sorted[index]
	if max <= 0 {
		return func(int) float64 { return 0 }
	}
	return func(v int) float64 {
		r := float64(v) / float64(max)
		if r > 1 {
			return 1
		}
		return r
	}
}

func RedGreenColor(ratio float64) string {
	r := int(255 * ratio)
	g := int(255 * (1 - ratio))
	return fmt.Sprintf("rgba(%d,%d,60,0.7)", r, g)
}
