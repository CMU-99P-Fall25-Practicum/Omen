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
  - example:
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
- exit code: 1 if errors were printed to stdout

## Test Runner

*In*:
- arg1: the json file validated by the prior input validation module execution.
  - [Example](example_files/test_run.json).
    - Because JSON does not support inline comments, the fields are documented here:
      - **schemaVersion**: "1.0"
      - **meta.backend**: "mininet"
      - **meta.name**: name to use for this run, to distinguish it from other tests. Has no impact on logic.
      - **meta.duration**: *currently unused*. the maximum duration the actual test script is allowed to run for.
      - **topo.nets.noise_th**: sets the noise threshold. TODO document suggested values and knock-on effects.
      - **topo.propagation_model**: selects a propagation model for simulation wireless signal degradation. Alters the energy loss by distance; set this to simulate a mostly free-space environment, a mostly indoor environment, etc. Each **model** has its own set of required and optional parameters.
        - Supported models:
          - "logDistance": logarithmic power loss over distance. Parameters:
            - **exp**: exponential loss factor
              - Suggested values: 4
      - **topo.aps**: [array] access points (wireless routers) in the topology
        - **id**: unique identifier for this node. Must have a unique number in it (this is used by Mininet to set a datapath-id).
        - **mode**: "a". TODO???
        - **channel**: Wi-Fi channel this node is broadcasting on
        - **ssid**: name of the ssid this ap is broadcasting.
          - currently, only a single SSID is supported
        - **position**: Coordinate offset in the 3D space. Used to determine distance from other nodes (in meters).
      - **topo.stations**: [array] wireless hosts in the topology
        - **id**: unique identifier for this node. Must have a unique number in it (this is used by Mininet to set a datapath-id).
        - **position**: Coordinate offset in the 3D space. Used to determine distance from other nodes (in meters).
      - **topo.tests**: [array] tests to run over the course of the run. Executed sequentially.
        - **name**: name to use for this test, to distinguish it from other tests. Has no impact on logic.
        - **type**: enumeration of possible actions. Each **type** has its own set of required and optional parameters.
          - Supported types:
            - "node movements": move a node to the given location. Parameters:
              - **node**: id of the node to move
              - **position**: x,y,z coordinate to move the node to.
            - "iw": execute iw on the named node
              - **node**: id of the node to run iw on. If left empty, this test will be run on every node.
      - **username**: username of the ssh-enabled user on the virtual machine that hosts mininet
      - **password**: password of the ssh-enabled user on the virtual machine that hosts mininet
      - **address**: ssh target. Must have the form <host>:<port>.

      

*Out*: 
- stdout: path to a directory containing raw output from the result of each test as it was printed/redirected. The exact format of the files is as-of-yet undetermined.

## [Coalesce Output](modules/2_mn_raw_output_processing)

*In*: 
- arg1: path to a directory containing at least one timestamped subdirectory with raw test output files.
  - Example:
    ```
    ./
    ├─ raw_output/
    │  ├─ 20251001_201140
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
- `./results` directory containing two files: `pingall_full_data.csv` (containing reachability results at each step in the test) and `final_iw_data.csv` (containing the results of running `iw` against node as the final test concludes).
  - Example:
    ```
    ./
    ├─ results/
    │  ├─ pingall_full_data.csv
    │  ├─ final_iw_data.csv
    ```

See [the module's README](modules/2_mn_raw_output_processing/README.md) for more information.

## Visualization

*In*: 
- arg1: path to a text file containing the human readable results generated by coalesce output

*Out*: [variable. Entirely dependent on the visualization technique employed by the specific implementation of this module]

