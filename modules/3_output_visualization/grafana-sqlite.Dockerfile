FROM grafana/grafana:latest

# Provisions the sqlite db at container startup (assuming a sqlite database will be mapped to /var/lib/grafana/data.db) as
# well as the dashboards

# Run with: docker run -it --rm -p 3000:3000 3_omen-output-visualizer
# Expects a populated sqlite3 db to be mounted at /var/lib/grafana/data.db

LABEL ARGS="Mount the database: \
-v data.db:/var/lib/grafana/data.db \
-p 3000:3000"

EXPOSE 3000

ENV GF_INSTALL_PLUGINS=frser-sqlite-datasource

# provision the sqlite db and the dashboards
COPY grafana_files/source-sqlite.yaml /etc/grafana/provisioning/datasources/source-sqlite.yaml
COPY grafana_files/dashboards.yaml /etc/grafana/provisioning/dashboards/dashboards.yaml

# place dashboards where they will be provisioned
COPY grafana_files/Dashboard.json /var/lib/grafana/dashboards/
COPY grafana_files/timeframe0.json /var/lib/grafana/dashboards/
COPY grafana_files/timeframe1.json /var/lib/grafana/dashboards/
COPY grafana_files/timeframe2.json /var/lib/grafana/dashboards/
