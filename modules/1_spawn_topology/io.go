package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

func getInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')

	// auto add port if not provided
	if strings.Contains(prompt, "Enter a valid target of the form '<host>:<port>':") {
		if !strings.Contains(input, ":") {
			fmt.Printf("No port detected -> Using default port 22\n")
			input = strings.TrimSpace(input) + ":22"
		}
	}
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

// copyResultsFromVM copies the latest test results from /tmp/test_results on the VM to ./mn_result_raw locally
func copyResultsFromVM(client *ssh.Client) error {
	// Find the latest results directory
	latestDir, err := findLatestResultsDir(client)
	if err != nil {
		return fmt.Errorf("find latest results directory: %w", err)
	}

	if latestDir == "" {
		fmt.Println("No test results found to copy")
		return nil
	}

	fmt.Printf("Found latest results directory: %s\n", latestDir)

	// Extract timestamp from the remote directory path
	timestamp := filepath.Base(latestDir)

	// Create local results directory with timestamp subdirectory
	localBaseDir := "./mn_result_raw"
	localDir := filepath.Join(localBaseDir, timestamp)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("create local directory %s: %w", localDir, err)
	}

	// Copy all files from the remote directory to local timestamped directory
	if err := copyDirectoryContents(client, latestDir, localDir); err != nil {
		return fmt.Errorf("copy directory contents: %w", err)
	}

	fmt.Printf("Successfully copied test results to %s\n", localDir)
	return nil
}

// findLatestResultsDir finds the latest timestamped directory in /tmp/test_results
func findLatestResultsDir(client *ssh.Client) (string, error) {
	baseDir := "/tmp/test_results"

	// Check if base directory exists and get latest timestamped directory
	cmd := fmt.Sprintf("[ -d %s ] && ls -1 %s | grep -E '^[0-9]{8}_[0-9]{6}$' | sort | tail -1", baseDir, baseDir)
	output, err := runSSHCommand(client, cmd)
	if err != nil {
		return "", fmt.Errorf("find latest directory: %w", err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return "", nil // No timestamped directories found
	}

	return filepath.Join(baseDir, output), nil
}

// copyDirectoryContents copies all files from remote directory to local directory
func copyDirectoryContents(client *ssh.Client, remoteDir, localDir string) error {
	// Get list of all files in the remote directory (recursively)
	cmd := fmt.Sprintf("find %s -type f", remoteDir)
	output, err := runSSHCommand(client, cmd)
	if err != nil {
		return fmt.Errorf("list files in %s: %w", remoteDir, err)
	}

	files := strings.Split(strings.TrimSpace(output), "\n")
	for _, filePath := range files {
		if filePath == "" {
			continue
		}

		// Calculate relative path from remote base directory
		relPath, err := filepath.Rel(remoteDir, filePath)
		if err != nil {
			return fmt.Errorf("calculate relative path: %w", err)
		}

		localPath := filepath.Join(localDir, relPath)

		// Create local directory structure if needed
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return fmt.Errorf("create local directory: %w", err)
		}

		// Copy file
		if err := downloadFile(client, filePath, localPath); err != nil {
			return fmt.Errorf("copy file %s: %w", filePath, err)
		}
		fmt.Printf("Copied: %s\n", relPath)
	}

	return nil
}

// downloadFile downloads a single file from remote to local using SSH commands
func downloadFile(client *ssh.Client, remotePath, localPath string) error {
	// Create SSH session
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	// Get file content using cat
	fileContent, err := session.Output(fmt.Sprintf("cat %s", remotePath))
	if err != nil {
		return fmt.Errorf("read remote file %s: %w", remotePath, err)
	}

	// Write to local file
	if err := os.WriteFile(localPath, fileContent, 0644); err != nil {
		return fmt.Errorf("write local file %s: %w", localPath, err)
	}

	return nil
}

// runSSHCommand runs a command on the remote server and returns the output
func runSSHCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	output, err := session.Output(command)
	if err != nil {
		return "", fmt.Errorf("run command '%s': %w", command, err)
	}

	return string(output), nil
}
