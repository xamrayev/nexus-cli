package server

import (
	"sync"
	"time"

	"github.com/xamrayev/nexus-cli/pkg/deepseek"
)

// SessionManager manages conversation sessions isolated by agent/user ID.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*deepseek.AgentSession
	ttl      time.Duration
}

// NewSessionManager creates a SessionManager with the given idle TTL.
func NewSessionManager(ttl time.Duration) *SessionManager {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}

	sm := &SessionManager{
		sessions: make(map[string]*deepseek.AgentSession),
		ttl:      ttl,
	}

	go sm.startCleaner()
	return sm
}

// GetOrCreate returns an existing session or creates a new one for agentID.
func (sm *SessionManager) GetOrCreate(agentID string) *deepseek.AgentSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, exists := sm.sessions[agentID]
	if !exists {
		s = &deepseek.AgentSession{
			CreatedAt:      time.Now(),
			LastActivityAt: time.Now(),
		}
		sm.sessions[agentID] = s
	} else {
		s.LastActivityAt = time.Now()
	}

	return s
}

// Reset clears the session for a specific agent or all agents if agentID is "all" or empty.
func (sm *SessionManager) Reset(agentID string) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if agentID == "" || agentID == "all" {
		count := len(sm.sessions)
		sm.sessions = make(map[string]*deepseek.AgentSession)
		return count
	}

	if _, ok := sm.sessions[agentID]; ok {
		delete(sm.sessions, agentID)
		return 1
	}

	return 0
}

// List returns a snapshot of active agent sessions.
func (sm *SessionManager) List() map[string]map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]map[string]interface{})
	for id, s := range sm.sessions {
		result[id] = map[string]interface{}{
			"remote_id":        s.ID,
			"created_at":       s.CreatedAt.Format(time.RFC3339),
			"last_activity_at": s.LastActivityAt.Format(time.RFC3339),
			"message_count":    s.MessageCount,
		}
	}

	return result
}

func (sm *SessionManager) startCleaner() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.sweep()
	}
}

func (sm *SessionManager) sweep() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for id, s := range sm.sessions {
		if now.Sub(s.LastActivityAt) > sm.ttl*2 {
			delete(sm.sessions, id)
		}
	}
}
