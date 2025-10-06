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

The coordinator is responsible for executing each step and passing I/O between modules. Each module can be executed individually, if that is preferred. See [Module Contracts](MODULE_CONTRACTS.md) for information about what I/O each module expected and returns.

### Manual Execution

You can also run each step manually, if preferred. This assumes that each binary has been compiled and placed into `./artefacts/` and each docker image built.

Each module strictly follows its [I/O contract](MODULE_CONTRACTS.md) to ensure proper cooperation.

Start by entering the artefacts directory: `cd artefacts`.

#### Input Validation

Run the validator: `docker run --rm -v path/to/user/input.json:/input/in.json 0_omen-input-validator:latest /input/in.json`

#### Execute Test

Run the test driver: `./1_spawn path/to/validated/in.json`

TODO

#### Output Coercion

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

Run the loader:
```bash
docker run  --rm -it /
    -e DB_HOST=172.17.0.2 /
    -e DB_PASS=mypass <OmenVizSQL ip address> /
    -v ./modules/3_output_visualization/nodes.csv:/input/nodes.csv /
    -v ./modules/3_output_visualization/edges.csv:/input/edges.csv /
    3_omen-output-visualizer /input/nodes.csv /input/edges.csv
```

# Architectural Diagram

![pipeline diagram](img/Pipeline.drawio.png)