package main

/**
This file is for the custom tests to run within mininet
*/

import (
	"fmt"
)

/*
*
Generate mininet command to either run "pingall" test or not based on the useCLI flag

(useCLI is a flag to determine if the user want to activate "interactive mode")

useCLI == true: run interactive mode with input topology
useCLI == false: run "pingall" test and end the session
*/
func genCommand(useCLI bool) string {
	// Build Mininet command
	var mnCommand string
	if useCLI {
		mnCommand = fmt.Sprintf("sudo mn --custom %s --topo fromjson", config.RemotePath)
		fmt.Printf("-> Starting interactive Mininet session (type 'exit' to quit)\n")
	} else {
		mnCommand = fmt.Sprintf("sudo mn --custom %s --topo fromjson --test pingall", config.RemotePath)
		fmt.Printf("-> Running automated pingall test\n")
	}

	return mnCommand
}
