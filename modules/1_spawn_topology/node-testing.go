package main

import (
	"fmt"
)

func genCommand(useCLI bool) string {
	// Build Mininet command
	var mnCommand string
	if useCLI {
		mnCommand = fmt.Sprintf("sudo -E mn --custom %s --topo fromjson", config.RemotePath)
		fmt.Printf("-> Starting interactive Mininet session (type 'exit' to quit)\n")
	} else {
		mnCommand = fmt.Sprintf("sudo -E mn --custom %s --topo fromjson --test pingall", config.RemotePath)
		fmt.Printf("-> Running automated pingall test\n")
	}

	return mnCommand
}
