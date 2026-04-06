package acp

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"

	"github.com/snana7mi/conchtalk-dlc/relay"
)

// Session manages a single ACP agent process with bidirectional I/O.
type Session struct {
	id       string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	cancel   context.CancelFunc
	mu       sync.Mutex
	closed   bool
	sendFunc func(relay.OutgoingMessage) error
}

// Manager manages multiple concurrent ACP sessions.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*Session)}
}

// Start launches an ACP agent process.
func (m *Manager) Start(sessionID, command, cwd string, sendFunc func(relay.OutgoingMessage) error) error {
	m.mu.Lock()
	if old, ok := m.sessions[sessionID]; ok {
		old.close()
	}
	m.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = cwd

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start: %w", err)
	}

	s := &Session{
		id:       sessionID,
		cmd:      cmd,
		stdin:    stdin,
		cancel:   cancel,
		sendFunc: sendFunc,
	}

	m.mu.Lock()
	m.sessions[sessionID] = s
	m.mu.Unlock()

	go s.readStream(stdout, "stdout")
	go s.readStream(stderr, "stderr")

	go func() {
		_ = cmd.Wait()
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		m.mu.Lock()
		// Only remove if this session is still the current one (not replaced).
		if m.sessions[sessionID] == s {
			delete(m.sessions, sessionID)
			m.mu.Unlock()
			_ = sendFunc(relay.OutgoingMessage{
				Type:      "acp_closed",
				SessionID: sessionID,
			})
		} else {
			m.mu.Unlock()
		}
	}()

	return nil
}

// Send writes data to an ACP process's stdin.
func (m *Manager) Send(sessionID, data string) error {
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("session closed: %s", sessionID)
	}
	_, err := s.stdin.Write([]byte(data))
	return err
}

// Close closes a specific session.
func (m *Manager) Close(sessionID string) {
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	if ok {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()
	if ok {
		s.close()
	}
}

// CloseAll closes all sessions.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	all := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		all = append(all, s)
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()
	for _, s := range all {
		s.close()
	}
}

func (s *Session) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	s.stdin.Close()
	s.cancel()
}

func (s *Session) readStream(r io.Reader, stream string) {
	buf := make([]byte, 64*1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if sendErr := s.sendFunc(relay.OutgoingMessage{
				Type:      "acp_data",
				SessionID: s.id,
				Stream:    stream,
				Data:      string(buf[:n]),
			}); sendErr != nil {
				log.Printf("[acp] send failed for session %s: %v", s.id, sendErr)
			}
		}
		if err != nil {
			break
		}
	}
}
