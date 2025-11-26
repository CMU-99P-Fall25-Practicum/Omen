This file details the I/O contracts between modules, to ensure that they can be swapped easily and run manually.
As long as a module consumes its input contract and its output satisfies the next module's input contract, modules can be freely interchanged.

# Modules

## Input Validation

As it says on the tin, Input Validation modules consume a configuration file and test its parameters for validity so all future modules can assume their inputs are proper.

*In*:
- arg1: Json file (path taken as argument) to be validated. This file contains a topology, a list of positions to move nodes between, propagation model to simulate environmental conditions, and all required metadata (ssh information & credentials, test name, backend, OSM connection information, etc).
  - [Example](example_files/test_run.json).
  - Because JSON does not support inline comments, the fields are documented here:
    - **schemaVersion**: "1.0"
    - **meta**
      - **backend**: "mininet"
      - **name**: Name to use for this run, to distinguish it from other tests. Has no impact on logic.
      - **duration**: *unused*. The maximum duration the actual test script is allowed to run for.
    - **topo**
      - **nets**
        - **noise_th**: sets the noise sensitivity threshold on receiving nodes. Lower thresholds mean more sensitive receivers. See the [README](README.md#noise-threshold) for suggested values.
        - **propagation_model**: selects a propagation model for simulation wireless signal degradation. Alters the energy loss by distance; set this to simulate a mostly free-space environment, a mostly indoor environment, etc. See the [README](README.md#wireless-propagation-models) for more information on supported models and suggested values.
          - **exp**: path-loss exponent (n)
          - **s**: shadowing standard deviation (σ)
      - **aps**: *array*. access points (wireless routers) in the topology
        - **id**: unique identifier for this node. Must have a unique number in it (this is used by Mininet to set a datapath-id).
        - **mode**: IEEE 802.11 mode. "a", "b", "g", "b", "p", "ax", "ac" should all be supported, but only "a" has been thoroughly tested.
        - **channel**: Wi-Fi channel this node is broadcasting on
        - **ssid**: name of the ssid this ap is broadcasting.
          - currently, all SSIDs used must match.
        - **position**: Coordinate offset in the 3D space. Used to determine distance from other nodes (in meters).
      - **stations**: *array*. wireless hosts in the topology
        - **id**: unique identifier for this node. Must have a unique number in it (this is used by Mininet to set a datapath-id).
        - **position**: Coordinate offset in the 3D space. Used to determine distance from other nodes (in meters).
    - **tests**: *array*. tests to run over the course of the run. Executed sequentially.
      - **name**: name to use for this test, to distinguish it from other tests. Has no impact on logic.
      - **type**: "node movements"
      - **timeframe**: movements in the same timeframe are (functionally) executed simultaneously. Once all movements within a timeframe are completed, connectivity tests are performed and the framework moves onto the next timeframe.
      - **node**: id of the node to move
      - **position**: coordinate offset to move the node to
    - **username**: username of the ssh-enabled user on the virtual machine that hosts mininet
    - **password**: password of the ssh-enabled user on the virtual machine that hosts mininet
    - **address**: ssh target. Must have the form <host>:<port>.

*Out*: 
- stdout: warnings and errors related to the validity of the given input file.
  - example of malformed input:
    ```json
    {
      "ok": false,
      "errors": [
        {
          "loc": "root",
          "code": "<>",
          "msg": "<>"
        }
      ],
      "warnings": []
    }
    ```
  - example of passing input:
    ```json
    {
      "ok": true,
      "errors": [],
      "warnings": []
    }
    ```
- exit code: 1 if errors were printed to stdout

## Test Runner

*In*:
- arg1: the json file validated by the prior input validation module execution.

*Out*: 
- `mn_output_raw` directory containing a timestamped subdirectory of the form YYYYMMDD-HHMMSS. Within the subdirectory will be one, raw output file per timeframe.
  - [Example](example_files/1_output-raw_results) of running this stage twice, once on 2025/11/03 and once on 2025/11/06

## [Coalesce Output](modules/2_mn_raw_output_processing)

This module consumes the raw data from Test Runner and transforms it such that Visualization can easily ingest and display useful results.

*In*: 
- arg1: path to a directory containing at least one timestamped subdirectory with raw test output files.
  - Example directory:
    ```
    some_dir/
    ├── 20251001_201140 <-- this one will be used/
    │   ├── timeframe0.txt
    │   ├── timeframe1.txt
    │   ├── ...
    │   └── timeframeN.txt
    └── 20250908_090142/
        └── ...
    ```

*Out*: 
- `./results` directory containing one subdirectory per timeframe and two CSV files:
  - ```
    results/
    ├── final_iw_data.csv
    ├── ping_data.csv
    ├── timeframe0/
    │   ├── edges.csv
    │   ├── nodes.csv
    │   └── ping_data_movement0.csv
    ├── timeframe1/
    │   ├── edges.csv
    │   ├── nodes.csv
    │   └── ping_data_movement1.csv
    ├── timeframe2/
    │   └── ...
    └── timeframeN/
        └── ...
  ```
  - `final_iw_data.csv` has 30 columns: device_type,test_file,device_name,interface,connected_to,ssid,freq,rx_bytes,rx_packets,tx_bytes,tx_packets,signal,rx_bitrate,tx_bitrate,bss_flags,dtim_period,beacon_int,flags,mtu,ether,tx_queue_len,rx_errors,rx_dropped,rx_overruns,rx_frame,tx_errors,tx_dropped,tx_overruns,tx_carrier,tx_collisions
    - [Example](example_files/2_results/final_iw_data.csv)
  - `ping_data.csv` has 11 columns: data_type,movement_number,test_file,node_name,position,src,dst,tx,rx,loss_pct,avg_rtt_ms
    - [Example](example_files/2_results/ping_data.csv)
  - `timeframeX/edges.csv` has 3 columns: id,source,target
  - `timeframeX/nodes.csv` has 8 columns: id,title,position,rx_bytes,rx_packets,tx_bytes,tx_packets,success_pct_rate
  - `timeframeX/ping_data_movement_X.csv` has 11 columns: data_type,movement_number,test_file,node_name,position,src,dst,tx,rx,loss_pct,avg_rtt_ms

## [Visualization](modules/3_output_visualization)
The Visualization module consumes the normalized CSV output from Coalesce Output and exposes it in a form that is easy for dashboards and operators to explore. 

In this implementation, visualization is a two-stage process:

1. Loading results into a SQLite database (via omenloader.py)
2. Rendering dashboards in Grafana using that database as a data source.

*In*: 
- arg1: path to the results directory (a directory produced by the Coalesce Output module.
  This directory is expected to contain:
  - Top-level CSVs:
    - `final_iw_data.csv`
    - `ping_data.csv`
  - One subdirectory per timeframe, each containing:
    - `timeframeX/nodes.csv`
    - `timeframeX/edges.csv`
    - `timeframeX/ping_data_movement_X.csv`
  Example:
 `results/
    ├── final_iw_data.csv
    ├── ping_data.csv
    ├── timeframe0/
    │   ├── edges.csv
    │   ├── nodes.csv
    │   └── ping_data_movement0.csv
    ├── timeframe1/
    │   ├── edges.csv
    │   ├── nodes.csv
    │   └── ping_data_movement1.csv
    ├── timeframe2/
    │   └── ...
    └── timeframeN/
        └── ...`

*Out*: 
1. SQLite database (for dashboards to query)
The visualization module writes a SQLite database (by default at `/opt/homebrew/var/lib/grafana/omen.db` containing:
- Per-timeframe graph tables (one prefix per timeframe), e.g:
  - `<prefix>_nodes`
     Columns (example):
      `id, title, subTitle, mainStat, severity, detail__rx_bytes, detail__rx_packets, detail__tx_bytes, detail__tx_packets, detail__success_rate, arc__success, arc__errors, latitude, longitude`
  - `<prefix>_edges`:
     Columns (example):
      `id, source, target, status`
  - `<prefix>_timeseries`:
    Columns taken directly from `timeframeX/ping_data_movement_X.csv`
- Global ping tables:
  - `ping_data`: Raw rows from `ping_data.csv` (all movements, all timeframe)
  - `ping_data_agg`:
    - Aggregated view of `ping_data`, grouped by `movement_number`
    - Numeric metrics (e.g., tx, rx, loss_pct, avg_rtt_ms) are averaged per movement to support smoother time-series       plots.
- Optionally: additonal tables reflecting `final_iw_data.csv` for link-level statistics.
These tables are indexed on common query keys (node IDs, edge endpoints) so that dashboard queries remain fast even as the dataset grows.

2. Visualization Layer (Grafana dashboards)
A Grafana instance is configured to point at the SQLite file via a SQLite data source. Dashboards (for eg, Dashboard.json) assume the schemas above and render:
- Node Graph panels:
  - Use `<prefix>_nodes` and `<prefix>_edges` to show APs and stations as nodes, links as edges.
  - Visual encodings:
    - severity -> node color (ok/ warning/ critical)
    - mainstat/ detail__sucess_rate -> primary health metric
    - `arc_success` / `arc_errors` -> proportional "donut" sucess vs errors.
   
<img width="620" height="287" alt="Screenshot 2025-11-25 at 4 29 04 PM" src="https://github.com/user-attachments/assets/203493f5-f9f4-45a1-86db-d8f1b1045967" />

   
<img width="1123" height="336" alt="Screenshot 2025-11-25 at 4 28 24 PM" src="https://github.com/user-attachments/assets/56c50e67-f0f7-4809-8677-5723433d721a" />

  
- Time-series panels:
  - Use `ping_data` or `ping_data_agg` to show connectivity metrics over time or movement number.
  - Typical metrics: success rate, loss fraction, average RTT per movement or per timeframe.

<img width="1478" height="333" alt="Screenshot 2025-11-25 at 4 32 25 PM" src="https://github.com/user-attachments/assets/53edf718-9eb9-4610-ad97-b9a3a15f9cf5" />

<img width="1486" height="326" alt="Screenshot 2025-11-25 at 4 32 05 PM" src="https://github.com/user-attachments/assets/2f38a8aa-6ba0-40e3-bfda-ef338aed4fde" />


- Geomap view:
  - Uses latitude/longitude derived from node position to overlay nodes on a campus map or geographic background.

 <img width="1484" height="401" alt="Screenshot 2025-11-25 at 4 34 08 PM" src="https://github.com/user-attachments/assets/0aef229c-31f8-48af-b8f9-ca8a7652dc32" />
 
Contract summary
Any Visualization module implementing this contract should:
- Consume the results/ directory structure described above, specifically:
  - ping_data.csv
  - final_iw_data.csv 
  - timeframeX/nodes.csv, timeframeX/edges.csv, timeframeX/ping_data_movement_X.csv for each timeframe.
- Produce:
  - A machine-readable data store (here: a SQLite database) with:
    - One set of node/edge tables per timeframe (prefix-based),
    - A raw ping table,
    - An aggregated ping table keyed by movement_number.
  - One or more visualization artifacts (here: Grafana dashboards) that rely only on this data contract.

As long as a replacement Visualization module reads the same input files and provides an equivalent data interface (tables or API), it can be swapped in without changing upstream modules.
  

