#!/usr/bin/env python3

# Python file to separate data from pingall csv. 
# This will produce:
# example_files/2_output-result/ping_data.csv
# example_files/2_output-result/movement_data.csv
# example_files/2_output-result/timeframe0/ping_data_movement_0.csv
# example_files/2_output-result/timeframe1/ping_data_movement_1.csv
# example_files/2_output-result/timeframe2/ping_data_movement_2.csv 
import pandas as pd
from pathlib import Path

# Run from: Omen/modules/3_output_visualization
SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT  = SCRIPT_DIR.parent.parent  # Omen/
INPUT_CSV  = REPO_ROOT / "example_files/2_output-result/pingall_full_data.csv"
OUTPUT_DIR = REPO_ROOT / "example_files/2_output-result"

TIMEFRAME_DIRS = {
    0: REPO_ROOT / "example_files/2_output-result/timeframe0",
    1: REPO_ROOT / "example_files/2_output-result/timeframe1",
    2: REPO_ROOT / "example_files/2_output-result/timeframe2",
}

def ensure_dirs(*paths: Path):
    for p in paths:
        p.mkdir(parents=True, exist_ok=True)

def main():
    ensure_dirs(OUTPUT_DIR, *TIMEFRAME_DIRS.values())

    print(f"Reading input file: {INPUT_CSV}")
    df = pd.read_csv(INPUT_CSV)

    # Normalize
    if "data_type" not in df.columns:
        raise ValueError("Expected column 'data_type' not found.")
    df["data_type"] = df["data_type"].astype(str).str.strip().str.lower()

    # Split
    ping_df     = df[df["data_type"] == "ping"].copy()
    movement_df = df[df["data_type"] == "movement"].copy()

    # Save base files (only these two go to the main output folder)
    (OUTPUT_DIR / "ping_data.csv").write_text(ping_df.to_csv(index=False))
    (OUTPUT_DIR / "movement_data.csv").write_text(movement_df.to_csv(index=False))
    print(" Saved ping_data.csv and movement_data.csv to", OUTPUT_DIR)

    # Which column carries the movement index?
    movement_col = None
    for cand in ("movement_number", "movement number"):
        if cand in ping_df.columns:
            movement_col = cand
            break
    if movement_col is None:
        print(" No 'movement_number' (or 'movement number') column — skipping movement splits.")
        return

    # Write ONLY into timeframe folders
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
