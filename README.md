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

>[!WARNING]
> Coordinator is not complete for MS2. The above instruction will be fully supported by MS3.

Build all components: `mage`.

Execute coordinator with a list of json files or directories containing json files: `artefacts/coordinator <>.json /path/to/dir/of/jsons/`.

## In Depth

Omen uses [mage](https://magefile.org/) as its build system, Go as the primary driver language, and dockerized Python scripts for some modules.

The coordinator is responsible for executing each step and passing I/O between modules. Each module can be executed individually, if that is preferred. See [Module Contracts](MODULE_CONTRACTS.md) for more information about each module's I/O expectations and results.

You can also run each step manually, if preferred.

Start by running `mage` from the top-level `./Omen` directory, then `cd` into `./artefacts/` (artefacts is where all build objects reside).

Each module strictly follows its [I/O contract](MODULE_CONTRACTS.md) to ensure proper cooperation.

#### Input Validation

Run the validator: `docker run --rm -v /path/to/user/input.json:/input/in.json 0_omen-input-validator:latest /input/in.json`

If this passes, the given file can be considered validated and ready for the rest of the pipeline.

#### Execute Test

This module is responsible for connecting to mininet, executing the test script, and collecting results for later processing.

Ensure your mininet vm is spinning and accessible via the u/p and address listed in the validated json.

Run the test driver: `./1_spawn /path/to/validated/in.json`

#### Output Coercion

The Raw Output module is responsible for transforming the the raw results from the test driver into usable input for the visualization module. Given a directory, this module will find the latest batch of results in the given path (by reading the timestamped subdirectories of the form YYYYMMDD_HHMMSS). It will coalesce the results into two files, placing them in a local `./results` directory.

Run output coercion: `./2_output_processing path/to/raw/results/directory/`

Example: `./2_output_processing ./mn_result_raw/`

#### Visualization

*TODO: all of these steps should really be automated. The Python loader can install expected configuration or that can be included in the docker image's build process. Spooling up the docker containers (and finding MySQL's IP) will be handled by Coordinator, as it has direct access to Docker via the client library.*

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

Run the loader:
```bash
docker run  --rm -it /
    -e DB_HOST=<OmenVizSQL ip address> /
    -e DB_PASS=mypass /
    -v ./path/to/nodes.csv:/input/nodes.csv /
    -v ./path/to/edges.csv:/input/edges.csv /
    3_omen-output-visualizer /input/nodes.csv /input/edges.csv
```

Install the Grafana dashboard used for node visualization:
- Navigate to Dashboards and click "new"
- Click "Import"
- Copy the [Dashboard JSON](modules/3_output_visualization/Dashboard.json) and replace all 3 instances of 'YOUR_DS_UID_HERE' with the id of the MySQL datasource.


# Architectural Diagram

![pipeline diagram](img/Pipeline.drawio.png)