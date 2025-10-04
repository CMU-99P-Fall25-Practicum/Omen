package main

/**
This file is for the custom tests to run within mininet
*/

import (
	"fmt"
)

/*
*
Generate mininet command

Current iteration: Execute mininet topology and tests within the Python script

## --cli flag current has no use

(useCLI is a flag to determine if the user want to activate "interactive mode")

useCLI == true: run interactive mode with input topology
useCLI == false: run "pingall" test and end the session
*/
func genCommand(useCLI bool) string {
	// Build Mininet command
	var mnCommand string = fmt.Sprintf("sudo python3 %s %s", config.RemotePathPython, config.RemotePathJSON)

	if useCLI {
		// mnCommand = fmt.Sprintf("sudo mn --custom %s --topo fromjson", config.RemotePath)
		// fmt.Printf("-> Starting interactive Mininet session (type 'exit' to quit)\n")
		fmt.Printf("-> Executing Python script: (cli flag enable)\n")
	} else {
		// mnCommand = fmt.Sprintf("sudo mn --custom %s --topo fromjson --test pingall", config.RemotePath)
		// fmt.Printf("-> Running automated pingall test\n")
		fmt.Printf("-> Executing Python script: (cli flag disable)\n")
	}

	return mnCommand
}
