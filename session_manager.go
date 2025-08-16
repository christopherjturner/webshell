package main

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/creack/pty"
)

type Client struct {
	ws     *websocket.Conn
	send   chan []byte
	cancel context.CancelFunc
}

type Session struct {
	id         string
	proc       *ShellProcess
	mu         sync.Mutex
	client     *Client
	done       chan struct{}
	lastActive atomic.Value
	ttl        time.Duration
}

type SessionManager struct {
	sessions map[string]*Session
	mu       sync.Mutex
	ttl      time.Duration
}

func NewSession(id string, proc *ShellProcess, ttl time.Duration) *Session {
	s := &Session{
		id:   id,
		proc: proc,
		ttl:  ttl,
		done: make(chan struct{}),
	}
	s.ImAlive()

	go s.readTTY()
	return s
}

func (s *Session) readTTY() {
	buf := make([]byte, maxBufferSizeBytes)
	for {
		n, err := s.proc.Read(buf)
		if err != nil {
			s.close()
			return
		}
		if n == 0 {
			continue
		}

		s.mu.Lock()
		c := s.client
		s.mu.Unlock()

		if c != nil {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			select {
			case c.send <- chunk:
			default:
				// TODO: should we be dropping data?
			}
		}
		s.ImAlive()
	}
}

func (s *Session) WriteToTTY(p []byte) (int, error) {
	s.ImAlive()
	return s.proc.Write(p)
}

func (s *Session) Attach(ctx context.Context, ws *websocket.Conn) *Client {
	ctx, cancel := context.WithCancel(ctx)

	// create a new client from the websocket
	client := &Client{ws: ws, send: make(chan []byte, 256), cancel: cancel}

	// detach the old client if it exists
	s.Detach()

	// attach the new client
	s.mu.Lock()
	s.client = client
	s.mu.Unlock()

	// copy output from terminal to websocket
	go func() {
		defer cancel()
		for msg := range client.send {
			if err := client.ws.Write(ctx, websocket.MessageBinary, msg); err != nil {
				break
			}
		}
	}()

	go func() {
		<-s.done
		cancel()
	}()

	s.ImAlive()

	return client
}

func (s *Session) ImAlive() {
	s.lastActive.Store(time.Now())
}

func (s *Session) Detach() {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()

	if client != nil {
		logger.Info("Detatching session")
		close(client.send)
		client.cancel()
		_ = client.ws.Close(websocket.StatusNormalClosure, "Detatched")
		s.client = nil
	}

	s.ImAlive()
}

func (s *Session) Resize(rows, cols uint16) error {
	return pty.Setsize(s.proc.tty, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

func (s *Session) close() {
	select {
	case <-s.done:
		return
	default:
		close(s.done)
	}
	_ = s.proc.Kill()
	//if s.onClose != nil {
	//	s.onClose()
	//}
}

func NewSessionManager(ttl time.Duration) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		ttl:      ttl,
	}
}

func (sm *SessionManager) GetSession(id string, newProc func() (*ShellProcess, error)) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if s, ok := sm.sessions[id]; ok {
		return s, nil
	}

	proc, err := newProc()
	if err != nil {
		return nil, err
	}

	session := NewSession(id, proc, sm.ttl)
	sm.sessions[id] = session

	return session, nil
}
