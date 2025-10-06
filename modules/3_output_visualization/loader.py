#!/usr/bin/env python3
"""
Loader for network graph data into MySQL (for Grafana Node Graph).

INPUT OPTIONS:
  1) A folder containing "nodes.csv" and "edges.csv"
  2) A single raw CSV file (ping/pingall data with columns: src,dst,tx,rx,loss_pct,avg_rtt_ms)  [not implemented here]
  3) Two explicit CSV files: nodes.csv edges.csv

DB connection via environment variables (or defaults below):
  DB_HOST=127.0.0.1
  DB_PORT=3306
  DB_USER=root
  DB_PASS=Practicum26
  DB_NAME=test
"""

import os, sys, csv, pathlib
from typing import Any, Dict, List, Optional
import mysql.connector as mc

DB = dict(
    host=os.getenv("DB_HOST", "127.0.0.1"),
    user=os.getenv("DB_USER", "root"),
    password=os.getenv("DB_PASS", "Practicum26"),
    database=os.getenv("DB_NAME", "test"),
    port=int(os.getenv("DB_PORT", "3306")),
)

CREATE_NODES = """
CREATE TABLE nodes (
  id                       VARCHAR(64)  PRIMARY KEY,
  title                    VARCHAR(128),
  subTitle                 VARCHAR(128) NULL,
  mainStat                 DOUBLE NULL,
  severity                 VARCHAR(32)  NULL,
  `detail__rx_bytes`       BIGINT NULL,
  `detail__rx_packets`     BIGINT NULL,
  `detail__tx_bytes`       BIGINT NULL,
  `detail__tx_packets`     BIGINT NULL,
  `detail__success_rate`   DOUBLE NULL,
  `arc__success`           DOUBLE NULL,
  `arc__errors`            DOUBLE NULL
) ENGINE=InnoDB;
"""

CREATE_EDGES = """
CREATE TABLE edges (
  id       VARCHAR(128) PRIMARY KEY,
  source   VARCHAR(64)  NOT NULL,
  target   VARCHAR(64)  NOT NULL,
  status   VARCHAR(32)  NULL,
  CONSTRAINT fk_edges_source FOREIGN KEY (source) REFERENCES nodes(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT fk_edges_target FOREIGN KEY (target) REFERENCES nodes(id)
    ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB;
"""

NODE_UPSERT = """
INSERT INTO nodes (
  id, title, subTitle, mainStat, severity,
  `detail__rx_bytes`, `detail__rx_packets`, `detail__tx_bytes`, `detail__tx_packets`,
  `detail__success_rate`, `arc__success`, `arc__errors`
)
VALUES (%s,%s,%s,%s,%s, %s,%s,%s,%s, %s,%s,%s)
AS new
ON DUPLICATE KEY UPDATE
  title=new.title,
  subTitle=new.subTitle,
  mainStat=new.mainStat,
  severity=new.severity,
  `detail__rx_bytes`=new.`detail__rx_bytes`,
  `detail__rx_packets`=new.`detail__rx_packets`,
  `detail__tx_bytes`=new.`detail__tx_bytes`,
  `detail__tx_packets`=new.`detail__tx_packets`,
  `detail__success_rate`=new.`detail__success_rate`,
  `arc__success`=new.`arc__success`,
  `arc__errors`=new.`arc__errors`;
"""

EDGE_UPSERT = """
INSERT INTO edges (id, source, target, status)
VALUES (%s, %s, %s, %s)
AS new
ON DUPLICATE KEY UPDATE
  source=new.source,
  target=new.target,
  status=new.status;
"""

def _read_csv(path: pathlib.Path) -> List[Dict[str, Any]]:
    with open(path, newline="", encoding="utf-8") as f:
        return list(csv.DictReader(f))

def _to_int(v: Any) -> Optional[int]:
    try:
        if v in (None, "", "null", "NULL"):
            return None
        return int(float(v))
    except (TypeError, ValueError):
        return None

def _to_float(v: Any) -> Optional[float]:
    try:
        if v in (None, "", "null", "NULL"):
            return None
        return float(v)
    except (TypeError, ValueError):
        return None

def _guess_subtitle(node_id: Optional[str]) -> str:
    if not node_id: return "network node"
    nid = node_id.lower()
    if nid.startswith("ap"):  return "access point"
    if nid.startswith("sta"): return "station"
    return "network node"

def _severity_from_success(p: Optional[float]) -> str:
    if p is None: return "unknown"
    if p >= 0.9:  return "ok"
    if p >= 0.6:  return "warning"
    return "critical"

def normalize_node(row: Dict[str, Any]) -> List[Any]:
    # input CSV expected cols: id,title,rx_bytes,rx_packets,tx_bytes,tx_packets,success_pct_rate
    node_id = row.get("id")
    title = row.get("title") or node_id
    sub_title = row.get("subTitle") or _guess_subtitle(node_id)

    rx_b = _to_int(row.get("rx_bytes"))
    rx_p = _to_int(row.get("rx_packets"))
    tx_b = _to_int(row.get("tx_bytes"))
    tx_p = _to_int(row.get("tx_packets"))
    succ = _to_float(row.get("success_pct_rate"))  # e.g., 0.60

    main_stat = succ
    severity  = _severity_from_success(succ)
    arc_success = succ if succ is not None else None
    arc_errors  = (1.0 - succ) if succ is not None else None

    return [
        node_id, title, sub_title, main_stat, severity,
        rx_b, rx_p, tx_b, tx_p,
        succ, arc_success, arc_errors
    ]

def _src_from_row(row: Dict[str, Any]) -> Optional[str]:
    return row.get("source") or row.get("sourse") or row.get("src")

def normalize_edge(row: Dict[str, Any]) -> Optional[List[Any]]:
    src = _src_from_row(row)
    tgt = row.get("target") or row.get("destination") or row.get("dst")
    if not src or not tgt: return None
    return [ row.get("id") or f"{src}-{tgt}", src, tgt, row.get("status") or "up" ]

def ensure_schema(cnx):
    cur = cnx.cursor()
    cur.execute("SET FOREIGN_KEY_CHECKS=0;")
    cur.execute("DROP TABLE IF EXISTS edges;")
    cur.execute("DROP TABLE IF EXISTS nodes;")
    cur.execute("SET FOREIGN_KEY_CHECKS=1;")
    cur.execute(CREATE_NODES)
    cur.execute(CREATE_EDGES)
    cnx.commit()
    cur.close()

def load_folder(path: pathlib.Path):
    nodes_p, edges_p = path / "nodes.csv", path / "edges.csv"
    nodes = [normalize_node(r) for r in _read_csv(nodes_p)] if nodes_p.exists() else []
    edges = [normalize_edge(r) for r in _read_csv(edges_p)] if edges_p.exists() else []
    edges = [e for e in edges if e]
    return nodes, edges

def load_two_files(nodes_path: pathlib.Path, edges_path: pathlib.Path):
    nodes = [normalize_node(r) for r in _read_csv(nodes_path)]
    edges = [normalize_edge(r) for r in _read_csv(edges_path)]
    edges = [e for e in edges if e]
    return nodes, edges

def main(argv: List[str]):
    if not (len(argv) == 1 or len(argv) == 2):
        print(__doc__); sys.exit(1)

    cnx = mc.connect(**DB)
    ensure_schema(cnx)
    cur = cnx.cursor()

    if len(argv) == 1:
        p = pathlib.Path(argv[0])
        if p.is_dir():
            nodes, edges = load_folder(p)
        else:
            print("If passing one argument, it must be a folder with nodes.csv and edges.csv"); sys.exit(2)
    else:
        nodes, edges = load_two_files(pathlib.Path(argv[0]), pathlib.Path(argv[1]))

    if nodes: cur.executemany(NODE_UPSERT, nodes)
    if edges: cur.executemany(EDGE_UPSERT, edges)

    cnx.commit()
    cur.close(); cnx.close()
    print(f"Loaded {len(nodes)} nodes and {len(edges)} edges into DB '{DB['database']}'.")
    
if __name__ == "__main__":
    main(sys.argv[1:])
