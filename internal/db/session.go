package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Session struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID        int       `json:"id"`
	SessionID int       `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Assumption struct {
	ID        int       `json:"id"`
	SessionID int       `json:"session_id"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Decision struct {
	ID        int       `json:"id"`
	SessionID int       `json:"session_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionStore struct {
	pool *pgxpool.Pool
}

func NewSessionStore(pool *pgxpool.Pool) *SessionStore {
	return &SessionStore{pool: pool}
}

func (s *SessionStore) CreateSession(ctx context.Context, title string) (*Session, error) {
	var session Session
	err := s.pool.QueryRow(
		ctx,
		"INSERT INTO sessions (title) VALUES ($1) RETURNING id, title, created_at, updated_at",
		title,
	).Scan(&session.ID, &session.Title, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return &session, nil
}

func (s *SessionStore) DeleteSession(ctx context.Context, sessionID int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM sessions WHERE id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *SessionStore) SetTitle(ctx context.Context, sessionID int, title string) error {
	_, err := s.pool.Exec(ctx, "UPDATE sessions SET title = $1, updated_at = NOW() WHERE id = $2", title, sessionID)
	if err != nil {
		return fmt.Errorf("set title: %w", err)
	}
	return nil
}

func (s *SessionStore) TouchSession(ctx context.Context, sessionID int) error {
	_, err := s.pool.Exec(ctx, "UPDATE sessions SET updated_at = NOW() WHERE id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

func (s *SessionStore) ClearMessages(ctx context.Context, sessionID int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM messages WHERE session_id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("clear messages: %w", err)
	}
	return nil
}

func (s *SessionStore) ClearAssumptions(ctx context.Context, sessionID int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM assumptions WHERE session_id = $1", sessionID)
	if err != nil {
		return fmt.Errorf("clear assumptions: %w", err)
	}
	return nil
}

func (s *SessionStore) SaveMessage(ctx context.Context, sessionID int, role, content string) error {
	_, err := s.pool.Exec(
		ctx,
		"INSERT INTO messages (session_id, role, content) VALUES ($1, $2, $3); UPDATE sessions SET updated_at = NOW() WHERE id = $1",
		sessionID, role, content,
	)
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	return nil
}

func (s *SessionStore) GetMessages(ctx context.Context, sessionID int) ([]Message, error) {
	rows, err := s.pool.Query(
		ctx,
		"SELECT id, session_id, role, content, created_at FROM messages WHERE session_id = $1 ORDER BY created_at ASC",
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (s *SessionStore) ListSessions(ctx context.Context) ([]Session, error) {
	rows, err := s.pool.Query(
		ctx,
		"SELECT id, title, created_at, updated_at FROM sessions ORDER BY updated_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	return scanSessions(rows)
}

func (s *SessionStore) RecentSessions(ctx context.Context, limit int) ([]Session, error) {
	rows, err := s.pool.Query(
		ctx,
		"SELECT id, title, created_at, updated_at FROM sessions ORDER BY updated_at DESC LIMIT $1",
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recent sessions: %w", err)
	}
	defer rows.Close()

	return scanSessions(rows)
}

func scanSessions(rows pgx.Rows) ([]Session, error) {
	var sessions []Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.Title, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}
func (s *SessionStore) SaveAssumption(ctx context.Context, sessionID int, content string) error {
	_, err := s.pool.Exec(
		ctx,
		"INSERT INTO assumptions (session_id, content) VALUES ($1, $2)",
		sessionID, content,
	)
	if err != nil {
		return fmt.Errorf("save assumption: %w", err)
	}
	return nil
}

func (s *SessionStore) GetAssumptions(ctx context.Context, sessionID int) ([]Assumption, error) {
	rows, err := s.pool.Query(
		ctx,
		"SELECT id, session_id, content, status, created_at FROM assumptions WHERE session_id = $1 ORDER BY created_at ASC",
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("get assumptions: %w", err)
	}
	defer rows.Close()

	var assumptions []Assumption
	for rows.Next() {
		var a Assumption
		if err := rows.Scan(&a.ID, &a.SessionID, &a.Content, &a.Status, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan assumption: %w", err)
		}
		assumptions = append(assumptions, a)
	}
	return assumptions, rows.Err()
}

func (s *SessionStore) SaveDecision(ctx context.Context, sessionID int, content string) error {
	_, err := s.pool.Exec(
		ctx,
		"INSERT INTO decisions (session_id, content) VALUES ($1, $2)",
		sessionID, content,
	)
	if err != nil {
		return fmt.Errorf("save decision: %w", err)
	}
	return nil
}

func (s *SessionStore) GetDecisions(ctx context.Context, sessionID int) ([]Decision, error) {
	rows, err := s.pool.Query(
		ctx,
		"SELECT id, session_id, content, created_at FROM decisions WHERE session_id = $1 ORDER BY created_at ASC",
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("get decisions: %w", err)
	}
	defer rows.Close()

	var decisions []Decision
	for rows.Next() {
		var d Decision
		if err := rows.Scan(&d.ID, &d.SessionID, &d.Content, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		decisions = append(decisions, d)
	}
	return decisions, rows.Err()
}
