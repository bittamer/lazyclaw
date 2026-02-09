package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/lazyclaw/lazyclaw/internal/models"
)

// CLIAdapter provides data by executing openclaw CLI commands
// Supports both local and remote (SSH) execution
type CLIAdapter struct {
	// Path to openclaw binary (empty = use PATH)
	BinaryPath string

	// SSH configuration for remote instances
	SSHConfig *models.SSHConfig

	// Instance name for display
	InstanceName string

	// Cached status
	mu          sync.RWMutex
	lastStatus  *models.OpenClawStatus
	lastFetched time.Time
	lastError   error

	// For log following
	logCmd    *exec.Cmd
	logCancel context.CancelFunc
}

// NewCLIAdapter creates a new CLI adapter for local execution
func NewCLIAdapter() *CLIAdapter {
	return &CLIAdapter{
		InstanceName: "local",
	}
}

// NewSSHCLIAdapter creates a new CLI adapter for remote execution via SSH
func NewSSHCLIAdapter(name string, sshConfig *models.SSHConfig, openclawPath string) *CLIAdapter {
	return &CLIAdapter{
		InstanceName: name,
		SSHConfig:    sshConfig,
		BinaryPath:   openclawPath,
	}
}

// IsRemote returns true if this adapter connects via SSH
func (c *CLIAdapter) IsRemote() bool {
	return c.SSHConfig != nil && c.SSHConfig.Host != ""
}

// GetInstanceName returns the instance name
func (c *CLIAdapter) GetInstanceName() string {
	return c.InstanceName
}

// GetLastError returns the last error encountered
func (c *CLIAdapter) GetLastError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastError
}

// GetFullStatus runs `openclaw status --json` and returns the full status
func (c *CLIAdapter) GetFullStatus() (*models.OpenClawStatus, error) {
	output, err := c.runCommand("status", "--json")
	if err != nil {
		c.mu.Lock()
		c.lastError = err
		c.mu.Unlock()
		return nil, err
	}

	var status models.OpenClawStatus
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		parseErr := fmt.Errorf("failed to parse status JSON: %w", err)
		c.mu.Lock()
		c.lastError = parseErr
		c.mu.Unlock()
		return nil, parseErr
	}

	// Cache the result
	c.mu.Lock()
	c.lastStatus = &status
	c.lastFetched = time.Now()
	c.lastError = nil
	c.mu.Unlock()

	return &status, nil
}

// GetCachedStatus returns the last fetched status without making a new request
func (c *CLIAdapter) GetCachedStatus() *models.OpenClawStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastStatus
}

// GetStatusAge returns how old the cached status is
func (c *CLIAdapter) GetStatusAge() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastFetched.IsZero() {
		return 0
	}
	return time.Since(c.lastFetched)
}

// IsGatewayReachable checks if the gateway is reachable based on cached status
func (c *CLIAdapter) IsGatewayReachable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastStatus == nil || c.lastStatus.Gateway == nil {
		return false
	}
	return c.lastStatus.Gateway.Reachable
}

// FollowLogs runs `openclaw logs --follow` and streams log events via channel
func (c *CLIAdapter) FollowLogs(ctx context.Context, logChan chan<- models.LogEvent) error {
	binary := c.getBinary()

	// Create a cancellable context
	ctx, cancel := context.WithCancel(ctx)
	c.logCancel = cancel

	cmd := exec.CommandContext(ctx, binary, "logs", "--follow")
	c.logCmd = cmd

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start logs command: %w", err)
	}

	// Read stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			event := parseLogLine(line)
			select {
			case logChan <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Read stderr (for errors)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			event := models.LogEvent{
				Timestamp: time.Now(),
				Level:     "error",
				Source:    "openclaw-cli",
				Message:   line,
				Raw:       line,
			}
			select {
			case logChan <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for command to finish in background
	go func() {
		_ = cmd.Wait()
	}()

	return nil
}

// StopFollowingLogs stops the log following process
func (c *CLIAdapter) StopFollowingLogs() {
	if c.logCancel != nil {
		c.logCancel()
	}
}

// parseLogLine attempts to parse a log line into structured form
// Format varies but often: "2024-01-15 10:30:45 [INFO] message"
func parseLogLine(line string) models.LogEvent {
	event := models.LogEvent{
		Timestamp: time.Now(),
		Level:     "info",
		Raw:       line,
	}

	// Try to parse JSON log format first
	var jsonLog struct {
		Time    string `json:"time"`
		Level   string `json:"level"`
		Msg     string `json:"msg"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(line), &jsonLog); err == nil {
		if jsonLog.Level != "" {
			event.Level = strings.ToLower(jsonLog.Level)
		}
		if jsonLog.Msg != "" {
			event.Message = jsonLog.Msg
		} else if jsonLog.Message != "" {
			event.Message = jsonLog.Message
		}
		if jsonLog.Time != "" {
			if t, err := time.Parse(time.RFC3339, jsonLog.Time); err == nil {
				event.Timestamp = t
			}
		}
		return event
	}

	// Try to parse bracketed level format: [INFO], [WARN], etc.
	line = strings.TrimSpace(line)
	if idx := strings.Index(line, "["); idx != -1 {
		if endIdx := strings.Index(line[idx:], "]"); endIdx != -1 {
			level := strings.ToLower(line[idx+1 : idx+endIdx])
			switch level {
			case "debug", "dbg":
				event.Level = "debug"
			case "info", "inf":
				event.Level = "info"
			case "warn", "warning", "wrn":
				event.Level = "warn"
			case "error", "err":
				event.Level = "error"
			}
			// Message is everything after the bracket
			event.Message = strings.TrimSpace(line[idx+endIdx+1:])
			return event
		}
	}

	// Fallback: use the whole line as message
	event.Message = line
	return event
}

// runCommand executes an openclaw CLI command (locally or via SSH)
func (c *CLIAdapter) runCommand(args ...string) (string, error) {
	if c.IsRemote() {
		return c.runSSHCommand(args...)
	}
	return c.runLocalCommand(args...)
}

// runLocalCommand executes openclaw locally
func (c *CLIAdapter) runLocalCommand(args ...string) (string, error) {
	binary := c.getBinary()
	cmd := exec.Command(binary, args...)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("command failed: %s", string(exitErr.Stderr))
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// runSSHCommand executes openclaw on a remote host via SSH
func (c *CLIAdapter) runSSHCommand(args ...string) (string, error) {
	sshArgs := c.buildSSHArgs()

	// Build the remote command
	remoteCmd := c.getBinary()
	for _, arg := range args {
		// Shell-escape arguments
		if strings.Contains(arg, " ") || strings.Contains(arg, "'") || strings.Contains(arg, "\"") {
			remoteCmd += " '" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
		} else {
			remoteCmd += " " + arg
		}
	}

	// Wrap in a login shell so the remote user's PATH (e.g. linuxbrew, nvm)
	// is loaded. Non-interactive SSH doesn't source .bashrc/.profile.
	remoteCmd = fmt.Sprintf("bash -lc %s", shellQuote(remoteCmd))

	sshArgs = append(sshArgs, remoteCmd)

	cmd := exec.Command("ssh", sshArgs...)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return "", fmt.Errorf("SSH command failed: %s", stderr)
			}
			return "", fmt.Errorf("SSH command failed with exit code %d", exitErr.ExitCode())
		}
		return "", fmt.Errorf("SSH connection failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// shellQuote wraps a string in single quotes for safe shell passing
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// buildSSHArgs builds the SSH command arguments
func (c *CLIAdapter) buildSSHArgs() []string {
	if c.SSHConfig == nil {
		return nil
	}

	var args []string

	// Batch mode - don't ask for passwords
	args = append(args, "-o", "BatchMode=yes")

	// Strict host key checking - disable for convenience (user can override)
	args = append(args, "-o", "StrictHostKeyChecking=accept-new")

	// Connection timeout
	timeout := c.SSHConfig.ConnectTimeout
	if timeout <= 0 {
		timeout = 10 // Default 10 seconds
	}
	args = append(args, "-o", fmt.Sprintf("ConnectTimeout=%d", timeout))

	// Port
	if c.SSHConfig.Port > 0 {
		args = append(args, "-p", fmt.Sprintf("%d", c.SSHConfig.Port))
	}

	// Identity file
	if c.SSHConfig.IdentityFile != "" {
		args = append(args, "-i", c.SSHConfig.IdentityFile)
	}

	// Proxy jump
	if c.SSHConfig.ProxyJump != "" {
		args = append(args, "-J", c.SSHConfig.ProxyJump)
	}

	// Build host string
	host := c.SSHConfig.Host
	if c.SSHConfig.User != "" && !strings.Contains(host, "@") {
		host = c.SSHConfig.User + "@" + host
	}
	args = append(args, host)

	return args
}

func (c *CLIAdapter) getBinary() string {
	if c.BinaryPath != "" {
		return c.BinaryPath
	}
	return "openclaw"
}

// CheckCLIAvailable checks if the openclaw CLI is available locally
func CheckCLIAvailable() bool {
	_, err := exec.LookPath("openclaw")
	return err == nil
}

// CheckSSHAvailable checks if SSH is available
func CheckSSHAvailable() bool {
	_, err := exec.LookPath("ssh")
	return err == nil
}
