package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type MininetController struct {
	host     string
	username string
	password string
	client   *ssh.Client
}

func NewMininetController(host, username, password string) *MininetController {
	return &MininetController{
		host:     host,
		username: username,
		password: password,
	}
}

func (mc *MininetController) Connect() error {
	config := &ssh.ClientConfig{
		User: mc.username,
		Auth: []ssh.AuthMethod{
			ssh.Password(mc.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, use proper host key verification
		Timeout:         30 * time.Second,
	}

	fmt.Printf("Connecting to %s@%s...\n", mc.username, mc.host)

	client, err := ssh.Dial("tcp", mc.host+":22", config)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	mc.client = client
	fmt.Println("SSH connection established successfully!")
	return nil
}

func (mc *MininetController) Disconnect() {
	if mc.client != nil {
		mc.client.Close()
		fmt.Println("SSH connection closed.")
	}
}

func (mc *MininetController) executeCommand(command string) (string, error) {
	session, err := mc.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// Request a pseudo terminal for interactive commands
	if err := session.RequestPty("xterm", 80, 40, ssh.TerminalModes{}); err != nil {
		return "", fmt.Errorf("failed to request pty: %v", err)
	}

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %v", err)
	}

	return string(output), nil
}

func (mc *MininetController) runInteractiveMininet() error {
	session, err := mc.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// Request a pseudo terminal for interactive session
	if err := session.RequestPty("xterm", 120, 40, ssh.TerminalModes{}); err != nil {
		return fmt.Errorf("failed to request pty: %v", err)
	}

	// Create pipes for stdin, stdout, stderr
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	fmt.Println("Starting Mininet...")
	fmt.Println("=" + strings.Repeat("=", 50) + "=")

	// Start the shell session
	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %v", err)
	}

	// Channel to signal when we're done reading output
	done := make(chan bool)

	// Goroutine to read and display output
	go func() {
		defer func() { done <- true }()

		// Create a multi-reader to read from both stdout and stderr
		reader := io.MultiReader(stdout, stderr)
		scanner := bufio.NewScanner(reader)

		sudoPasswordSent := false
		mininetStarted := false
		commandSent := false
		lastLineTime := time.Now()

		// Timer to detect when output stops (indicating a prompt is waiting)
		promptTimer := time.NewTimer(3 * time.Second)
		defer promptTimer.Stop()

		go func() {
			for {
				select {
				case <-promptTimer.C:
					// If we haven't sent the password and it's been quiet, there might be a password prompt
					if !sudoPasswordSent && !mininetStarted && commandSent && time.Since(lastLineTime) > 2*time.Second {
						fmt.Println("\n[DEBUG] Detected potential password prompt (no output for 2 seconds), sending password...")
						stdin.Write([]byte(mc.password + "\n"))
						sudoPasswordSent = true
					}
					promptTimer.Reset(1 * time.Second)
				}
			}
		}()

		for scanner.Scan() {
			line := scanner.Text()
			lastLineTime = time.Now()
			fmt.Println(line)

			// Reset the timer since we got output
			promptTimer.Reset(3 * time.Second)

			// Detect various sudo password prompt formats
			lowerLine := strings.ToLower(line)
			if !sudoPasswordSent && ((strings.Contains(lowerLine, "password") && strings.Contains(lowerLine, "sudo")) ||
				strings.Contains(line, "[sudo]") ||
				strings.Contains(lowerLine, "password for") ||
				strings.HasSuffix(strings.TrimSpace(line), ":") && strings.Contains(lowerLine, "password")) {
				fmt.Println("\n[DEBUG] Detected sudo password prompt, sending password...")
				time.Sleep(500 * time.Millisecond)
				stdin.Write([]byte(mc.password + "\n"))
				sudoPasswordSent = true
			}

			// Detect when Mininet has started
			if strings.Contains(line, "mininet>") && !mininetStarted {
				mininetStarted = true
				fmt.Println("\n[DEBUG] Mininet started, sending 'exit' command...")
				time.Sleep(500 * time.Millisecond) // Reduced delay
				stdin.Write([]byte("exit\n"))
			}

			// Detect Mininet startup messages (alternative detection)
			if !mininetStarted && strings.Contains(line, "*** Starting CLI:") {
				fmt.Println("\n[DEBUG] Detected Mininet CLI starting...")
				time.Sleep(1 * time.Second) // Wait for mininet> prompt
				fmt.Println("\n[DEBUG] Sending 'exit' command...")
				stdin.Write([]byte("exit\n"))
				mininetStarted = true
			}

			// Detect when we're back to the shell prompt after exiting Mininet
			if mininetStarted && (strings.Contains(line, "$ ") || strings.Contains(line, "# ") ||
				strings.HasSuffix(strings.TrimSpace(line), "$") ||
				strings.HasSuffix(strings.TrimSpace(line), "#") ||
				(strings.Contains(line, "completed in") && strings.Contains(line, "seconds"))) {
				fmt.Println("\n[DEBUG] Mininet completed, sending 'logout' command...")
				time.Sleep(500 * time.Millisecond)
				stdin.Write([]byte("logout\n"))
				time.Sleep(500 * time.Millisecond)
				break
			}
		}
	}()

	// Wait a moment for the shell to be ready
	time.Sleep(500 * time.Millisecond)

	// Send the Mininet command with double newline to trigger sudo password prompt
	fmt.Println("Executing: sudo -E mn")
	_, err = stdin.Write([]byte("sudo -E mn\n\n"))
	if err != nil {
		return fmt.Errorf("failed to send mininet command: %v", err)
	}

	// Wait for the session to complete or timeout
	sessionDone := make(chan error)
	go func() {
		sessionDone <- session.Wait()
	}()

	select {
	case err := <-sessionDone:
		fmt.Println("=" + strings.Repeat("=", 50) + "=")
		fmt.Println("Session completed")
		if err != nil && err.Error() != "Process exited with status 130" { // 130 is normal for Ctrl+C
			return fmt.Errorf("session error: %v", err)
		}
	case <-time.After(60 * time.Second):
		fmt.Println("=" + strings.Repeat("=", 50) + "=")
		fmt.Println("Session timeout - this is normal for interactive sessions")
	}

	// Wait for output reading to complete
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}

	return nil
}

func getInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func getPassword(prompt string) string {
	fmt.Print(prompt)
	// For password input, we'll use a simple approach
	// In a production environment, you might want to use a library like golang.org/x/term for hidden input
	reader := bufio.NewReader(os.Stdin)
	password, _ := reader.ReadString('\n')
	return strings.TrimSpace(password)
}

func main() {
	// Configuration - leave empty strings to trigger interactive input
	var (
		host     = ""
		username = ""
		password = ""
	)

	// Check command line arguments first
	if len(os.Args) >= 4 {
		host = os.Args[1]
		username = os.Args[2]
		password = os.Args[3]
		fmt.Printf("Using command line arguments: %s@%s\n", username, host)
	} else if len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		fmt.Println("Usage: go run main.go [host] [username] [password]")
		fmt.Println("If no arguments provided, you will be prompted for input")
		fmt.Println("You can also hardcode values in the source code")
		return
	} else {
		// Check if values are hardcoded (not empty)
		if host == "" {
			host = getInput("Enter host IP address: ")
		} else {
			fmt.Printf("Using hardcoded host: %s\n", host)
		}

		if username == "" {
			username = getInput("Enter username: ")
		} else {
			fmt.Printf("Using hardcoded username: %s\n", username)
		}

		if password == "" {
			password = getPassword("Enter password: ")
		} else {
			fmt.Println("Using hardcoded password: [hidden]")
		}

		fmt.Printf("Connecting to %s@%s\n", username, host)
	}

	// Validate inputs
	if host == "" || username == "" || password == "" {
		fmt.Println("Error: Host, username, and password are required")
		return
	}

	// Create controller
	controller := NewMininetController(host, username, password)

	// Connect to the VM
	if err := controller.Connect(); err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer controller.Disconnect()

	// Test basic connectivity
	fmt.Println("\nTesting connection with 'whoami' command...")
	output, err := controller.executeCommand("whoami")
	if err != nil {
		fmt.Printf("Test command failed: %v\n", err)
	} else {
		fmt.Printf("Connected as: %s\n", strings.TrimSpace(output))
	}

	// Check if Mininet is available
	fmt.Println("\nChecking Mininet availability...")
	output, err = controller.executeCommand("which mn")
	if err != nil {
		fmt.Printf("Mininet check failed: %v\n", err)
		fmt.Println("Please ensure Mininet is installed on the target system")
	} else {
		fmt.Printf("Mininet found at: %s\n", strings.TrimSpace(output))
	}

	// Run interactive Mininet session
	fmt.Println("\nStarting interactive Mininet session...")
	if err := controller.runInteractiveMininet(); err != nil {
		log.Printf("Mininet session error: %v", err)
	}

	fmt.Println("\nProgram completed successfully!")
}
