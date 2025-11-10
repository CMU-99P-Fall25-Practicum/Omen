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
        - **noise_th**: sets the noise sensitivity threshold on receiving nodes. Lower thresholds mean more sensitive receivers. See the [README](README.md#noise-thresholds) for suggested values.
        - **propagation_model**: selects a propagation model for simulation wireless signal degradation. Alters the energy loss by distance; set this to simulate a mostly free-space environment, a mostly indoor environment, etc. See the [README](README.md#wireless-propagation-models) for more information on supported models and suggested values.
          - **exp**: path-loss exponent (n)
          - **s**: shadowing standard deviation (σ)
      - **aps**: *array*. access points (wireless routers) in the topology
        - **id**: unique identifier for this node. Must have a unique number in it (this is used by Mininet to set a datapath-id).
        - **mode**: "a". IEEE 802.11 mode.
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

*In*: 
- arg1: path to the results directory (a directory containing two files: `nodes.csv` and `edges.csv`)

*Out*: [variable. Entirely dependent on the visualization technique employed by the specific implementation of this module]

