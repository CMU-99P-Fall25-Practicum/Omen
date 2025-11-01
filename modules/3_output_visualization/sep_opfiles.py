#!/usr/bin/env python3
""""
──────────────────────────────────────────────────────────────────────────────
 P I N G A L L   S E P A R A T O R
──────────────────────────────────────────────────────────────────────────────
WHAT THIS SCRIPT DOES
  • Reads a single pingall CSV (ping + movement rows mixed).
  • Normalizes the `data_type` column and splits rows into:
      - ping_data.csv     (only rows where data_type == "ping")
      - movement_data.csv (only rows where data_type == "movement")
  • Further splits PING rows by movement number (0, 1, 2) and writes:
      - timeframe0/ping_data_movement_0.csv
      - timeframe1/ping_data_movement_1.csv
      - timeframe2/ping_data_movement_2.csv

INTENDED REPO LAYOUT (run from: Omen/modules/3_output_visualization)
  Omen/
    example_files/2_output-result/pingall_full_data.csv           <-- input
    example_files/2_output-result/ping_data.csv                   <-- out
    example_files/2_output-result/movement_data.csv               <-- out
    example_files/2_output-result/timeframe0/ping_data_movement_0.csv  <-- out
    example_files/2_output-result/timeframe1/ping_data_movement_1.csv  <-- out
    example_files/2_output-result/timeframe2/ping_data_movement_2.csv  <-- out

ASSUMPTIONS
  • The input CSV has a `data_type` column with values like "ping" or "movement".
  • The movement index for ping rows is in `movement_number` or `movement number`.
  • Only the two base files (ping_data.csv, movement_data.csv) are written to
    the main 2_output-result folder; the movement splits go ONLY into the
    timeframeN folders (not duplicated next to the base files).

USAGE
  From the repo root:
    cd Omen/modules/3_output_visualization
    python3 sep_opfiles.py
NOTES
  • The script creates directories if they don't exist.
  • If no movement-number column exists, it still writes the two base files
    and skips the per-timeframe splits.
"""
import pandas as pd
from pathlib import Path

# Run from: Omen/modules/3_output_visualization
# Resolve paths relative to thi script so it works on other machines as-is.
SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT  = SCRIPT_DIR.parent.parent  # Omen/
INPUT_CSV  = REPO_ROOT / "example_files/2_output-result/pingall_full_data.csv"
OUTPUT_DIR = REPO_ROOT / "example_files/2_output-result"

# Where to place the per-movement splits
TIMEFRAME_DIRS = {
    0: REPO_ROOT / "example_files/2_output-result/timeframe0",
    1: REPO_ROOT / "example_files/2_output-result/timeframe1",
    2: REPO_ROOT / "example_files/2_output-result/timeframe2",
}

def ensure_dirs(*paths: Path):
    # Create folders if they don't exist (idempotent).
    for p in paths:
        p.mkdir(parents=True, exist_ok=True)

def main():
    # Ensure the output structure is present
    ensure_dirs(OUTPUT_DIR, *TIMEFRAME_DIRS.values())

    print(f"Reading input file: {INPUT_CSV}")
    df = pd.read_csv(INPUT_CSV)

    # Basic schema check + normlization
    if "data_type" not in df.columns:
        raise ValueError("Expected column 'data_type' not found.")
    df["data_type"] = df["data_type"].astype(str).str.strip().str.lower()

    # Split into ping vs movement
    ping_df     = df[df["data_type"] == "ping"].copy()
    movement_df = df[df["data_type"] == "movement"].copy()

    # Write ONLY the two base files into the main output folder.
    (OUTPUT_DIR / "ping_data.csv").write_text(ping_df.to_csv(index=False))
    (OUTPUT_DIR / "movement_data.csv").write_text(movement_df.to_csv(index=False))
    print(" Saved ping_data.csv and movement_data.csv to", OUTPUT_DIR)

    # Identify the movement-number column; support two common spellings
    movement_col = None
    for cand in ("movement_number", "movement number"):
        if cand in ping_df.columns:
            movement_col = cand
            break

    if movement_col is None:
        # No grouping column means we cannot produce per-timeframe files. 
        print(" No 'movement_number' (or 'movement number') column — skipping movement splits.")
        return

    # Create per-timeframe CSVs: ONLY in timeframe folders (not duplicacted in OUTPUT_DIR)
    found_any = False
    for m in (0, 1, 2):
        group = ping_df[ping_df[movement_col] == m]
        if group.empty:
            print(f" No rows for movement {m}; skipping.")
            continue
        found_any = True
        out_path = TIMEFRAME_DIRS[m] / f"ping_data_movement_{m}.csv"
        ensure_dirs(TIMEFRAME_DIRS[m])
        out_path.write_text(group.to_csv(index=False))
        print(f" Movement {m}: {len(group)} rows → {out_path}")

    if not found_any:
        print(" Did not find movement 0/1/2 rows in ping data.")
    else:
        print(" Done. Only timeframe files were created for movement splits.")

if __name__ == "__main__":
    main()
