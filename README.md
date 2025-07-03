# karastat-viewer

**karastat-viewer** is a companion visualization tool for [karastat](https://github.com/ClopenSet/KaraStat), a project that records keyboard key counts. This viewer renders an interactive heatmap of key usage on a macOS keyboard using a pre-labeled SVG layout and real-time key statistics from an SQLite database.

---

## 🔧 Requirements

This viewer depends on the existence of the `key_counts` table provided by `karastat`, stored in a SQLite database located at:

```
~/Library/Application Support/KaraStat/key_stats.sqlite
```

> ⚠️ Without this database, the viewer will not display any heatmap data.

---

## 🖼️ SVG Layout

This project includes a carefully crafted SVG file representing a macOS keyboard. Each key is represented by:

* A **group** (e.g., `<g id="a">`)
* An **inner ring** path with an `id` like `a-inner`
* An **outer ring** path with an `id` like `a-outer`

This consistent naming allows dynamic color updates of each key's inner fill, based on usage data.

---

## 🚀 Features

* **Live-updating** key heatmap using Server-Sent Events (SSE)
* **Color-coded** usage: green (low), red (high), with smooth gradient
* **Hover count preview**
* **Optional in-key count display**
* **No polling, no reloads — fully real-time**

---

## 📂 Project Structure

```
.
├── main.go          ← Go backend server (embedded mode)
└── static/          ← Embedded frontend resources
    ├── index.html   ← Main UI page
    ├── render.js    ← SVG logic & heatmap rendering
    └── keyboard.svg ← Named SVG layout of the macOS keyboard
```

> All static files are embedded into the Go binary using `go:embed`. No external files are required at runtime.

---

## 🏃 Usage

### Development:

```bash
go run main.go
```

Then open [http://localhost:8080](http://localhost:8080) in your browser.

### Production Build:

```bash
go build -ldflags="-s -w" -o karastat-viewer
```

This produces a standalone binary (\~10–15MB) that includes the web UI, logic, and SQLite access.

This software is a depedence of KaraStat, which can be installed by brew. See also [karastat](https://github.com/ClopenSet/KaraStat).

---

## 📡 How It Works

* The Go server exposes a `/events` endpoint that streams live JSON updates using SSE every second.
* The frontend (`render.js`) connects via `EventSource`, parses the streamed JSON, and updates the colors of keys accordingly.
* Optional: toggle on-screen key counts with the “Show count” button.

---

## 📝 Notes

* The SVG file was created using **Inkscape**, and the naming convention (`key-inner`, `key-outer`) is strictly followed for compatibility.
* This project assumes a **macOS keyboard layout**. Adapting for other layouts requires modifying the SVG structure accordingly.
