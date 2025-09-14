package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

func getInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func uploadFile(client *ssh.Client, localPath, remotePath string) error {
	// Read local file
	localData, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("read local file: %w", err)
	}

	// Create remote file using SSH session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	// Use cat command to write file content
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("create stdin pipe: %w", err)
	}

	// Start the cat command to write to remote file
	if err := session.Start(fmt.Sprintf("cat > %s", remotePath)); err != nil {
		return fmt.Errorf("start cat command: %w", err)
	}

	// Write file content
	if _, err := stdin.Write(localData); err != nil {
		return fmt.Errorf("write file content: %w", err)
	}
	stdin.Close()

	// Wait for completion
	if err := session.Wait(); err != nil {
		return fmt.Errorf("wait for upload: %w", err)
	}

	return nil
}
