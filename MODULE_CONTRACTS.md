This file details the I/O contracts between modules, to ensure that they can be swapped easily.
As long as a module consumes its input contract and its output satisfies the next module's input contract, modules can be freely interchanged.

# Modules

## Input Validation

As it says on the tin, Input Validation modules consume a configuration file and test its parameters for validity so all future modules can assume their inputs are proper.

*In*:
- arg1: Json file (path taken as argument) to be validated. This file contains a topology, at least one test to execute against the topology, environmental conditions to apply to the topology’s links and nodes, and all required metadata (ssh information & credentials, test name, backend, OSM connection information, etc).
  - [Example](example_files/test_run.json).

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
- stdout: path to a directory containing at least one timestamped subdirectory with raw output files.
  - The directory will contain one subdirectory for each run of the pipeline.
  - Each subdirectory will be named with the timestamp of that run's execution, of the form YYYYMMDD-HHMMSS.
    - There will be one file per test within a given subdirectory.
      - Each file will contain different data depending on the test run.
  - [Example](example_files/1_output-raw_results)

## [Coalesce Output](modules/2_mn_raw_output_processing)

This module consumes the raw data from Test Runner and transforms it such that Visualization can easily ingest and display useful results.

*In*: 
- arg1: path to a directory containing at least one timestamped subdirectory with raw test output files.
  - Example directory:
    ```
    ./
    ├─ raw_output/
    │  ├─ 20251001_201140 <-- this one will be used
    │  │  ├─ test1.txt
    │  │  ├─ test2.txt
    │  │  ├─ test3.txt
    │  │  ├─ test4.txt
    │  │  ├─ ...
    │  ├─ 20250908_090142
    │  │  ├─ test1.txt
    │  │  ├─ test2.txt
    ...
    ```

*Out*: 

- `./results` directory containing two files: `nodes.csv` and `edges.csv`.
  - `nodes.csv` has X columns: id,title,rx_bytes,rx_packets,tx_bytes,tx_packets,success_pct_rate
    - example file:
      ```
      id,title,rx_bytes,rx_packets,tx_bytes,tx_packets,success_pct_rate
      sta1,sta1,356802,8716,4898,68,0.50
      sta2,sta2,356533,8712,4898,68,0.50
      sta3,sta3,165551,4059,2066,29,0.50
      sta4,sta4,354209,8683,4286,64,0.50
      ap1,ap1,8598,137,11064,137,0.60
      ap2,ap2,5116,84,6628,84,0.60
      ```
  - `edges.csv` has Y columns: id,source,target
    - example file:
      ```
      id,source,target
      sta1-ap1,sta1,ap1
      sta1-ap2,sta1,ap2
      sta2-ap1,sta2,ap1
      sta2-ap2,sta2,ap2
      sta3-ap1,sta3,ap1
      sta3-ap2,sta3,ap2
      sta4-ap1,sta4,ap1
      sta4-ap2,sta4,ap2
      ap1-sta1,ap1,sta1
      ap1-sta2,ap1,sta2
      ap1-sta3,ap1,sta3
      ap1-sta4,ap1,sta4
      ap1-ap2,ap1,ap2
      ap2-sta1,ap2,sta1
      ap2-sta2,ap2,sta2
      ap2-sta3,ap2,sta3
      ap2-sta4,ap2,sta4
      ap2-ap1,ap2,ap1
      ```
  - Example directory:
    ```
    ./
    ├─ raw_output/
    │  ├─ 20251001_201140
    │  ├─ 20250908_090142
    ├─ results/
    │  ├─ pingall_full_data.csv
    │  ├─ final_iw_data.csv
    ├─ other_file.txt
    ├─ other_directory/
    ```
    - [pingall_full_data.csv](example_files/2_output-pingall_full_data.csv).
    - [final_iw_data.csv](example_files/2_output-final_iw_data.csv)
      

See [the module's README](modules/2_mn_raw_output_processing/README.md) for more information.

## [Visualization](modules/3_output_visualization)

*In*: 
- arg1: path to the results directory (a directory containing two files: `nodes.csv` and `edges.csv`)

*Out*: [variable. Entirely dependent on the visualization technique employed by the specific implementation of this module]

