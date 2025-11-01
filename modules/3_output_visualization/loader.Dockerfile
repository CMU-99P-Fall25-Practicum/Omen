FROM python:3.12.11-bookworm

LABEL ARGS="\
INPUT: \
 A directory of the form: \
 some_directory/ \
 ├── timeframe0/ \
 │   ├── nodes.csv \
 │   └── edges.csv \
 ├── timeframe1/ \
 │   ├── nodes.csv \
 │   └── edges.csv \
 ├── timeframe2/ \
 │   ├── nodes.csv \
 │   └── edges.csv \
 ├── ... \
 ├── timeframeX/ \
 │   ├── nodes.csv \
 │   └── edges.csv \
 ├── pingall_data.csv \
 └── movements.csv  \
"

RUN mkdir /input

WORKDIR /app
COPY loader.py .
RUN chmod +x loader.py

# install dependencies
RUN pip install pandas mysql-connector-python

ENTRYPOINT [ "/app/loader.py" ]