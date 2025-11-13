package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// --------------------- Metrics Definitions ---------------------

var (
	// Player-level metrics
	topScorer = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "premier_league_player_goals", Help: "Goals scored by each Premier League player"},
		[]string{"player", "team"},
	)
	topAssists = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "premier_league_player_assists", Help: "Assists made by each Premier League player"},
		[]string{"player", "team"},
	)
	cleanSheets = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "premier_league_goalkeeper_clean_sheets", Help: "Number of clean sheets by each goalkeeper"},
		[]string{"player", "team"},
	)

	// Team-level metrics
	teamPoints       = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "premier_league_team_points", Help: "Current Premier League points per team"}, []string{"team"})
	teamGoalsFor     = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "premier_league_team_goals_for", Help: "Total goals scored per team"}, []string{"team"})
	teamGoalsAgainst = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "premier_league_team_goals_against", Help: "Total goals conceded per team"}, []string{"team"})
	teamWins         = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "premier_league_team_wins", Help: "Total wins per team"}, []string{"team"})
	teamDraws        = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "premier_league_team_draws", Help: "Total draws per team"}, []string{"team"})
	teamLosses       = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "premier_league_team_losses", Help: "Total losses per team"}, []string{"team"})

	// Exporter health metrics
	scrapeSuccess  = prometheus.NewGauge(prometheus.GaugeOpts{Name: "fbref_scrape_success", Help: "Whether the last scrape succeeded (1=success, 0=failure)"})
	scrapeDuration = prometheus.NewGauge(prometheus.GaugeOpts{Name: "fbref_scrape_duration_seconds", Help: "Time taken for the last FBref scrape in seconds"})
)

func init() {
	prometheus.MustRegister(topScorer, topAssists, cleanSheets)
	prometheus.MustRegister(teamPoints, teamGoalsFor, teamGoalsAgainst, teamWins, teamDraws, teamLosses)
	prometheus.MustRegister(scrapeSuccess, scrapeDuration)
}

// --------------------- HTML Fetching ---------------------

func fetchHTML(url string) (*goquery.Document, error) {
	client := &http.Client{Timeout: 25 * time.Second}
	for attempt := 1; attempt <= 3; attempt++ {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Referer", "https://fbref.com/")
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			if resp != nil {
				resp.Body.Close()
			}
			log.Printf("[WARN] Attempt %d failed: %v. Retrying...", attempt, err)
			time.Sleep(time.Duration(attempt*2) * time.Second)
			continue
		}
		doc, err := goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("[WARN] Failed to parse HTML on attempt %d: %v", attempt, err)
			time.Sleep(time.Duration(attempt*2) * time.Second)
			continue
		}
		return doc, nil
	}
	return nil, fmt.Errorf("failed to fetch HTML after 3 attempts")
}

// --------------------- Scraper Logic ---------------------

func extractCommentTables(html string) []*goquery.Document {
	re := regexp.MustCompile(`<!--([\s\S]*?)-->`)
	matches := re.FindAllStringSubmatch(html, -1)
	var docs []*goquery.Document
	for _, m := range matches {
		if strings.Contains(m[1], "<table") {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(m[1]))
			if err == nil {
				docs = append(docs, doc)
			}
		}
	}
	return docs
}

func scrapeFBref() {
	start := time.Now()
	defer func() { scrapeDuration.Set(time.Since(start).Seconds()) }()

	log.Println("[INFO] Starting FBref Premier League scrape...")

	// Reset metrics
	topScorer.Reset()
	topAssists.Reset()
	cleanSheets.Reset()
	teamPoints.Reset()
	teamGoalsFor.Reset()
	teamGoalsAgainst.Reset()
	teamWins.Reset()
	teamDraws.Reset()
	teamLosses.Reset()

	doc, err := fetchHTML("https://fbref.com/en/comps/9/Premier-League-Stats")
	if err != nil {
		log.Printf("[ERROR] Failed to fetch HTML: %v", err)
		scrapeSuccess.Set(0)
		return
	}

	htmlStr, _ := doc.Html()
	allDocs := append([]*goquery.Document{doc}, extractCommentTables(htmlStr)...)

	playerCount, teamCount, gkCount := 0, 0, 0

	for _, d := range allDocs {
		// --- Player stats ---
		if d.Find("th[data-stat='player']").Length() > 0 && d.Find("td[data-stat='goals']").Length() > 0 {
			d.Find("tbody tr").Each(func(_ int, s *goquery.Selection) {
				player := strings.TrimSpace(s.Find("td[data-stat='player']").Text())
				team := strings.TrimSpace(s.Find("td[data-stat='team']").Text())
				goals, _ := strconv.ParseFloat(strings.TrimSpace(s.Find("td[data-stat='goals']").Text()), 64)
				assists, _ := strconv.ParseFloat(strings.TrimSpace(s.Find("td[data-stat='assists']").Text()), 64)
				if player != "" && team != "" {
					topScorer.WithLabelValues(player, team).Set(goals)
					topAssists.WithLabelValues(player, team).Set(assists)
					playerCount++
				}
			})
		}

		// --- Goalkeeper clean sheets ---
		if d.Find("th[data-stat='player']").Length() > 0 && d.Find("td[data-stat='clean_sheets']").Length() > 0 {
			d.Find("tbody tr").Each(func(_ int, s *goquery.Selection) {
				player := strings.TrimSpace(s.Find("td[data-stat='player']").Text())
				team := strings.TrimSpace(s.Find("td[data-stat='team']").Text())
				cs, _ := strconv.ParseFloat(strings.TrimSpace(s.Find("td[data-stat='clean_sheets']").Text()), 64)
				if player != "" && team != "" {
					cleanSheets.WithLabelValues(player, team).Set(cs)
					gkCount++
				}
			})
		}

		// --- Team stats ---
		if d.Find("th[data-stat='team']").Length() > 0 && d.Find("td[data-stat='points']").Length() > 0 {
			d.Find("tbody tr").Each(func(_ int, s *goquery.Selection) {
				team := strings.TrimSpace(s.Find("th[data-stat='team']").Text())
				if team == "" {
					return
				}
				points, _ := strconv.ParseFloat(strings.TrimSpace(s.Find("td[data-stat='points']").Text()), 64)
				goalsFor, _ := strconv.ParseFloat(strings.TrimSpace(s.Find("td[data-stat='goals_for']").Text()), 64)
				goalsAgainst, _ := strconv.ParseFloat(strings.TrimSpace(s.Find("td[data-stat='goals_against']").Text()), 64)
				wins, _ := strconv.ParseFloat(strings.TrimSpace(s.Find("td[data-stat='wins']").Text()), 64)
				draws, _ := strconv.ParseFloat(strings.TrimSpace(s.Find("td[data-stat='draws']").Text()), 64)
				losses, _ := strconv.ParseFloat(strings.TrimSpace(s.Find("td[data-stat='losses']").Text()), 64)

				teamPoints.WithLabelValues(team).Set(points)
				teamGoalsFor.WithLabelValues(team).Set(goalsFor)
				teamGoalsAgainst.WithLabelValues(team).Set(goalsAgainst)
				teamWins.WithLabelValues(team).Set(wins)
				teamDraws.WithLabelValues(team).Set(draws)
				teamLosses.WithLabelValues(team).Set(losses)
				teamCount++
			})
		}
	}

	log.Printf("[INFO] Scraped %d players, %d teams, %d goalkeepers", playerCount, teamCount, gkCount)
	scrapeSuccess.Set(1)
}

// --------------------- Exporter Start ---------------------

func startScraping() {
	scrapeFBref()
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			scrapeFBref()
		}
	}()
}

// --------------------- Main ---------------------

func main() {
	const addr = ":2113"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("[FATAL] Port %s already in use: %v", addr, err)
	}
	l.Close()

	log.Printf("[INFO] Starting Premier League metrics exporter on %s", addr)
	startScraping()

	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("[FATAL] HTTP server failed: %v", err)
	}
}
