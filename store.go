package main

import (
	"database/sql"
	"encoding/json"
	"errors"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS players (
	chat_id INTEGER NOT NULL,
	user_id INTEGER NOT NULL,
	name    TEXT    NOT NULL,
	active  INTEGER NOT NULL DEFAULT 1,
	PRIMARY KEY (chat_id, user_id)
);
CREATE TABLE IF NOT EXISTS draws (
	chat_id INTEGER NOT NULL,
	day     TEXT    NOT NULL,
	user_id INTEGER NOT NULL,
	PRIMARY KEY (chat_id, day)
);
CREATE TABLE IF NOT EXISTS beauties (
	chat_id INTEGER NOT NULL,
	day     TEXT    NOT NULL,
	user_id INTEGER NOT NULL,
	PRIMARY KEY (chat_id, day)
);`

type Store struct{ db *sql.DB }

type Score struct {
	Name  string
	Count int
}

// temp_store(memory): with readOnlyRootFilesystem SQLite can't spill temp files to /tmp.
func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=temp_store(memory)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema + schemaFun); err != nil {
		return nil, err
	}
	return &Store{db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Migrate moves a chat's data to its new id after a group is upgraded to a supergroup.
func (s *Store) Migrate(from, to int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{
		`UPDATE OR IGNORE players SET chat_id = ? WHERE chat_id = ?`,
		`UPDATE OR IGNORE draws SET chat_id = ? WHERE chat_id = ?`,
		`UPDATE OR IGNORE beauties SET chat_id = ? WHERE chat_id = ?`,
	} {
		if _, err := tx.Exec(q, to, from); err != nil {
			return err
		}
	}
	for _, q := range []string{`DELETE FROM players WHERE chat_id = ?`, `DELETE FROM draws WHERE chat_id = ?`, `DELETE FROM beauties WHERE chat_id = ?`} {
		if _, err := tx.Exec(q, from); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Join(chat, user int64, name string) (bool, error) {
	res, err := s.db.Exec(`UPDATE players SET name = ? WHERE chat_id = ? AND user_id = ? AND active = 1`, name, chat, user)
	if already, err := affected(res, err); already || err != nil {
		return false, err
	}
	res, err = s.db.Exec(`INSERT INTO players (chat_id, user_id, name) VALUES (?, ?, ?)
		ON CONFLICT (chat_id, user_id) DO UPDATE SET name = excluded.name, active = 1 WHERE active = 0`,
		chat, user, name)
	return affected(res, err)
}

func (s *Store) Leave(chat, user int64) (bool, error) {
	res, err := s.db.Exec(`UPDATE players SET active = 0 WHERE chat_id = ? AND user_id = ? AND active = 1`, chat, user)
	return affected(res, err)
}

func (s *Store) Players(chat int64) (n int, err error) {
	err = s.db.QueryRow(`SELECT COUNT(*) FROM players WHERE chat_id = ? AND active = 1`, chat).Scan(&n)
	return
}

// Draw picks today's winner once per chat; concurrent calls get the same row via the PK.
// ok=false means nobody was drawn today and there are fewer than two players.
func (s *Store) Draw(chat int64, day string) (user int64, name string, fresh, ok bool, err error) {
	return s.draw(chat, day, "[]")
}

// DrawExcept skips the excluded users; if that leaves nobody, everyone takes part again.
func (s *Store) DrawExcept(chat int64, day string, excl []int64) (user int64, name string, fresh, ok bool, err error) {
	if len(excl) > 0 {
		b, _ := json.Marshal(excl)
		if user, name, fresh, ok, err = s.draw(chat, day, string(b)); err != nil || ok {
			return
		}
	}
	return s.draw(chat, day, "[]")
}

func (s *Store) draw(chat int64, day, excl string) (user int64, name string, fresh, ok bool, err error) {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO draws (chat_id, day, user_id)
		SELECT ?, ?, user_id FROM players
		WHERE chat_id = ? AND active = 1
		  AND user_id NOT IN (SELECT value FROM json_each(?))
		  AND (SELECT COUNT(*) FROM players WHERE chat_id = ? AND active = 1) >= 2
		ORDER BY RANDOM() LIMIT 1`, chat, day, chat, excl, chat)
	if fresh, err = affected(res, err); err != nil {
		return
	}
	err = s.db.QueryRow(`SELECT d.user_id, p.name FROM draws d
		JOIN players p ON p.chat_id = d.chat_id AND p.user_id = d.user_id
		WHERE d.chat_id = ? AND d.day = ?`, chat, day).Scan(&user, &name)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", false, false, nil
	}
	return user, name, fresh, err == nil, err
}

// Beauty picks today's beauty once per chat, never the same person as the winner.
func (s *Store) Beauty(chat int64, day string, winner int64) (user int64, name string, ok bool, err error) {
	if _, err = s.db.Exec(`INSERT OR IGNORE INTO beauties (chat_id, day, user_id)
		SELECT ?, ?, user_id FROM players
		WHERE chat_id = ? AND active = 1 AND user_id != ?
		ORDER BY RANDOM() LIMIT 1`, chat, day, chat, winner); err != nil {
		return
	}
	err = s.db.QueryRow(`SELECT b.user_id, p.name FROM beauties b
		JOIN players p ON p.chat_id = b.chat_id AND p.user_id = b.user_id
		WHERE b.chat_id = ? AND b.day = ?`, chat, day).Scan(&user, &name)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", false, nil
	}
	return user, name, err == nil, err
}

func (s *Store) Wins(chat, user int64, since string) (int, error) {
	return s.wins("draws", chat, user, since)
}

func (s *Store) BeautyWins(chat, user int64, since string) (int, error) {
	return s.wins("beauties", chat, user, since)
}

func (s *Store) wins(table string, chat, user int64, since string) (n int, err error) {
	err = s.db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE chat_id = ? AND user_id = ? AND day >= ?`,
		chat, user, since).Scan(&n)
	return
}

func (s *Store) Top(chat int64, since string) ([]Score, error) { return s.top("draws", chat, since) }

func (s *Store) BeautyTop(chat int64, since string) ([]Score, error) {
	return s.top("beauties", chat, since)
}

func (s *Store) top(table string, chat int64, since string) ([]Score, error) {
	rows, err := s.db.Query(`SELECT p.name, COUNT(*) AS c FROM `+table+` d
		JOIN players p ON p.chat_id = d.chat_id AND p.user_id = d.user_id
		WHERE d.chat_id = ? AND d.day >= ?
		GROUP BY d.user_id ORDER BY c DESC, p.name LIMIT 20`, chat, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var top []Score
	for rows.Next() {
		var sc Score
		if err := rows.Scan(&sc.Name, &sc.Count); err != nil {
			return nil, err
		}
		top = append(top, sc)
	}
	return top, rows.Err()
}

func affected(res sql.Result, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}
