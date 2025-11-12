FROM prom/prometheus:v2.47.2

# Copy custom config
COPY prometheus.yml /etc/prometheus/prometheus.yml

# Optional: expose config for CI/CD validation
VOLUME ["/etc/prometheus"]
