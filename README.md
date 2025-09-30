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

# Architectural Diagram

![pipeline diagram](img/Pipeline.drawio.png)