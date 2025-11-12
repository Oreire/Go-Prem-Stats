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

