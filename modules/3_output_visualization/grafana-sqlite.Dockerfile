FROM grafana/grafana:latest

# Run with: docker run -it --rm -p 3000:3000 3_omen-output-visualizer

# Grafana will not create a new db, so we need to have something in place
RUN touch /var/lib/grafana/data.db

EXPOSE 3000

ENV GF_INSTALL_PLUGINS=frser-sqlite-datasource

COPY ./source-sqlite.yaml /etc/grafana/provisioning/datasources/source-sqlite.yaml