#!/usr/bin/env python3
"""
Loader for network graph data into MySQL (for Grafana Node Graph).
INPUT OPTIONS:
  1. A folder containing "nodes.csv" and "edges.csv"
  2. A single raw CSV file (ping/pingall data with src,dst,tx,rx,loss_pct,avg_rtt_ms)
  3. Two explicit CSV files: nodes.csv and edges.csv

DB connection via environment variables:
  DB_HOST=127.0.0.1
  DB_PORT=3306
  DB_USER=root
  DB_PASS=Practicum26
  DB_NAME=test
"""

import os, sys, csv, pathlib
import pandas as pd
import mysql.connector as mc
from typing import Any, Dict, List, Optional

# DB Config
DB = dict(
    host=os.getenv("DB_HOST", "127.0.0.1"),
    user=os.getenv("DB_USER", "root"),
    password=os.getenv("DB_PASS", "Practicum26"),
    database=os.getenv("DB_NAME", "test"),
    port=int(os.getenv("DB_PORT", "3306")),
)

# Table schemas
NODE_COLS = [
    "id", "title", "sub_title", "main_stat", "severity",
    "detail__tx", "detail__rx", "detail__loss_pct", "detail__avg_rtt_ms",
]
EDGE_COLS = [
    "id", "source", "target", "main_stat", "status",
    "detail__tx", "detail__rx", "detail__loss_pct", "detail__avg_rtt_ms",
]

CREATE_NODES = """
CREATE TABLE nodes (
  id         VARCHAR(64)  PRIMARY KEY,
  title      VARCHAR(128),
  sub_title  VARCHAR(128),
  main_stat  DOUBLE NULL,
  severity   VARCHAR(32),
  `detail__tx`        DOUBLE NULL,
  `detail__rx`        DOUBLE NULL,
  `detail__loss_pct`  DOUBLE NULL,
  `detail__avg_rtt_ms` DOUBLE NULL
) ENGINE=InnoDB;
"""

CREATE_EDGES = """
CREATE TABLE edges (
  id         VARCHAR(128) PRIMARY KEY,
  source     VARCHAR(64)  NOT NULL,
  target     VARCHAR(64)  NOT NULL,
  main_stat  DOUBLE NULL,
  status     VARCHAR(32),
  `detail__tx`        DOUBLE NULL,
  `detail__rx`        DOUBLE NULL,
  `detail__loss_pct`  DOUBLE NULL,
  `detail__avg_rtt_ms` DOUBLE NULL,
  CONSTRAINT fk_edges_source FOREIGN KEY (source) REFERENCES nodes(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_edges_target FOREIGN KEY (target) REFERENCES nodes(id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;
"""

# Helpers
def to_float(val: Any) -> Optional[float]:
    try:
        return float(val) if val not in (None, "", "null", "NULL") else None
    except Exception:
        return None

def normalize_node(d: Dict[str, Any]) -> List[Any]:
    return [
        d.get("id"),
        d.get("title"),
        d.get("sub_title"),
        to_float(d.get("main_stat")),
        d.get("severity"),
        to_float(d.get("detail__tx")),
        to_float(d.get("detail__rx")),
        to_float(d.get("detail__loss_pct")),
        to_float(d.get("detail__avg_rtt_ms")),
    ]

def normalize_edge(d: Dict[str, Any]) -> List[Any]:
    return [
        d.get("id"),
        d.get("source"),
        d.get("target"),
        to_float(d.get("main_stat")),
        d.get("status"),
        to_float(d.get("detail__tx")),
        to_float(d.get("detail__rx")),
        to_float(d.get("detail__loss_pct")),
        to_float(d.get("detail__avg_rtt_ms")),
    ]

# Input modes 
def load_csv_folder(path: pathlib.Path):
    def read(name: str):
        f = path / name
        return list(csv.DictReader(open(f))) if f.exists() else []
    return [normalize_node(r) for r in read("nodes.csv")], \
           [normalize_edge(r) for r in read("edges.csv")]

def load_raw_csv(path: pathlib.Path):
    df = pd.read_csv(path)
    nodes = pd.unique(df[['src','dst']].values.ravel('K'))
    node_rows = []
    for n in nodes:
        subdf = df[(df['src']==n) | (df['dst']==n)]
        node_rows.append(normalize_node({
            "id": n,
            "title": n,
            "sub_title": "rawcsv",
            "severity": "ok",
            "detail__tx": subdf['tx'].sum(),
            "detail__rx": subdf['rx'].sum(),
            "detail__loss_pct": subdf['loss_pct'].mean(),
            "detail__avg_rtt_ms": subdf['avg_rtt_ms'].mean(),
        }))
    edge_rows = []
    for _,r in df.iterrows():
        edge_rows.append(normalize_edge({
            "id": f"{r.src}-{r.dst}",
            "source": r.src,
            "target": r.dst,
            "status": "up" if r.loss_pct==0 else "loss",
            "detail__tx": r.tx,
            "detail__rx": r.rx,
            "detail__loss_pct": r.loss_pct,
            "detail__avg_rtt_ms": r.avg_rtt_ms,
        }))
    return node_rows, edge_rows

def load_two_csvs(nodes_path: pathlib.Path, edges_path: pathlib.Path):
    with open(nodes_path, newline="", encoding="utf-8") as f:
        nodes_raw = list(csv.DictReader(f))
    with open(edges_path, newline="", encoding="utf-8") as f:
        edges_raw = list(csv.DictReader(f))
    return [normalize_node(r) for r in nodes_raw], \
           [normalize_edge(r) for r in edges_raw]

# DB ops
NODE_UPSERT = f"""
INSERT INTO nodes ({", ".join(NODE_COLS)})
VALUES ({", ".join(["%s"]*len(NODE_COLS))}) AS new
ON DUPLICATE KEY UPDATE
  title=new.title,
  sub_title=new.sub_title,
  main_stat=new.main_stat,
  severity=new.severity,
  `detail__tx`=new.`detail__tx`,
  `detail__rx`=new.`detail__rx`,
  `detail__loss_pct`=new.`detail__loss_pct`,
  `detail__avg_rtt_ms`=new.`detail__avg_rtt_ms`;
"""
EDGE_UPSERT = f"""
INSERT INTO edges ({", ".join(EDGE_COLS)})
VALUES ({", ".join(["%s"]*len(EDGE_COLS))}) AS new
ON DUPLICATE KEY UPDATE
  source=new.source,
  target=new.target,
  main_stat=new.main_stat,
  status=new.status,
  `detail__tx`=new.`detail__tx`,
  `detail__rx`=new.`detail__rx`,
  `detail__loss_pct`=new.`detail__loss_pct`,
  `detail__avg_rtt_ms`=new.`detail__avg_rtt_ms`;
"""

def ensure_schema(cnx):
    cur = cnx.cursor()
    # Drop old tables first
    cur.execute("DROP TABLE IF EXISTS edges;")
    cur.execute("DROP TABLE IF EXISTS nodes;")
    # Recreate with updated schema
    cur.execute(CREATE_NODES)
    cur.execute(CREATE_EDGES)
    cnx.commit()
    cur.close()

# Main 
def main(args: List[str]):
    cnx = mc.connect(**DB)
    ensure_schema(cnx)
    cur = cnx.cursor()

    if len(args) == 1:
        path = pathlib.Path(args[0])
        if path.is_dir():
            nodes, edges = load_csv_folder(path)
        elif path.is_file() and path.suffix.lower()==".csv":
            nodes, edges = load_raw_csv(path)
        else:
            print("Unsupported input format")
            sys.exit(2)
    elif len(args) == 2:
        nodes_path, edges_path = map(pathlib.Path, args)
        nodes, edges = load_two_csvs(nodes_path, edges_path)
    else:
        print(__doc__)
        sys.exit(1)

    if nodes: cur.executemany(NODE_UPSERT, nodes)
    if edges: cur.executemany(EDGE_UPSERT, edges)
    cnx.commit()
    cur.close(); cnx.close()
    print(f"Loaded {len(nodes)} nodes, {len(edges)} edges into DB {DB['database']}.")

if __name__=="__main__":
    main(sys.argv[1:])
