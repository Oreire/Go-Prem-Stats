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

var (
	topScorer = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "premier_league_top_scorer_goals",
			Help: "Goals scored by Premier League players",
		},
		[]string{"player", "team"},
	)

	topAssists = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "premier_league_top_assists",
			Help: "Assists made by Premier League players",
		},
		[]string{"player", "team"},
	)

	scrapeSuccess = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "fbref_scrape_success",
			Help: "Whether the last scrape of fbref.com succeeded (1 = success, 0 = fail)",
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
	prometheus.MustRegister(topScorer)
	prometheus.MustRegister(topAssists)
	prometheus.MustRegister(scrapeSuccess)
	prometheus.MustRegister(scrapeDuration)
}

func scrapeFBref() {
	start := time.Now()
	defer func() {
		scrapeDuration.Set(time.Since(start).Seconds())
	}()

	log.Println("[INFO] Starting scrape of FBref Premier League stats")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get("https://fbref.com/en/comps/9/Premier-League-Stats")
	if err != nil {
		log.Printf("[ERROR] HTTP request failed: %v", err)
		scrapeSuccess.Set(0)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("[ERROR] Received non-200 status: %d", resp.StatusCode)
		scrapeSuccess.Set(0)
		return
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to parse document: %v", err)
		scrapeSuccess.Set(0)
		return
	}

	// Reset metrics to avoid stale labels
	topScorer.Reset()
	topAssists.Reset()

	rowCount := 0

	// ✅ Corrected selector: table ID is now stats_standard_9
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

		rowCount++
	})

	log.Printf("[INFO] Successfully scraped %d player rows", rowCount)
	scrapeSuccess.Set(1)
}

func startScraping() {
	// Run once immediately
	scrapeFBref()

	// Schedule periodic scraping
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
		log.Fatalf("[FATAL] Server exited: %v", err)
	}
}
