package ctakes

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

var (
	ErrProcessNotRunning = errors.New("ctakes process not running")
	ErrInvalidResponse   = errors.New("invalid response from ctakes")
	ErrTimeout          = errors.New("ctakes request timeout")
)

type Manager struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	scanner  *bufio.Scanner
	mu       sync.Mutex
	running  bool
	jarPath  string
	heapSize string
}

type Request struct {
	ID       string                 `json:"id"`
	Action   string                 `json:"action"`
	Text     string                 `json:"text,omitempty"`
	Pipeline string                 `json:"pipeline,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type Response struct {
	ID      string      `json:"id"`
	Status  string      `json:"status"`
	Error   string      `json:"error,omitempty"`
	Results interface{} `json:"results,omitempty"`
}

func NewManager(jarPath, heapSize string) *Manager {
	return &Manager{
		jarPath:  jarPath,
		heapSize: heapSize,
	}
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	javaCmd := "java"
	if javaPath := os.Getenv("JAVA_HOME"); javaPath != "" {
		javaCmd = javaPath + "/bin/java"
	}

	m.cmd = exec.Command(javaCmd, "-Xmx"+m.heapSize, "-jar", m.jarPath)

	stdin, err := m.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	m.stdin = stdin

	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	m.stdout = stdout
	m.scanner = bufio.NewScanner(stdout)

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ctakes process: %w", err)
	}

	m.running = true
	return nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	if m.stdin != nil {
		m.stdin.Close()
	}

	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
		m.cmd.Wait()
	}

	m.running = false
	return nil
}

func (m *Manager) SendRequest(req Request) (*Response, error) {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil, ErrProcessNotRunning
	}
	m.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if _, err := fmt.Fprintln(m.stdin, string(data)); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	respChan := make(chan *Response, 1)
	errChan := make(chan error, 1)

	go func() {
		if m.scanner.Scan() {
			line := m.scanner.Text()
			var resp Response
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				errChan <- fmt.Errorf("failed to unmarshal response: %w", err)
				return
			}
			respChan <- &resp
		} else if err := m.scanner.Err(); err != nil {
			errChan <- fmt.Errorf("scanner error: %w", err)
		} else {
			errChan <- io.EOF
		}
	}()

	select {
	case resp := <-respChan:
		return resp, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(30 * time.Second):
		return nil, ErrTimeout
	}
}

func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}