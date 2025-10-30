#!/usr/bin/env python3
"""
sqliteloader.py

Load up to THREE (nodes.csv, edges.csv) pairs into a SQLite DB.
Each set writes to its own tables using a required prefix:
  <prefix>_nodes, <prefix>_edges

- Derives latitude/longitude from `position="x,y,z"` when present, else uses CSV lat/lon.
- No movement/time series.

Examples:
  cd /Omen/modules/3_output_visualization

  # Using --root + --setN-dir (recommended)
  python3 sqliteloader.py \
    --db /opt/homebrew/var/lib/grafana/omen.db --recreate \
    --root ../../example_files/2_output-result \
    --set1-prefix netA --set1-dir timeframe0 \
    --set2-prefix netB --set2-dir timeframe1 \
    --set3-prefix netC --set3-dir timeframe0

  # Override specific files if needed
  python3 sqliteloader.py \
    --db /opt/homebrew/var/lib/grafana/omen.db \
    --root ../../example_files/2_output-result \
    --set1-prefix netA --set1-nodes timeframe0/nodes.csv --set1-edges timeframe0/edges.csv
"""

import argparse
import csv
import math
import os
import sqlite3
from pathlib import Path
from typing import Optional, Tuple

DEFAULT_DB = "/opt/homebrew/var/lib/grafana/omen.db"

# ---------- Helpers ----------

def ensure_parent(path: Path):
    path.parent.mkdir(parents=True, exist_ok=True)

def open_db(db_path: Path) -> sqlite3.Connection:
    ensure_parent(db_path)
    conn = sqlite3.connect(db_path)
    conn.execute("PRAGMA journal_mode=WAL;")
    conn.execute("PRAGMA foreign_keys=ON;")
    return conn

def qident(name: str) -> str:
    return '"' + name.replace('"', '""') + '"'

def to_int(s: Optional[str]) -> Optional[int]:
    if s is None or s == "" or str(s).lower() == "null":
        return None
    try:
        return int(float(s))
    except ValueError:
        return None

def to_float(s: Optional[str]) -> Optional[float]:
    if s is None or s == "" or str(s).lower() == "null":
        return None
    try:
        return float(s)
    except ValueError:
        return None

def parse_position_xyz(pos: Optional[str]) -> Tuple[Optional[float], Optional[float], Optional[float]]:
    """Parse 'x,y,z' -> (x, y, z) meters."""
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
    """
    Convert local cartesian offsets in meters to (lat, lon) around a base origin.
    x -> east (+), y -> north (+).
    """
    meters_per_deg_lat = 111_320.0
    meters_per_deg_lon = 111_320.0 * math.cos(math.radians(base_lat))
    lat = base_lat + (y_m / meters_per_deg_lat)
    lon = base_lon + (x_m / meters_per_deg_lon)
    return lat, lon

def guess_subtitle(node_id: Optional[str]) -> str:
    if not node_id:
        return "network node"
    nid = node_id.lower()
    if nid.startswith("ap"):
        return "access point"
    if nid.startswith("sta"):
        return "station"
    return "network node"

def severity_from_success(p: Optional[float]) -> str:
    if p is None:
        return "unknown"
    if p >= 0.9:
        return "ok"
    if p >= 0.6:
        return "warning"
    return "critical"

def resolve_path(p: Optional[Path], root: Path) -> Optional[Path]:
    if p is None:
        return None
    p = Path(p)
    return p if p.is_absolute() else (root / p)

# ---------- Schema per prefix ----------

def drop_and_create_schema_for_prefix(conn: sqlite3.Connection, prefix: str):
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
    );
    """)

    cur.execute(f"""
    CREATE TABLE {qident(edges_tbl)} (
        id      TEXT PRIMARY KEY,
        source  TEXT NOT NULL,
        target  TEXT NOT NULL,
        status  TEXT,
        FOREIGN KEY(source) REFERENCES {qident(nodes_tbl)}(id) ON DELETE CASCADE ON UPDATE CASCADE,
        FOREIGN KEY(target) REFERENCES {qident(nodes_tbl)}(id) ON DELETE CASCADE ON UPDATE CASCADE
    );
    """)
    conn.commit()

def ensure_schema_for_prefix(conn: sqlite3.Connection, prefix: str):
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
    );
    """)
    cur.execute(f"""
    CREATE TABLE IF NOT EXISTS {qident(edges_tbl)} (
        id      TEXT PRIMARY KEY,
        source  TEXT NOT NULL,
        target  TEXT NOT NULL,
        status  TEXT
    );
    """)
    conn.commit()

# ---------- Ingest per prefix ----------

def ingest_nodes(conn: sqlite3.Connection, prefix: str, csv_path: Path,
                 base_lat: float, base_lon: float,
                 prefer_position_over_csv_latlon: bool = True) -> int:
    table = f"{prefix}_nodes"
    cur = conn.cursor()
    inserted = 0
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
            severity  = severity_from_success(succ)
            arc_success = succ
            arc_errors  = (1.0 - succ) if succ is not None else None

            # Derive lat/lon from position if present; else use csv lat/lon
            lat = None
            lon = None
            pos = row.get("position")
            x, y, _ = parse_position_xyz(pos)
            if prefer_position_over_csv_latlon and x is not None and y is not None:
                lat, lon = cartesian_to_geo(x, y, base_lat=base_lat, base_lon=base_lon)
            else:
                lat = to_float(row.get("latitude"))
                lon = to_float(row.get("longitude"))
                if (lat is None or lon is None) and x is not None and y is not None:
                    lat, lon = cartesian_to_geo(x, y, base_lat=base_lat, base_lon=base_lon)

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
            """, (
                nid, title, sub_title, main_stat, severity,
                rx_b, rx_p, tx_b, tx_p,
                succ, arc_success, arc_errors, lat, lon
            ))
            inserted += 1
    conn.commit()
    return inserted

def ingest_edges(conn: sqlite3.Connection, prefix: str, csv_path: Path) -> int:
    table = f"{prefix}_edges"
    cur = conn.cursor()
    inserted = 0
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
            inserted += 1
    conn.commit()
    return inserted

# ---------- Driver ----------

def add_set_args(ap: argparse.ArgumentParser, idx: int):
    ap.add_argument(f"--set{idx}-prefix", help=f"Table prefix for set {idx} (required if set{idx}-nodes/edges provided)")
    ap.add_argument(f"--set{idx}-dir", type=Path, help=f"Directory containing nodes.csv and edges.csv for set {idx}")
    ap.add_argument(f"--set{idx}-nodes", type=Path, help=f"nodes.csv for set {idx}")
    ap.add_argument(f"--set{idx}-edges", type=Path, help=f"edges.csv for set {idx}")
    ap.add_argument(f"--set{idx}-pos-base-lat", type=float, default=37.4270, help=f"Base latitude for set {idx}")
    ap.add_argument(f"--set{idx}-pos-base-lon", type=float, default=-122.1690, help=f"Base longitude for set {idx}")

def process_set(conn: sqlite3.Connection, args: argparse.Namespace, idx: int, recreate: bool, root: Path):
    prefix = getattr(args, f"set{idx}_prefix")
    set_dir = getattr(args, f"set{idx}_dir")
    nodes = getattr(args, f"set{idx}_nodes")
    edges = getattr(args, f"set{idx}_edges")

    # If a directory is provided and explicit files are not, auto-pick nodes.csv / edges.csv
    if set_dir and not nodes and not edges:
        nodes = Path(set_dir) / "nodes.csv"
        edges = Path(set_dir) / "edges.csv"
        setattr(args, f"set{idx}_nodes", nodes)
        setattr(args, f"set{idx}_edges", edges)

    # If no inputs for this set, skip silently
    if not any([prefix, nodes, edges, set_dir]):
        return False

    # Validate provided args
    if not prefix:
        raise ValueError(f"--set{idx}-prefix is required when providing CSVs for set {idx}")
    if not nodes or not edges:
        raise ValueError(f"--set{idx}-nodes and --set{idx}-edges are both required for set {idx}")

    # Resolve paths relative to root if not absolute
    nodes_path = resolve_path(Path(nodes), root)
    edges_path = resolve_path(Path(edges), root)

    if not nodes_path.exists():
        raise FileNotFoundError(f"Set {idx}: nodes file not found: {nodes_path}")
    if not edges_path.exists():
        raise FileNotFoundError(f"Set {idx}: edges file not found: {edges_path}")

    if recreate:
        drop_and_create_schema_for_prefix(conn, prefix)
    else:
        ensure_schema_for_prefix(conn, prefix)

    pos_base_lat = getattr(args, f"set{idx}_pos_base_lat")
    pos_base_lon = getattr(args, f"set{idx}_pos_base_lon")

    n = ingest_nodes(conn, prefix, nodes_path, base_lat=pos_base_lat, base_lon=pos_base_lon)
    e = ingest_edges(conn, prefix, edges_path)

    # Indexes
    cur = conn.cursor()
    cur.execute(f"CREATE INDEX IF NOT EXISTS {qident(f'idx_{prefix}_nodes_id')} ON {qident(prefix+'_nodes')}(id);")
    cur.execute(f"CREATE INDEX IF NOT EXISTS {qident(f'idx_{prefix}_edges_src')} ON {qident(prefix+'_edges')}(source);")
    cur.execute(f"CREATE INDEX IF NOT EXISTS {qident(f'idx_{prefix}_edges_tgt')} ON {qident(prefix+'_edges')}(target);")
    conn.commit()

    print(f"[{prefix}] loaded nodes={n}, edges={e}")
    return True

def main():
    ap = argparse.ArgumentParser(description="Ingest up to 3 (nodes,edges) CSV pairs into SQLite with per-set prefixes.")
    ap.add_argument("--db", default=DEFAULT_DB, help=f"SQLite DB path (default: {DEFAULT_DB})")
    ap.add_argument("--recreate", action="store_true", help="Drop & recreate tables for any provided set")
    # root defaults to the script directory (portable)
    default_root = Path(__file__).resolve().parent
    ap.add_argument("--root", type=Path, default=default_root,
                    help="Base directory to resolve relative CSV paths (default: script folder)")
    add_set_args(ap, 1)
    add_set_args(ap, 2)
    add_set_args(ap, 3)
    args = ap.parse_args()

    root = args.root.resolve()
    conn = open_db(Path(args.db))
    used = 0
    for i in (1, 2, 3):
        if process_set(conn, args, i, args.recreate, root):
            used += 1

    if used == 0:
        print("No sets provided. Use --set{1|2|3}-prefix + (--set{1|2|3}-dir OR --set{1|2|3}-nodes + --set{1|2|3}-edges).")
    else:
        print(f"Done. Processed {used} set(s). DB: {args.db}")
    conn.close()

if __name__ == "__main__":
    main()
