This file details the I/O contracts between modules, to ensure that they can be swapped easily.
As long as a module consumes its input contract and its output satisfies the next module's input contract, modules can be freely interchanged.

# Modules

## Input Validation

As it says on the tin, Input Validation modules consume a configuration file and test its parameters for validity so all future modules can assume their inputs are proper.

*In*: a configuration file in the JSON schema

*Out*: 1 `meta.json` file, containing information for run-wide config (ssh credentials, broad settings), and X `bundle` files, each containing a single topology and the tests to run against it.

## Spawn Topology

Responsible for turning validated topology.json files into mininet topologies via python scripts.

*In*: a single bundle file
*Out* a topology script to spin up the mininet topology that begins a bundle's test execution