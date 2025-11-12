FROM grafana/grafana:latest

# Copy dashboards into Grafana provisioning path
COPY dashboards /var/lib/grafana/dashboards

# Optional: preload dashboard config
ENV GF_SECURITY_ADMIN_PASSWORD=admin
ENV GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_PATH=/var/lib/grafana/dashboards/premier_league.json
