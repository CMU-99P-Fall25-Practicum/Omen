FROM grafana/grafana:latest

# Run with: docker run -it --rm -p 3000:3000 3_omen-output-visualizer
# Expects a populated sqlite3 db to be mounted at /var/lib/grafana/data.db

LABEL ARGS="Mount the database: \
-v data.db:/var/lib/grafana/data.db \
-p 3000:3000"

EXPOSE 3000

ENV GF_INSTALL_PLUGINS=frser-sqlite-datasource

COPY ./source-sqlite.yaml /etc/grafana/provisioning/datasources/source-sqlite.yaml