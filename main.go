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
	"math"

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
				getRatio := LogNormalizer(counts)
				getColor := RainbowColor

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

// RainbowColor maps a normalized ratio [0.0, 1.0] to a perceptual color
// using the HSL color space.
//
// The hue range is set from 270° (purple-blue) to 0° (red), covering the
// full perceptual "rainbow" gradient: purple → blue → cyan → green → yellow → red.
// Even the lowest input values (e.g., v = 1) produce a small non-zero ratio
// when using logarithmic normalization (e.g., log(2)/log(max+1)).
// Therefore, to ensure we still see the coldest colors (near hue=270°),
// we expand the hue range to start from 270°, not 240°.

func RainbowColor(ratio float64) string {
    // Hue from 270° (purple) to 0° (red)
    h := int(270 * (1 - ratio))  // ratio=0 → purple =1 → red
    return fmt.Sprintf("hsl(%d, 100%%, 50%%)", h)
}

func LogNormalizer(counts []int) func(int) float64 {
	max := 0
	for _, v := range counts {
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		return func(int) float64 { return 0 }
	}
	logMax := math.Log(float64(max) + 1)
	return func(v int) float64 {
		if v < 0 {
			v = 0
		}
		r := math.Log(float64(v)+1) / logMax
		if r > 1 {
			return 1
		}
		return r
	}
}
