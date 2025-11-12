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
            Help: "Goals scored by top scorer",
        },
        []string{"player", "team"},
    )
    topAssists = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "premier_league_top_assists",
            Help: "Assists by top players",
        },
        []string{"player", "team"},
    )
    cleanSheets = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "premier_league_clean_sheets",
            Help: "Clean sheets by goalkeepers",
        },
        []string{"player", "team"},
    )
)

func init() {
    prometheus.MustRegister(topScorer)
    prometheus.MustRegister(topAssists)
    prometheus.MustRegister(cleanSheets)
}

func scrapeFBref() {
    resp, err := http.Get("https://fbref.com/en/comps/9/Premier-League-Stats")
    if err != nil {
        log.Println("Error fetching page:", err)
        return
    }
    defer resp.Body.Close()

    doc, err := goquery.NewDocumentFromReader(resp.Body)
    if err != nil {
        log.Println("Error parsing page:", err)
        return
    }

    doc.Find("table#stats_standard tbody tr").Each(func(i int, s *goquery.Selection) {
        player := strings.TrimSpace(s.Find("td[data-stat='player']").Text())
        team := strings.TrimSpace(s.Find("td[data-stat='team']").Text())
        goals := strings.TrimSpace(s.Find("td[data-stat='goals']").Text())
        assists := strings.TrimSpace(s.Find("td[data-stat='assists']").Text())
        sheets := strings.TrimSpace(s.Find("td[data-stat='clean_sheets']").Text())

        if g, err := strconv.ParseFloat(goals, 64); err == nil {
            topScorer.WithLabelValues(player, team).Set(g)
        }
        if a, err := strconv.ParseFloat(assists, 64); err == nil {
            topAssists.WithLabelValues(player, team).Set(a)
        }
        if cs, err := strconv.ParseFloat(sheets, 64); err == nil {
            cleanSheets.WithLabelValues(player, team).Set(cs)
        }
    })
}

func startScraping() {
    scrapeFBref() // run once immediately
    ticker := time.NewTicker(1 * time.Hour)
    go func() {
        for range ticker.C {
            scrapeFBref()
        }
    }()
}

func main() {
    startScraping()
    http.Handle("/metrics", promhttp.Handler())
    log.Println("Exporter running on :2112/metrics")
    log.Fatal(http.ListenAndServe(":2112", nil))
}
