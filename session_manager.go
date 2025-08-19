package main

import (
	"fmt"
	"sync"
	"time"
	"webshell/ttyrec"
)

type SessionManager struct {
	sessions  map[string]*Session
	mu        sync.Mutex
	ttl       time.Duration
	auditTTY  bool
	auditExec bool
	auditPath string
}

func NewSessionManager(config Config) *SessionManager {
	return &SessionManager{
		sessions:  make(map[string]*Session),
		ttl:       config.Grace,
		auditTTY:  config.AuditTTY,
		auditExec: config.AuditExec,
		auditPath: config.AuditPath,
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

	var rec ttyrec.TTYRecorder = nil

	if sm.auditTTY {
		timestamp := time.Now().Format(time.RFC3339)
		auditFile := fmt.Sprintf("%s_%s.tty.audit", timestamp, id)
		rec, err = ttyrec.NewRecorder(sm.auditPath, auditFile)
		if err != nil {
			return nil, err
		}
	}

	session := NewSession(id, proc, sm.ttl, rec)
	sm.sessions[id] = session

	go func() {
		<-session.Done()
		sm.mu.Lock()
		if cur, ok := sm.sessions[id]; ok && cur == session {
			logger.Info("Removed session")
			delete(sm.sessions, id)
		}
		sm.mu.Unlock()
	}()

	return session, nil
}

func (sm *SessionManager) ExpireSessions() {

	// check each session, see if expired

	// use the most recent lastActive

	for _, session := range sm.sessions {
		idleDuration := time.Since(session.lastActive.Load().(time.Time))
		if idleDuration >= sm.ttl {
			session.Done()
		}
	}
}
