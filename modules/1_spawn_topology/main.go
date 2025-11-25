/*
Package main implements the test runner module, capable of executing topologies and tests against a remote mininet host.

# Workflow

The internal logic of the module is as follows:

1. Slurp input json, using the ssh info to connect to the mininet vm.

2. Upload the driver script and input json files to the vm.

3. Run the script via `sudo python3 /tmp/mininet-script.py /tmp/input-topo.json`.

4. Download the raw output files for further processing in the [next (output handler)](../2_mn_raw_output_processing) module.

# Dependencies

- Go 1.24.4+

- ssh and scp in client PATH

- Remote vm with the following items in their path:
Mininet
Python (3.11+)
Sudo (required to run mininet)

- Remote vm must also have an ssh server available for connection and superuser permissions (to run mininet).
*/
package main

import (
	omen "Omen"
	"Omen/modules/1_spawn_topology/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var appName string = "test_runner"

// Configuration - Set these to hardcode values, leave empty for prompting
var (
	defaultHost         = ""
	defaultUsername     = ""
	defaultPassword     = ""
	defaultTopoFile     = "input-topo.json"   // default topology filename
	defaultPythonScript = "mininet-script.py" // default python script filename
)

// main application info.
// Constructed from args and flags
var (
	config = models.Config{
		TopoFile: defaultTopoFile,
	}
	inputTopo *models.Input
)

// resolveConfig is responsible for finalizing and error-checking the global config singleton hierarchically.
//
// Hierarchical priority: command line flags > JSON file > hardcoded defaults > user input
func resolveConfig() error {
	// Resolve username
	if config.Username == "" {
		if inputTopo.Username != "" {
			config.Username = inputTopo.Username
			fmt.Printf("Using username from JSON: %s\n", config.Username)
		} else if defaultUsername != "" {
			config.Username = defaultUsername
			fmt.Printf("Using hardcoded username: %s\n", config.Username)
		} else if config.Interactive {
			config.Username = getInput("Enter username: ")
		}
	} else {
		fmt.Printf("Using username from --remote flag: %s\n", config.Username)
	}

	if config.Username == "" {
		return errors.New("username must be supplied")
	}

	// Resolve host
	config.Host = func() netip.AddrPort {
		// if it was set by cli, we are done
		if config.Host.IsValid() {
			fmt.Printf("Using host from --remote flag: %v\n", config.Host)
			return config.Host
		}

		// Pull VM address from input JSON
		// Check if default port exists
		if inputTopo.AP != "" && !strings.Contains(inputTopo.AP, ":") {
			fmt.Printf("No port detected -> Using default port 22\n")
			inputTopo.AP = inputTopo.AP + ":22"
		}
		if ap, err := netip.ParseAddrPort(inputTopo.AP); err == nil {
			fmt.Printf("Using host from JSON: %v\n", ap)
			return ap
		}

		// Pull hosts from input JSON
		if ap, err := netip.ParseAddrPort(defaultHost); err == nil {
			fmt.Printf("Using hardcoded host: %v\n", ap)
			return ap
		}

		if config.Interactive {
			// pull from stdin
			var ap netip.AddrPort
			var err error
			for ap, err = netip.ParseAddrPort(getInput("Enter a valid target of the form '<host>:<port>':")); err != nil; {
			}
			return ap
		}
		return netip.AddrPort{}
	}()
	if !config.Host.IsValid() {
		return errors.New("a valid host/target must be supplied")
	}

	// Resolve password
	if config.Password == "" {
		if inputTopo.Password != "" {
			config.Password = inputTopo.Password
			fmt.Println("Using password from JSON: [hidden]")
		} else if defaultPassword != "" {
			config.Password = defaultPassword
			fmt.Println("Using hardcoded password: [hidden]")
		} else if config.Interactive {
			config.Password = getInput("Enter password (SSH/sudo): ")
		}
	}

	// Validate required fields
	if config.Username == "" || !config.Host.IsValid() || config.Password == "" {
		return fmt.Errorf("username, host, and password are required")
	}

	return nil
}

func main() {
	// define flags
	fs := pflag.FlagSet{}
	fs.Bool("help", false, "Tada!")
	fs.String("remote", "", "remote target to run on, e.g. username@192.168.64.5")
	fs.BoolVar(&config.UseCLI, "cli", false, "enter Mininet CLI instead of running pingall. Do not use with interactivity is disabled.")
	fs.StringVar(&config.RemotePathPython, "remote-path-python", "/tmp/"+defaultPythonScript, "remote path for the generated Python file")
	fs.StringVar(&config.RemotePathJSON, "remote-path-json", "/tmp/"+defaultTopoFile, "remote path for the generated JSON file")
	fs.BoolVar(&config.Interactive, "interactive", true, "enables prompting for missing information."+
		"If false, this module will fail out on missing information rather than prompting for it.")
	fs.MarkHidden("cli")

	// generate command "tree"
	root := &cobra.Command{
		Use:   appName + " <topo>.json",
		Short: appName + " drives the testing and remote connection functionality of Omen",
		Long: appName + " creates and runs Mininet topologies from JSON files on remote VMs." +
			"It handles SSH connections, uploads topology scripts, manages Mininet sessions, and collects raw output." +
			"If --interactive, " + appName + " will prompt for required inputs not supplied in the topology JSON.",
		Example: appName + " input.json\n" +
			appName + " --remote=wifi@127.0.0.1 --interactive=false input.json",
		Args: cobra.ExactArgs(1),

		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Sets SSH information if --remote was specified.
			remote, err := cmd.Flags().GetString("remote")
			if err != nil {
				return err
			}

			if remote = strings.TrimSpace(remote); remote != "" {
				parts := strings.Split(remote, "@")
				if len(parts) != 2 {
					return fmt.Errorf("invalid remote format, expected username@host")
				}
				config.Username = parts[0]
				config.Host, _ = netip.ParseAddrPort(parts[1]) // throw away error; validity is checked later
			}

			{ // slurp topology
				if args[0] = strings.TrimSpace(args[0]); args[0] != "" {
					config.TopoFile = args[0]
				}
				fmt.Printf("Loading topology from: %s\n", config.TopoFile)
				data, err := os.ReadFile(config.TopoFile)
				if err != nil {
					return fmt.Errorf("read topo file: %w", err)
				}

				if err := json.Unmarshal(data, &inputTopo); err != nil {
					return fmt.Errorf("parse topology JSON: %w", err)
				}
			}

			// validate config set from flags
			return resolveConfig()
		},
		RunE: run,
	}

	// attach flags
	root.Flags().AddFlagSet(&fs)

	if err := fang.Execute(context.Background(),
		root,
		fang.WithoutCompletions(),
		fang.WithVersion(omen.Version),
		fang.WithErrorHandler(omen.FangErrorHandler),
	); err != nil {
		os.Exit(1)
	}

}

// run is the primary driver application.
// Expects the topology and all configuration to be valid.
func run(cmd *cobra.Command, args []string) error {
	// Display final configuration
	fmt.Printf("\n"+`Final Configuration:
	Host               : `+config.Host.String()+`
	Username           : `+config.Username+`
	Password           : [hidden]
	Topology File      : `+config.TopoFile+`
	Py Script          : %s
	Mode               : %s
	Remote Python path : %s
	Remote JSON path   : %s
	Hosts              : %v
	Stations           : %v
	Switches           : %v
	Aps                : %v
	Links              : %v`+"\n",
		defaultPythonScript,
		map[bool]string{true: "Interactive CLI", false: "Automated pingall"}[config.UseCLI],
		config.RemotePathPython,
		config.RemotePathJSON,
		inputTopo.Topo.Hosts,
		inputTopo.Topo.Stations,
		inputTopo.Topo.Switches,
		inputTopo.Topo.Aps,
		inputTopo.Topo.Links)

	// Execute the remote Mininet session
	if err := runRemoteMininet(&config, defaultPythonScript); err != nil {
		return fmt.Errorf("ERROR: run remote mininet: %w", err)
	}

	fmt.Println("Program completed successfully!")
	return nil
}
