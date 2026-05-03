package oracle

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

//go:embed oracle_helper.py
var helperScript string

// Subprocess speaks newline-delimited JSON to a Python interpreter
// running internal/oracle/oracle_helper.py. Calls are serialized via
// a mutex; the helper is single-threaded.
type Subprocess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader

	nextID atomic.Int64
	mu     sync.Mutex
	closed atomic.Bool
}

// Start spawns python at the given path, feeds it the embedded helper
// script via -c, and waits for the readiness ping. python must point
// at a usable Python 3 interpreter. The lifetime is bound to ctx;
// canceling ctx terminates the subprocess.
func Start(ctx context.Context, python string) (*Subprocess, error) {
	if python == "" {
		return nil, fmt.Errorf("oracle: empty python path")
	}
	// #nosec G204 -- the python path is discovered from the project's
	// own venv or the user's PATH; running it is the point of the
	// oracle. Callers gate this behind --oracle.
	cmd := exec.CommandContext(ctx, python, "-c", helperScript)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("oracle: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("oracle: stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("oracle: start %s: %w", python, err)
	}
	s := &Subprocess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}
	if err := s.awaitReady(); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

func (s *Subprocess) awaitReady() error {
	line, err := s.stdout.ReadString('\n')
	if err != nil {
		return fmt.Errorf("oracle: read ready: %w", err)
	}
	var ready struct {
		Ready bool `json:"ready"`
	}
	if err := json.Unmarshal([]byte(line), &ready); err != nil {
		return fmt.Errorf("oracle: parse ready %q: %w", line, err)
	}
	if !ready.Ready {
		return fmt.Errorf("oracle: not ready: %q", line)
	}
	return nil
}

// DeviceOf implements Oracle.
func (s *Subprocess) DeviceOf(expr string) Result {
	r, err := s.call("device_of", map[string]any{"expr": expr})
	if err != nil {
		return Unknown
	}
	return r
}

// SubclassesNNModule implements Oracle.
func (s *Subprocess) SubclassesNNModule(qualname string) Result {
	r, err := s.call("subclasses_nn_module", map[string]any{"qualname": qualname})
	if err != nil {
		return Unknown
	}
	return r
}

// Close terminates the subprocess. Safe to call multiple times.
func (s *Subprocess) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	_ = s.stdin.Close()
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	_ = s.cmd.Wait()
	return nil
}

func (s *Subprocess) call(method string, params any) (Result, error) {
	if s.closed.Load() {
		return Unknown, fmt.Errorf("oracle: closed")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.nextID.Add(1)
	req, err := json.Marshal(map[string]any{
		"id":     id,
		"method": method,
		"params": params,
	})
	if err != nil {
		return Unknown, fmt.Errorf("oracle: marshal: %w", err)
	}
	if _, err := s.stdin.Write(append(req, '\n')); err != nil {
		return Unknown, fmt.Errorf("oracle: write: %w", err)
	}
	line, err := s.stdout.ReadString('\n')
	if err != nil {
		return Unknown, fmt.Errorf("oracle: read: %w", err)
	}
	var resp struct {
		ID     int64  `json:"id"`
		Result Result `json:"result"`
		Error  string `json:"error,omitempty"`
	}
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return Unknown, fmt.Errorf("oracle: parse %q: %w", line, err)
	}
	if resp.Error != "" {
		return Unknown, fmt.Errorf("oracle: %s", resp.Error)
	}
	return resp.Result, nil
}
