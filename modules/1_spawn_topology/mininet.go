package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/CMU-99P-Fall25-Practicum/Omen/modules/spawn_topology/models"
	"golang.org/x/crypto/ssh"
)

func runRemoteMininet(config *models.Config, defaultPythonScript string) error {
	// 1) Validate that the local file exists
	if _, err := os.Stat(defaultPythonScript); os.IsNotExist(err) {
		return fmt.Errorf("local Python file does not exist: %s", defaultPythonScript)
	}

	// 2) Establish SSH connection
	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	fmt.Printf("-> Connecting to %s@%s\n", config.Username, config.Host)
	client, err := ssh.Dial("tcp", config.Host.String(), sshConfig)
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}
	defer client.Close()

	// 3) Upload Python file via SFTP-like functionality
	fmt.Printf("-> Uploading topology script {%s} to {%s}\n", defaultPythonScript, config.RemotePathPython)
	if err := uploadFile(client, defaultPythonScript, config.RemotePathPython); err != nil {
		return fmt.Errorf("file upload failed: %w", err)
	}

	// 4) Upload Topo JSON file via SFTP-like functionality
	fmt.Printf("-> Uploading topology JSON {%s} to {%s}\n", config.TopoFile, config.RemotePathJSON)
	if err := uploadFile(client, config.TopoFile, config.RemotePathJSON); err != nil {
		return fmt.Errorf("file upload failed: %w", err)
	}

	// 5) Run Mininet command
	if err := runMininet(client, config); err != nil {
		return fmt.Errorf("mininet execution failed: %w", err)
	}

	return nil
}

func runMininet(client *ssh.Client, config *models.Config) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	// Request a pseudo terminal for interactive session
	if err := session.RequestPty("xterm", 120, 40, ssh.TerminalModes{}); err != nil {
		return fmt.Errorf("request pty: %w", err)
	}

	// Create pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("create stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("create stderr pipe: %w", err)
	}

	// Build Mininet command
	// TODO: Add --cli flag in python script to enable cli mode if requested
	// Current: Execute Python script that we just uploaded
	var mnCommand string = genCommand(config.UseCLI)

	fmt.Printf("-> Executing: %s\n", mnCommand)

	// Start shell session
	if err := session.Shell(); err != nil {
		return fmt.Errorf("start shell: %w", err)
	}

	// Handle output and input in goroutines
	done := make(chan bool)

	// Output handling goroutine
	go func() {
		defer func() { close(done) }()

		reader := io.MultiReader(stdout, stderr)
		scanner := bufio.NewScanner(reader)

		sudoPasswordSent := false
		mininetStarted := false

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.Contains(line, config.Password) { // forbit password output on terminal
				fmt.Println(line)
			}

			// Detect sudo password prompt and auto-respond
			lowerLine := strings.ToLower(line)
			if !sudoPasswordSent && ((strings.Contains(lowerLine, "password") && strings.Contains(lowerLine, "sudo")) ||
				strings.Contains(line, "[sudo]") ||
				strings.Contains(lowerLine, "password for") ||
				(strings.HasSuffix(strings.TrimSpace(line), ":") && strings.Contains(lowerLine, "password"))) {
				fmt.Println("\n[DEBUG] Detected sudo password prompt, sending password...")
				time.Sleep(300 * time.Millisecond)
				stdin.Write([]byte(config.Password + "\n"))
				sudoPasswordSent = true
			}

			// For CLI mode, detect when Mininet starts and handle exit
			if config.UseCLI {
				if strings.Contains(line, "mininet>") && !mininetStarted {
					mininetStarted = true
					fmt.Println("\n[DEBUG] Mininet CLI started. Type commands or 'exit' to quit.")
					// In CLI mode, let user interact directly
				}

				// Detect when user exits Mininet in CLI mode
				if mininetStarted && (strings.Contains(line, "*** Stopping") ||
					strings.Contains(line, "completed in") && strings.Contains(line, "seconds")) {
					fmt.Println("\n[DEBUG] Mininet session ended, logging out...")
					time.Sleep(500 * time.Millisecond)
					stdin.Write([]byte("exit\n"))
					time.Sleep(500 * time.Millisecond)
					break
				}
			} else {
				// For automated mode, detect completion
				if strings.Contains(line, "*** Done") {
					fmt.Println("\n[DEBUG] Pingall test completed, ending session...")
					time.Sleep(500 * time.Millisecond)
					stdin.Write([]byte("exit\n"))
					time.Sleep(500 * time.Millisecond)
					break
				}
			}
		}
	}()

	// Send the Mininet command
	time.Sleep(500 * time.Millisecond)               // Wait for shell to be ready
	_, err = stdin.Write([]byte(mnCommand + "\n\n")) // Double newline to trigger sudo prompt
	if err != nil {
		return fmt.Errorf("send command: %w", err)
	}

	// For CLI mode, also handle direct user input
	if config.UseCLI {
		go func() {
			// Forward user input to remote session
			userInput := bufio.NewScanner(os.Stdin)
			for userInput.Scan() {
				line := userInput.Text()
				stdin.Write([]byte(line + "\n"))
				if line == "exit" {
					break
				}
			}
		}()
	}

	// Wait for session completion or timeout
	sessionDone := make(chan error)
	go func() {
		sessionDone <- session.Wait()
	}()

	select {
	case err := <-sessionDone:
		if err != nil && err.Error() != "Process exited with status 130" { // 130 is normal for Ctrl+C
			return fmt.Errorf("session error: %w", err)
		}
	case <-time.After(120 * time.Second): // Longer timeout for interactive sessions
		fmt.Println("\n[DEBUG] Session timeout")
	}

	// Wait for output processing to complete
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}

	return nil
}
