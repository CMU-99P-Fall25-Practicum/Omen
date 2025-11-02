#!/usr/bin/env python3
"""
──────────────────────────────────────────────────────────────────────────────
  O M E N   L O A D E R   (SQLite ⇄ Grafana)
──────────────────────────────────────────────────────────────────────────────
WHAT THIS DOES
  One CLI with two subcommands:
    1) graph
       - Loads up to THREE (nodes.csv, edges.csv) pairs into SQLite with a 
         per-set prefix, e.g. <prefix>_nodes and <prefix>_edges
       - Optionally ingests a per-set timeseries CSV (e.g ping_data_movement_0.csv)
         into <prefix>_timeseries
       - Auto-derives latitudes/longitude from 'position+"x,y,z" ' if present
    2) timeseries
       - Loads a single CSV into a table; can also create an aggregated table
         averaged by a column (e.g. movement_number) as <table>_agg

EXPECTED CSV SHAPES
  nodes.csv: id,title,subTitle,position,rx_bytes,rx_packets,tx_bytes,tx_packets,success_pct_rate,latitude,longitude
             (position is "x,y,z" in meters; lat/lon optional fallback)
  edges.csv: id,source,target,status  (id optional; source/dest also accepted)
  ts.csv   : any flat table (e.g., ping_data_movement_*.csv)

NAMING / SCHEMA
  Per set (prefix = "netA" | "netB" | "netC" ...):
      <prefix>_nodes(id TEXT PK, title TEXT, subTitle TEXT,
                     mainStat REAL, severity TEXT,
                     detail__rx_bytes INTEGER, detail__rx_packets INTEGER,
                     detail__tx_bytes INTEGER, detail__tx_packets INTEGER,
                     detail__success_rate REAL, arc__success REAL, arc__errors REAL,
                     latitude REAL, longitude REAL)
      <prefix>_edges(id TEXT PK, source TEXT, target TEXT, status TEXT)
      <prefix>_timeseries(...)   # columns taken as-is from CSV

USAGE (run from: Omen/modules/3_output_visualization): 
  python3 omenloader.py graph \
  --db /opt/homebrew/var/lib/grafana/omen.db \
  --recreate \
  --root ../../example_files/2_output-result \
  --set1-prefix netA --set1-dir timeframe0 --set1-ts timeframe0/ping_data_movement_0.csv \
  --set2-prefix netB --set2-dir timeframe1 --set2-ts timeframe1/ping_data_movement_1.csv \
  --set3-prefix netC --set3-dir timeframe2 --set3-ts timeframe2/ping_data_movement_2.csv && \
python3 omenloader.py timeseries \
  --root ../../example_files/2_output-result \
  --csv ping_data.csv \
  --db /opt/homebrew/var/lib/grafana/omen.db \
  --table ping_data \
  --if-exists replace \
  --aggregate-by movement_number
"""

import argparse
import csv
import math
import sqlite3
from pathlib import Path
from typing import Optional, Tuple, Union

import pandas as pd

DEFAULT_DB = "/opt/homebrew/var/lib/grafana/omen.db"

# ------------------------ Common helpers ------------------------

def ensure_parent(path: Path):
    # Ensure the parent directory of a file path exists (safe dor SQLite files).
    path.parent.mkdir(parents=True, exist_ok=True)

def open_db(db_path: Path) -> sqlite3.Connection:
    # Open SQLite with sane defaults for Grafana usage (WAL + FKs).
    ensure_parent(db_path)
    conn = sqlite3.connect(str(db_path))
    conn.execute("PRAGMA journal_mode=WAL;")
    conn.execute("PRAGMA foreign_keys=ON;")
    return conn

def resolve_path(p: Union[str, Path], root: Path) -> Path:
    # Resolve a possibly-relative path against --root.
    p = Path(p)
    return p if p.is_absolute() else (root / p)

def qident(name: str) -> str:
    # Quote an identifier for SQLite (avoids clashes / reserved words).
    return '"' + name.replace('"', '""') + '"'

def to_int(s: Optional[str]) -> Optional[int]:
    # Best-effort int parsing with None/null/'' tolerance.
    if s is None or s == "" or str(s).lower() == "null":
        return None
    try:
        return int(float(s))
    except ValueError:
        return None

def to_float(s: Optional[str]) -> Optional[float]:
    # Best-effort float parsing with None/null./'' tolerance.
    if s is None or s == "" or str(s).lower() == "null":
        return None
    try:
        return float(s)
    except ValueError:
        return None

# ------------------------ Geo helpers (graph) ------------------------

def parse_position_xyz(pos: Optional[str]) -> Tuple[Optional[float], Optional[float], Optional[float]]:
    # Parse 'x,y,z' (meters) → floats. Returns (None, None, None) is missing/invalid.
    if not pos or "," not in pos:
        return None, None, None
    try:
        parts = [p.strip() for p in pos.split(",")]
        x = float(parts[0])
        y = float(parts[1]) if len(parts) > 1 else 0.0
        z = float(parts[2]) if len(parts) > 2 else 0.0
        return x, y, z
    except Exception:
        return None, None, None

def cartesian_to_geo(x_m: float, y_m: float, base_lat: float, base_lon: float) -> Tuple[float, float]:
    # Convert local (x east, y north) meters into lat/lon degrees around a base origin.
    # Good enough for small campus-scale offsets.
    meters_per_deg_lat = 111_320.0
    meters_per_deg_lon = 111_320.0 * math.cos(math.radians(base_lat))
    lat = base_lat + (y_m / meters_per_deg_lat)
    lon = base_lon + (x_m / meters_per_deg_lon)
    return lat, lon

def guess_subtitle(node_id: Optional[str]) -> str:
    # Cheap labeler for node type; used if subTitle not provided. 
    if not node_id:
        return "network node"
    nid = node_id.lower()
    if nid.startswith("ap"):
        return "access point"
    if nid.startswith("sta"):
        return "station"
    return "network node"

def severity_from_success(p: Optional[float]) -> str:
    # Map success ratio to severity string used by Node Graph styling. 
    if p is None:
        return "unknown"
    if p >= 0.9:
        return "ok"
    if p >= 0.6:
        return "warning"
    return "critical"

# ------------------------ GRAPH: schema + ingest ------------------------

def drop_and_create_schema_for_prefix(conn: sqlite3.Connection, prefix: str):
    # Drop and recreate <prefix>_nodes and <prefix>_edges.
    cur = conn.cursor()
    nodes_tbl = f"{prefix}_nodes"
    edges_tbl = f"{prefix}_edges"
    cur.execute(f"DROP TABLE IF EXISTS {qident(edges_tbl)};")
    cur.execute(f"DROP TABLE IF EXISTS {qident(nodes_tbl)};")
    cur.execute(f"""
    CREATE TABLE {qident(nodes_tbl)} (
        id                   TEXT PRIMARY KEY,
        title                TEXT,
        subTitle             TEXT,
        mainStat             REAL,
        severity             TEXT,
        detail__rx_bytes     INTEGER,
        detail__rx_packets   INTEGER,
        detail__tx_bytes     INTEGER,
        detail__tx_packets   INTEGER,
        detail__success_rate REAL,
        arc__success         REAL,
        arc__errors          REAL,
        latitude             REAL,
        longitude            REAL
    );""")
    cur.execute(f"""
    CREATE TABLE {qident(edges_tbl)} (
        id      TEXT PRIMARY KEY,
        source  TEXT NOT NULL,
        target  TEXT NOT NULL,
        status  TEXT,
        FOREIGN KEY(source) REFERENCES {qident(nodes_tbl)}(id) ON DELETE CASCADE ON UPDATE CASCADE,
        FOREIGN KEY(target) REFERENCES {qident(nodes_tbl)}(id) ON DELETE CASCADE ON UPDATE CASCADE
    );""")
    conn.commit()

def ensure_schema_for_prefix(conn: sqlite3.Connection, prefix: str):
    # Create <prefix>_nodes and <prefix>_edges if they do not exist.
    cur = conn.cursor()
    nodes_tbl = f"{prefix}_nodes"
    edges_tbl = f"{prefix}_edges"
    cur.execute(f"""
    CREATE TABLE IF NOT EXISTS {qident(nodes_tbl)} (
        id                   TEXT PRIMARY KEY,
        title                TEXT,
        subTitle             TEXT,
        mainStat             REAL,
        severity             TEXT,
        detail__rx_bytes     INTEGER,
        detail__rx_packets   INTEGER,
        detail__tx_bytes     INTEGER,
        detail__tx_packets   INTEGER,
        detail__success_rate REAL,
        arc__success         REAL,
        arc__errors          REAL,
        latitude             REAL,
        longitude            REAL
    );""")
    cur.execute(f"""
    CREATE TABLE IF NOT EXISTS {qident(edges_tbl)} (
        id      TEXT PRIMARY KEY,
        source  TEXT NOT NULL,
        target  TEXT NOT NULL,
        status  TEXT
    );""")
    conn.commit()

def ingest_nodes(conn: sqlite3.Connection, prefix: str, csv_path: Path,
                 base_lat: float, base_lon: float, prefer_pos_over_latlon: bool = True) -> int:
    
    # Insert/UPSERT rows from nodes.csv into <prefix>_nodes.
    # - Derives (lat,lon) from 'position' when available; falls bck to CSV lat/lon.
    # - Computes mainStat/severity/arcs from sucess_pct_rate for Node Graph visuals.
    table = f"{prefix}_nodes"
    cur = conn.cursor()
    count = 0
    with csv_path.open(newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            nid = row.get("id")
            if not nid:
                continue
            title = row.get("title") or nid
            sub_title = row.get("subTitle") or guess_subtitle(nid)
            rx_b = to_int(row.get("rx_bytes"))
            rx_p = to_int(row.get("rx_packets"))
            tx_b = to_int(row.get("tx_bytes"))
            tx_p = to_int(row.get("tx_packets"))
            succ = to_float(row.get("success_pct_rate"))
            main_stat = succ
            severity = severity_from_success(succ)
            arc_success = succ
            arc_errors = (1.0 - succ) if succ is not None else None
            
            # Choose coordinate source: position → (lat,lon) else provided lat/lon
            lat = None
            lon = None
            x, y, _ = parse_position_xyz(row.get("position"))
            if prefer_pos_over_latlon and x is not None and y is not None:
                lat, lon = cartesian_to_geo(x, y, base_lat, base_lon)
            else:
                lat = to_float(row.get("latitude"))
                lon = to_float(row.get("longitude"))
                if (lat is None or lon is None) and x is not None and y is not None:
                    lat, lon = cartesian_to_geo(x, y, base_lat, base_lon)

            cur.execute(f"""
            INSERT INTO {qident(table)} (
              id, title, subTitle, mainStat, severity,
              detail__rx_bytes, detail__rx_packets, detail__tx_bytes, detail__tx_packets,
              detail__success_rate, arc__success, arc__errors, latitude, longitude
            ) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
            ON CONFLICT(id) DO UPDATE SET
              title=excluded.title,
              subTitle=excluded.subTitle,
              mainStat=excluded.mainStat,
              severity=excluded.severity,
              detail__rx_bytes=excluded.detail__rx_bytes,
              detail__rx_packets=excluded.detail__rx_packets,
              detail__tx_bytes=excluded.detail__tx_bytes,
              detail__tx_packets=excluded.detail__tx_packets,
              detail__success_rate=excluded.detail__success_rate,
              arc__success=excluded.arc__success,
              arc__errors=excluded.arc__errors,
              latitude=excluded.latitude,
              longitude=excluded.longitude;
            """, (nid, title, sub_title, main_stat, severity,
                  rx_b, rx_p, tx_b, tx_p, succ, arc_success, arc_errors, lat, lon))
            count += 1
    conn.commit()
    return count

def ingest_edges(conn: sqlite3.Connection, prefix: str, csv_path: Path) -> int:
    # Insert/UPSERT rows from edges.csv into <prefix>_edges.
    # - If id missing, derive "source-target".
    # - Accepts 'source' | 'src' and 'target' | 'dst' naming variants.
    table = f"{prefix}_edges"
    cur = conn.cursor()
    count = 0
    with csv_path.open(newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            edge_id = row.get("id")
            src = row.get("source") or row.get("sourse") or row.get("src")
            tgt = row.get("target") or row.get("destination") or row.get("dst")
            status = row.get("status") or "up"
            if not src or not tgt:
                continue
            if not edge_id:
                edge_id = f"{src}-{tgt}"
            cur.execute(f"""
            INSERT INTO {qident(table)} (id, source, target, status)
            VALUES (?,?,?,?)
            ON CONFLICT(id) DO UPDATE SET
              source=excluded.source,
              target=excluded.target,
              status=excluded.status;
            """, (edge_id, src, tgt, status))
            count += 1
    conn.commit()
    return count

def ingest_timeseries_raw(conn: sqlite3.Connection, table: str, csv_path: Path,
                          if_exists: str = "replace") -> int:
    # Load any CSV as-is into a table (user per-set movement series).
    df = pd.read_csv(csv_path)
    df.to_sql(table, conn, if_exists=if_exists, index=False)
    return len(df)

# ------------------------ Subcommand: graph ------------------------

def add_graph_args(sp: argparse.ArgumentParser):
    # CLI arguments for the graph subcommand (supports up to 3 sets).
    sp.add_argument("--db", default=DEFAULT_DB, help=f"SQLite DB path (default: {DEFAULT_DB})")
    sp.add_argument("--recreate", action="store_true", help="Drop & recreate tables for any provided set")
    sp.add_argument("--root", type=Path, default=Path(__file__).resolve().parent,
                    help="Base directory to resolve relative CSV paths (default: script folder)")
    for i in (1, 2, 3):
        sp.add_argument(f"--set{i}-prefix", help=f"Table prefix for set {i}")
        sp.add_argument(f"--set{i}-dir", type=Path, help=f"Directory containing nodes.csv and edges.csv for set {i} (and optionally ping_data_movement_*.csv)")
        sp.add_argument(f"--set{i}-nodes", type=Path, help=f"nodes.csv for set {i}")
        sp.add_argument(f"--set{i}-edges", type=Path, help=f"edges.csv for set {i}")
        sp.add_argument(f"--set{i}-ts", type=Path, help=f"timeseries CSV for set {i} (e.g., ping_data_movement_0.csv)")
        sp.add_argument(f"--set{i}-ts-table", help=f"Destination table for set {i} timeseries (default: <prefix>_timeseries)")
        sp.add_argument(f"--set{i}-pos-base-lat", type=float, default=37.4270, help=f"Base latitude for set {i}")
        sp.add_argument(f"--set{i}-pos-base-lon", type=float, default=-122.1690, help=f"Base longitude for set {i}")

def _auto_detect_timeseries(set_dir: Path) -> Optional[Path]:
    #If a set dir is provided, try to find a ping_data_movement_*.csv inside it.
    # Returns the first match if found, else None.
    if not set_dir:
        return None
    candidates = sorted(set_dir.glob("ping_data_movement_*.csv"))
    return candidates[0] if candidates else None

def run_graph(args: argparse.Namespace):
    # Driver for 'graph': load per-set nodes/edges (+ optional timeseries) into SQLite.
    root = args.root.resolve()
    conn = open_db(Path(args.db))
    used = 0

    def process_set(idx: int):
        # Process one of the up to three graph sets. 
        prefix = getattr(args, f"set{idx}_prefix")
        set_dir = getattr(args, f"set{idx}_dir")
        nodes = getattr(args, f"set{idx}_nodes")
        edges = getattr(args, f"set{idx}_edges")
        ts    = getattr(args, f"set{idx}_ts")
        ts_table = getattr(args, f"set{idx}_ts_table") or (f"{prefix}_timeseries" if prefix else None)

        # Allow dir-driven defaults (nodes/edges, and try to auto-find movement CSV)
        if set_dir and not nodes and not edges:
            nodes = Path(set_dir) / "nodes.csv"
            edges = Path(set_dir) / "edges.csv"
        if set_dir and not ts:
            guess = _auto_detect_timeseries(Path(set_dir))
            if guess:
                ts = guess

        if not any([prefix, nodes, edges, set_dir, ts]):
            return False
        if not prefix:
            raise ValueError(f"--set{idx}-prefix is required when providing CSVs for set {idx}")
        if not nodes or not edges:
            raise ValueError(f"--set{idx}-nodes and --set{idx}-edges are both required for set {idx}")

        nodes_path = resolve_path(nodes, root)
        edges_path = resolve_path(edges, root)
        if not nodes_path.exists():
            raise FileNotFoundError(f"Set {idx}: nodes file not found: {nodes_path}")
        if not edges_path.exists():
            raise FileNotFoundError(f"Set {idx}: edges file not found: {edges_path}")

        # Create/ensure schemas
        if args.recreate:
            drop_and_create_schema_for_prefix(conn, prefix)
        else:
            ensure_schema_for_prefix(conn, prefix)

        # Ingest graph entities
        lat = getattr(args, f"set{idx}_pos_base_lat")
        lon = getattr(args, f"set{idx}_pos_base_lon")

        n = ingest_nodes(conn, prefix, nodes_path, base_lat=lat, base_lon=lon)
        e = ingest_edges(conn, prefix, edges_path)

        # Optional per-set timeseries 
        if ts:
            ts_path = resolve_path(ts, root)
            if not ts_path.exists():
                raise FileNotFoundError(f"Set {idx}: timeseries file not found: {ts_path}")
            ts_tbl = ts_table or f"{prefix}_timeseries"
            rows = ingest_timeseries_raw(conn, ts_tbl, ts_path, if_exists="replace")
            print(f"[{prefix}] loaded timeseries table={ts_tbl} rows={rows}")

        # Helpful indexes for Grafana queries
        cur = conn.cursor()
        cur.execute(f"CREATE INDEX IF NOT EXISTS {qident(f'idx_{prefix}_nodes_id')} ON {qident(prefix+'_nodes')}(id);")
        cur.execute(f"CREATE INDEX IF NOT EXISTS {qident(f'idx_{prefix}_edges_src')} ON {qident(prefix+'_edges')}(source);")
        cur.execute(f"CREATE INDEX IF NOT EXISTS {qident(f'idx_{prefix}_edges_tgt')} ON {qident(prefix+'_edges')}(target);")
        conn.commit()

        print(f"[{prefix}] loaded nodes={n}, edges={e}")
        return True
    
    # Process up to three sets
    for i in (1, 2, 3):
        if process_set(i):
            used += 1

    if used == 0:
        print("No sets provided. Use --set{1|2|3}-prefix + (--set{1|2|3}-dir OR --set{1|2|3}-nodes + --set{1|2|3}-edges) and optionally --set{1|2|3}-ts.")
    else:
        print(f"Done. Processed {used} set(s). DB: {args.db}")
    conn.close()

# ------------------------ Subcommand: timeseries ------------------------

def add_timeseries_args(sp: argparse.ArgumentParser):
    # CLI arguments for the standalone timeseries loader/aggregator. 
    sp.add_argument("--csv", required=True, type=Path, help="CSV file (relative to --root if not absolute)")
    sp.add_argument("--db", required=True, default=DEFAULT_DB, help="SQLite database file path")
    sp.add_argument("--table", required=True, help="Destination table name for raw data")
    sp.add_argument("--aggregate-by", default=None, help="Column to group by (e.g., 'movement_number')")
    sp.add_argument("--aggregate-into", default=None, help="Name of aggregated result table (default: <table>_agg)")
    sp.add_argument("--if-exists", choices=["replace", "append", "fail"], default="replace")
    sp.add_argument("--root", type=Path, default=Path(__file__).resolve().parent,
                    help="Base directory to resolve relative CSV paths (default: script folder)")

def run_timeseries(args: argparse.Namespace):
    # Driver for 'timeseries':
    #  - writes raw CSV to --table
    #  - if --aggregate-by provided, also writes aggregated mean to <table>_agg (or --aggregate-into)
    root = args.root.resolve()
    csv_path = resolve_path(args.csv, root)
    if not csv_path.exists():
        raise FileNotFoundError(f"CSV file not found: {csv_path}")

    df = pd.read_csv(csv_path)
    print(f"Loaded CSV with {len(df)} rows and {len(df.columns)} columns from {csv_path}.")

    conn = open_db(Path(args.db))
    
    # Raw table
    df.to_sql(args.table, conn, if_exists=args.if_exists, index=False)
    print(f"Inserted raw table '{args.table}' into {args.db}.")
    
    # Optional aggregation
    if args.aggregate_by:
        if args.aggregate_by not in df.columns:
            raise ValueError(f"Column '{args.aggregate_by}' not found in CSV columns: {list(df.columns)}")
        
        #Keep key column; average numeric columns only
        df_coerced = df.apply(pd.to_numeric, errors="ignore")
        grouped = df_coerced.groupby(args.aggregate_by, as_index=False).mean(numeric_only=True)

        agg_name = args.aggregate_into or f"{args.table}_agg"
        grouped.to_sql(agg_name, conn, if_exists="replace", index=False)
        print(f"Created aggregated table '{agg_name}' grouped by '{args.aggregate_by}' ({len(grouped)} rows).")

    conn.close()
    print("✅ Done.")

# ------------------------ Main entry ------------------------

def main():
    # Top-level CLI dispatcher
    ap = argparse.ArgumentParser(description="Omen unified loader (graph + timeseries) for SQLite/Grafana.")
    sub = ap.add_subparsers(dest="cmd", required=True)
    
    # graph subcommand
    sp_graph = sub.add_parser("graph", help="Load nodes/edges CSVs into prefixed tables (+ optional per-set timeseries)")
    add_graph_args(sp_graph)
    sp_graph.set_defaults(func=run_graph)
    
    # timeseries subcommand
    sp_ts = sub.add_parser("timeseries", help="Load a CSV (and optional aggregated view) into SQLite")
    add_timeseries_args(sp_ts)
    sp_ts.set_defaults(func=run_timeseries)

    args = ap.parse_args()
    args.func(args)

if __name__ == "__main__":
    main()

