FROM python:3.12.11-bookworm

LABEL ARGS="Define 4 environment variables: \
-e DB_HOST=127.0.0.1 \
-e DB_PORT=3306 \
-e DB_USER=root \
-e DB_PASS=Practicum26 \
-e DB_NAME=test \
 \
INPUT OPTIONS: \
  1. A folder containing 'nodes.csv' and 'edges.csv' \
  2. A single raw CSV file (ping/pingall data with src,dst,tx,rx,loss_pct,avg_rtt_ms) \
  3. Two explicit CSV files: nodes.csv and edges.csv \
"

RUN mkdir /input

WORKDIR /app
COPY loader.py .
RUN chmod +x loader.py

# install dependencies
RUN pip install pandas mysql-connector-python

ENTRYPOINT [ "/app/loader.py" ]