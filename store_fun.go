package main

import (
	"database/sql"
	"errors"
)

const schemaFun = `
CREATE TABLE IF NOT EXISTS birthdays (
	user_id INTEGER PRIMARY KEY,
	month   INTEGER NOT NULL,
	day     INTEGER NOT NULL,
	manual  INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS quotes (
	id       INTEGER PRIMARY KEY AUTOINCREMENT,
	chat_id  INTEGER NOT NULL,
	msg_id   INTEGER NOT NULL,
	author   TEXT    NOT NULL,
	text     TEXT    NOT NULL,
	saved_at TEXT    NOT NULL,
	UNIQUE (chat_id, msg_id)
);
CREATE TABLE IF NOT EXISTS greetings (
	chat_id INTEGER NOT NULL,
	user_id INTEGER NOT NULL,
	kind    TEXT    NOT NULL,
	year    INTEGER NOT NULL,
	PRIMARY KEY (chat_id, user_id, kind, year)
);`

type Birthday struct {
	Chat, User int64
	Name       string
	Month, Day int
}

type Quote struct{ Author, Text, SavedAt string }

// A date typed by hand is never overwritten by the one read from the profile.
func (s *Store) SetBirthday(user int64, month, day int, manual bool) error {
	m := 0
	if manual {
		m = 1
	}
	_, err := s.db.Exec(`INSERT INTO birthdays (user_id, month, day, manual) VALUES (?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET month = excluded.month, day = excluded.day, manual = excluded.manual
		WHERE excluded.manual = 1 OR birthdays.manual = 0`, user, month, day, m)
	return err
}

func (s *Store) GetBirthday(user int64) (month, day int, ok bool, err error) {
	err = s.db.QueryRow(`SELECT month, day FROM birthdays WHERE user_id = ?`, user).Scan(&month, &day)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, false, nil
	}
	return month, day, err == nil, err
}

func (s *Store) Birthdays() ([]Birthday, error) {
	rows, err := s.db.Query(`SELECT p.chat_id, p.user_id, p.name, b.month, b.day
		FROM players p JOIN birthdays b ON b.user_id = p.user_id WHERE p.active = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Birthday
	for rows.Next() {
		var b Birthday
		if err := rows.Scan(&b.Chat, &b.User, &b.Name, &b.Month, &b.Day); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Greet reports true once per (chat, user, kind, year), so a restart never repeats a greeting.
func (s *Store) Greet(chat, user int64, kind string, year int) (bool, error) {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO greetings (chat_id, user_id, kind, year) VALUES (?, ?, ?, ?)`,
		chat, user, kind, year)
	return affected(res, err)
}

func (s *Store) Chats() ([]int64, error) {
	rows, err := s.db.Query(`SELECT DISTINCT chat_id FROM players WHERE active = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) FirstDraw(chat int64) (day string, err error) {
	var d sql.NullString
	err = s.db.QueryRow(`SELECT MIN(day) FROM draws WHERE chat_id = ?`, chat).Scan(&d)
	return d.String, err
}

func (s *Store) AddQuote(chat int64, msg int, author, text, at string) (bool, error) {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO quotes (chat_id, msg_id, author, text, saved_at) VALUES (?, ?, ?, ?, ?)`,
		chat, msg, author, text, at)
	return affected(res, err)
}

func (s *Store) RandomQuote(chat int64) (q Quote, ok bool, err error) {
	err = s.db.QueryRow(`SELECT author, text, saved_at FROM quotes WHERE chat_id = ? ORDER BY RANDOM() LIMIT 1`, chat).
		Scan(&q.Author, &q.Text, &q.SavedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return q, false, nil
	}
	return q, err == nil, err
}
