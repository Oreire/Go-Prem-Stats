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
	cleanSheets = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "premier_league_goalkeeper_clean_sheets",
			Help: "Number of clean sheets by each goalkeeper",
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
	prometheus.MustRegister(topScorer, topAssists, cleanSheets)
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

	// Reset metrics before new scrape
	topScorer.Reset()
	topAssists.Reset()
	cleanSheets.Reset()
	teamPoints.Reset()
	teamGoalsFor.Reset()
	teamGoalsAgainst.Reset()
	teamWins.Reset()
	teamDraws.Reset()
	teamLosses.Reset()

	playerCount, teamCount, gkCount := 0, 0, 0

	// --- PLAYER STATS ---
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

	// --- TEAM STATS ---
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

	// --- GOALKEEPER CLEAN SHEETS ---
	doc.Find("table#stats_keeper_9 tbody tr").Each(func(i int, s *goquery.Selection) {
		player := strings.TrimSpace(s.Find("td[data-stat='player']").Text())
		team := strings.TrimSpace(s.Find("td[data-stat='team']").Text())
		cs := strings.TrimSpace(s.Find("td[data-stat='clean_sheets']").Text())

		if player == "" || team == "" || cs == "" {
			return
		}

		if c, err := strconv.ParseFloat(cs, 64); err == nil {
			cleanSheets.WithLabelValues(player, team).Set(c)
			gkCount++
		}
	})

	log.Printf("[INFO] Scraped %d players, %d teams, %d goalkeepers", playerCount, teamCount, gkCount)
	scrapeSuccess.Set(1)
}

func startScraping() {
	scrapeFBref()
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
