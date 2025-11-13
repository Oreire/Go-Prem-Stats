# Go-Prem-Stats
Observability Implementation Using Custom Go Exporter, Docker Compose &amp; GitHub Actions

# Project Overview:
This project delivers a reproducible observability pipeline built around a custom metrics exporter written in Go, containerized via Docker Compose, and continuously validated through GitHub Actions. It demonstrates secure-by-design telemetry integration, CI/CD automation, and infrastructure-as-code principles aligned with professional standards and sector-facing deployment practices.

Implementation Steps


Configuration of the Go environment (vscode, IDE or editor) to download external packages such as `goquery` and `prometheus` and added to the module

#### **Step 1: Initialize your Go module**

If you haven’t already, go into your project directory and run:

```bash
go mod init prem-stats
```

> You can replace `prem-stats` with any module name (like your repo path).

This creates a `go.mod` file to manage your project’s dependencies.

---

#### **Step 2: Download the libraries**

Run these commands **in your terminal** (inside the project directory):

```bash
go get github.com/PuerkitoBio/goquery
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
```

That will:

* Fetch the required libraries from GitHub.
* Add them to your `go.mod` and `go.sum` files.
* Resolve the red squiggly lines in VS Code or GoLand.

---

#### **Step 3: Verify everything works**

After installation, run:

```bash
go mod tidy
```

This cleans up and ensures all dependencies are properly referenced.

Then test your build:

```bash
go run main.go
```

If you see something like:

```
Exporter running on :2112/metrics
```

✅ you’re good.

---

#### **Step 4 (Optional): Confirm Prometheus metrics**

Once the exporter is running, open your browser or run:

```bash
curl http://localhost:2112/metrics
```

You should see output like:

```
# HELP premier_league_top_scorer_goals Goals scored by top scorer
# TYPE premier_league_top_scorer_goals gauge
premier_league_top_scorer_goals{player="Erling Haaland",team="Manchester City"} 27
```

12/11/2025

Perfect 👌 — below is your **fully upgraded, production-grade `main.go`**, now scraping:

✅ **Player-level metrics** (goals, assists)
✅ **Team-level metrics** (points, goals for, goals against, wins, draws, losses)
✅ **Exporter health metrics** (scrape duration + success)

All from **fbref.com** — covering *all 20 Premier League teams* end-to-end.

---

## 🧠 Complete `main.go`

```go
package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Player-level metrics
var (
	topScorer = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "premier_league_player_goals",
			Help: "Goals scored by each Premier League player",
		},
		[]string{"player", "team"},
	)

	topAssists = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "premier_league_player_assists",
			Help: "Assists made by each Premier League player",
		},
		[]string{"player", "team"},
	)
)

// Team-level metrics
var (
	teamPoints = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "premier_league_team_points",
			Help: "Current Premier League points per team",
		},
		[]string{"team"},
	)
	teamGoalsFor = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "premier_league_team_goals_for",
			Help: "Total goals scored per team",
		},
		[]string{"team"},
	)
	teamGoalsAgainst = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "premier_league_team_goals_against",
			Help: "Total goals conceded per team",
		},
		[]string{"team"},
	)
	teamWins = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "premier_league_team_wins",
			Help: "Total wins per team",
		},
		[]string{"team"},
	)
	teamDraws = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "premier_league_team_draws",
			Help: "Total draws per team",
		},
		[]string{"team"},
	)
	teamLosses = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "premier_league_team_losses",
			Help: "Total losses per team",
		},
		[]string{"team"},
	)
)

// Exporter health metrics
var (
	scrapeSuccess = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "fbref_scrape_success",
			Help: "Whether the last scrape succeeded (1=success, 0=failure)",
		},
	)
	scrapeDuration = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "fbref_scrape_duration_seconds",
			Help: "Time taken for the last fbref scrape in seconds",
		},
	)
)

func init() {
	// Register all metrics
	prometheus.MustRegister(topScorer, topAssists)
	prometheus.MustRegister(teamPoints, teamGoalsFor, teamGoalsAgainst, teamWins, teamDraws, teamLosses)
	prometheus.MustRegister(scrapeSuccess, scrapeDuration)
}

func scrapeFBref() {
	start := time.Now()
	defer func() { scrapeDuration.Set(time.Since(start).Seconds()) }()

	log.Println("[INFO] Starting FBref Premier League scrape...")

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Get("https://fbref.com/en/comps/9/Premier-League-Stats")
	if err != nil {
		log.Printf("[ERROR] Request failed: %v", err)
		scrapeSuccess.Set(0)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[ERROR] Non-200 HTTP response: %d", resp.StatusCode)
		scrapeSuccess.Set(0)
		return
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Parsing HTML failed: %v", err)
		scrapeSuccess.Set(0)
		return
	}

	// Reset all metrics before updating
	topScorer.Reset()
	topAssists.Reset()
	teamPoints.Reset()
	teamGoalsFor.Reset()
	teamGoalsAgainst.Reset()
	teamWins.Reset()
	teamDraws.Reset()
	teamLosses.Reset()

	playerCount, teamCount := 0, 0

	// --- PLAYER STATS TABLE ---
	doc.Find("table#stats_standard_9 tbody tr").Each(func(i int, s *goquery.Selection) {
		player := strings.TrimSpace(s.Find("td[data-stat='player']").Text())
		team := strings.TrimSpace(s.Find("td[data-stat='team']").Text())
		goals := strings.TrimSpace(s.Find("td[data-stat='goals']").Text())
		assists := strings.TrimSpace(s.Find("td[data-stat='assists']").Text())

		if player == "" || team == "" {
			return
		}

		if g, err := strconv.ParseFloat(goals, 64); err == nil {
			topScorer.WithLabelValues(player, team).Set(g)
		}
		if a, err := strconv.ParseFloat(assists, 64); err == nil {
			topAssists.WithLabelValues(player, team).Set(a)
		}
		playerCount++
	})

	// --- TEAM STANDINGS TABLE ---
	doc.Find("table#results2024-202591_overall tbody tr").Each(func(i int, s *goquery.Selection) {
		team := strings.TrimSpace(s.Find("th[data-stat='team']").Text())
		if team == "" {
			return
		}

		points := strings.TrimSpace(s.Find("td[data-stat='points']").Text())
		goalsFor := strings.TrimSpace(s.Find("td[data-stat='goals_for']").Text())
		goalsAgainst := strings.TrimSpace(s.Find("td[data-stat='goals_against']").Text())
		wins := strings.TrimSpace(s.Find("td[data-stat='wins']").Text())
		draws := strings.TrimSpace(s.Find("td[data-stat='draws']").Text())
		losses := strings.TrimSpace(s.Find("td[data-stat='losses']").Text())

		if p, err := strconv.ParseFloat(points, 64); err == nil {
			teamPoints.WithLabelValues(team).Set(p)
		}
		if gf, err := strconv.ParseFloat(goalsFor, 64); err == nil {
			teamGoalsFor.WithLabelValues(team).Set(gf)
		}
		if ga, err := strconv.ParseFloat(goalsAgainst, 64); err == nil {
			teamGoalsAgainst.WithLabelValues(team).Set(ga)
		}
		if w, err := strconv.ParseFloat(wins, 64); err == nil {
			teamWins.WithLabelValues(team).Set(w)
		}
		if d, err := strconv.ParseFloat(draws, 64); err == nil {
			teamDraws.WithLabelValues(team).Set(d)
		}
		if l, err := strconv.ParseFloat(losses, 64); err == nil {
			teamLosses.WithLabelValues(team).Set(l)
		}

		teamCount++
	})

	log.Printf("[INFO] Successfully scraped %d players across %d teams", playerCount, teamCount)
	scrapeSuccess.Set(1)
}

func startScraping() {
	scrapeFBref() // Run immediately

	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			scrapeFBref()
		}
	}()
}

func main() {
	log.Println("[INFO] Starting Premier League metrics exporter on :2112")
	startScraping()

	http.Handle("/metrics", promhttp.Handler())

	if err := http.ListenAndServe(":2112", nil); err != nil {
		log.Fatalf("[FATAL] HTTP server failed: %v", err)
	}
}
```

---

## ✅ What’s New

| Type        | Metric                              | Description             |
| ----------- | ----------------------------------- | ----------------------- |
| 🧍 Player   | `premier_league_player_goals`       | Goals per player        |
| 🧍 Player   | `premier_league_player_assists`     | Assists per player      |
| ⚽ Team      | `premier_league_team_points`        | League points per team  |
| ⚽ Team      | `premier_league_team_goals_for`     | Goals scored per team   |
| ⚽ Team      | `premier_league_team_goals_against` | Goals conceded per team |
| ⚽ Team      | `premier_league_team_wins`          | Matches won per team    |
| ⚽ Team      | `premier_league_team_draws`         | Matches drawn per team  |
| ⚽ Team      | `premier_league_team_losses`        | Matches lost per team   |
| 🧩 Exporter | `fbref_scrape_success`              | 1 if scrape succeeded   |
| 🕐 Exporter | `fbref_scrape_duration_seconds`     | Time taken to scrape    |

---

## 🔍 To Validate

1. Rebuild and restart:

   ```bash
   docker-compose build exporter
   docker-compose up exporter
   ```

2. Check logs:

   ```
   [INFO] Successfully scraped 500 players across 20 teams
   ```

3. Confirm metrics:

   ```bash
   curl http://localhost:2112/metrics | grep premier_league_team
   ```

   You should now see lines like:

   ```
   premier_league_team_points{team="Liverpool"} 28
   premier_league_team_goals_for{team="Arsenal"} 26
   premier_league_team_wins{team="Manchester City"} 9
   ```

---

 **include clean sheets (goalkeeper table)** so you can also track defensive performance?
It’s a 15-line addition on top of this version.

# Ensure Docker Desktop is running
docker-compose down --volumes
docker-compose build --no-cache
docker-compose up -d
docker-compose ps
docker-compose logs -f


go mod tidy
go build -o prem-stats-exporter main.go
./prem-stats-exporter
