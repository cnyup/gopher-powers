// Package store 对齐 server/accounts.mjs（modernc/sqlite，表结构与 guests.sqlite 兼容）。
package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// User 对齐 SELECT id,name,wins,losses,draws 视图。
type User struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Wins   int    `json:"wins"`
	Losses int    `json:"losses"`
	Draws  int    `json:"draws"`
}

// Accounts 对齐 accounts.mjs Accounts。
type Accounts struct {
	db         *sql.DB
	cookieName string
}

func hashToken(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

// Open 建库（WAL + 0700 目录 + 0600 文件，表结构兼容源）。
func Open(directory, cookieName string) (*Accounts, error) {
	if cookieName != "gp_guest" && cookieName != "gp_six_guest" {
		return nil, errors.New("会话名称无效")
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	file := filepath.Join(directory, "guests.sqlite")
	db, err := sql.Open("sqlite", file)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(file, 0o600)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;
CREATE TABLE IF NOT EXISTS guests(id TEXT PRIMARY KEY, name TEXT NOT NULL, token TEXT UNIQUE NOT NULL, expires INTEGER NOT NULL, wins INTEGER DEFAULT 0, losses INTEGER DEFAULT 0, draws INTEGER DEFAULT 0);
CREATE TABLE IF NOT EXISTS matches(id TEXT PRIMARY KEY, a TEXT, b TEXT, winner TEXT, at INTEGER);`); err != nil {
		db.Close()
		return nil, err
	}
	return &Accounts{db: db, cookieName: cookieName}, nil
}

// Close 关库。
func (a *Accounts) Close() error { return a.db.Close() }

// CookieName 会话 cookie 名。
func (a *Accounts) CookieName() string { return a.cookieName }

// Find 对齐 find(cookie)：token 命中且未过期。
func (a *Accounts) Find(cookie string) *User {
	token := parseCookie(cookie, a.cookieName)
	if token == "" {
		return nil
	}
	u := &User{}
	err := a.db.QueryRow(`SELECT id,name,wins,losses,draws FROM guests WHERE token=? AND expires>?`,
		hashToken(token), time.Now().UnixMilli()).Scan(&u.ID, &u.Name, &u.Wins, &u.Losses, &u.Draws)
	if err != nil {
		return nil
	}
	return u
}

// Create 对齐 create(name)：返回 token + user。
func (a *Accounts) Create(name string) (string, *User, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, err
	}
	token := hex.EncodeToString(tokenBytes)
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", nil, err
	}
	// UUID v4 形态（对齐 randomUUID）。
	id := fmt.Sprintf("%s-%s-4%s-%s%s-%s",
		hex.EncodeToString(idBytes[0:4]), hex.EncodeToString(idBytes[4:6]),
		hex.EncodeToString(idBytes[6:7])[1:], hex.EncodeToString([]byte{0x80 | idBytes[7]&0x3f})[:1],
		hex.EncodeToString(idBytes[8:10])[1:], hex.EncodeToString(idBytes[10:16]))
	if _, err := a.db.Exec(`INSERT INTO guests(id,name,token,expires) VALUES(?,?,?,?)`,
		id, name, hashToken(token), time.Now().UnixMilli()+30*86400000); err != nil {
		return "", nil, err
	}
	return token, &User{ID: id, Name: name}, nil
}

// Rename 对齐 rename(id,name)。
func (a *Accounts) Rename(id, name string) error {
	_, err := a.db.Exec(`UPDATE guests SET name=? WHERE id=?`, name, id)
	return err
}

// Settle 对齐 settle(id,users,winner)：INSERT OR IGNORE matches + 胜负场累计。
func (a *Accounts) Settle(id string, users [2]*User, winner string) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT OR IGNORE INTO matches VALUES(?,?,?,?,?)`,
		id, users[0].ID, users[1].ID, winner, time.Now().UnixMilli())
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		for side, u := range users {
			field := "losses"
			switch {
			case winner == "draw":
				field = "draws"
			case winner == fmt.Sprint(side):
				field = "wins"
			}
			if _, err := tx.Exec(fmt.Sprintf(`UPDATE guests SET %[1]s=%[1]s+1 WHERE id=?`, field), u.ID); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func parseCookie(cookie, name string) string {
	for _, part := range splitSemicolon(cookie) {
		if len(part) > len(name)+1 && part[:len(name)+1] == name+"=" {
			return part[len(name)+1:]
		}
	}
	return ""
}

func splitSemicolon(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ';' {
			out = append(out, trimSpace(s[start:i]))
			start = i + 1
		}
	}
	out = append(out, trimSpace(s[start:]))
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
