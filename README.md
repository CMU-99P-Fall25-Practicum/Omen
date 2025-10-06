# Omen

A modular pipeline for instrumenting network simulations and producing real, usable results.

# Team Members

*Project Manager* - Anna Bernsteiner

*Tech Team*
- Vaidehi Mehra
- Tiger Li
- Gavin Liao

*Tech Lead* - Rory Landau

# Building

## Dependencies

- Docker
- Go 1.25+
- A Mininet_Wifi VM available over SSH

## Quick Start

>[!NOTE]
> All commands executed from the top level directory.

Build all components: `mage`.

Execute coordinator with a list of json files or directories containing json files: `artefacts/coordinator <>.json /path/to/dir/of/jsons/`.

## In Depth

Omen uses [mage](https://magefile.org/) as its build system, Go as the primary driver language, and dockerized Python scripts for some modules.

The coordinator is responsible for executing each step and passing I/O between modules. Each module can be executed individually, if that is preferred. See [Module Contracts](MODULE_CONTRACTS.md) for more information about each module and the I/O each module expected and returns.

### Manual Execution & Module Descriptions

You can also run each step manually, if preferred. This assumes that each binary has been compiled and placed into `./artefacts/` and each docker image built.

Each module strictly follows its [I/O contract](MODULE_CONTRACTS.md) to ensure proper cooperation.

Start by entering the artefacts directory: `cd artefacts`.

#### Input Validation

Run the validator: `docker run --rm -v path/to/user/input.json:/input/in.json 0_omen-input-validator:latest /input/in.json`

#### Execute Test

Run the test driver: `./1_spawn path/to/validated/in.json`

#### Output Coercion

The Raw Output module is responsible for transforming the the raw results from the test driver into usable input for the visualization module. Given a directory, this module will find the latest batch of results in the given path (by reading the timestamped subdirectories of the form YYYYMMDD_HHMMSS). It will coalesce the results into two files, placing them in a local `./results` directory.

Run output coercion: `./2_output_processing path/to/raw/results/directory/`

TODO

#### Visualization

As of Milestone 2, each run of the visualizer expects a clean environment; as such, Docker containers are started fresh each run.

Spool up Grafana: `docker run --name OmenVizGrafana -d -p 3000:3000 --name=grafana grafana/grafana`

Spool up MySQL: `docker run --name OmenVizSQL -e MYSQL_DATABASE=test -e MYSQL_ROOT_PASSWORD=mypass -p 3306:3306 -d mysql:latest`

Find the address we can connect on:
```bash
docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' OmenVizSQL
```

(Optional) manually connect to the database: `mysql -h <OmenVizSQL ip address> -u root -p`

Grafana now must also be configured to connect to the SQL server. Access the Grafana server at `localhost:3000`, navigate to "connections/add new connection", select mySQL, and enter the following details:
- **Host URL**: <OmenVizSQL_IP>:3306
- **Database name**: test
- **Username**: root
- **Password**: mypass

*TODO: this really should all be handled automatically, via configuration files installed into the container.*

Run the loader:
```bash
docker run  --rm -it /
    -e DB_HOST=<OmenVizSQL ip address> /
    -e DB_PASS=mypass /
    -v ./path/to/nodes.csv:/input/nodes.csv /
    -v ./path/to/edges.csv:/input/edges.csv /
    3_omen-output-visualizer /input/nodes.csv /input/edges.csv
```

*TODO: how do we initialize Grafana's dashboard. Ideally, we want this to be automated*

# Architectural Diagram

![pipeline diagram](img/Pipeline.drawio.png)