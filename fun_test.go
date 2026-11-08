package main

import (
	"testing"
	"time"
)

func TestParseDM(t *testing.T) {
	for in, want := range map[string][2]int{"15.03": {3, 15}, " 1.1 ": {1, 1}, "29.02": {2, 29}, "15.03.1990": {3, 15}} {
		m, d, ok := parseDM(in)
		if !ok || m != want[0] || d != want[1] {
			t.Errorf("parseDM(%q) = %d,%d,%v", in, m, d, ok)
		}
	}
	for _, in := range []string{"", "31.02", "32.01", "15.13", "0.5", "abc", "15", "15,03"} {
		if _, _, ok := parseDM(in); ok {
			t.Errorf("parseDM(%q) must fail", in)
		}
	}
}

func TestBirthdayToday(t *testing.T) {
	day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 12, 0, 0, 0, time.UTC) }
	if !birthdayToday(3, 15, day(2026, 3, 15)) || birthdayToday(3, 15, day(2026, 3, 16)) {
		t.Fatal("plain date")
	}
	if !birthdayToday(2, 29, day(2027, 2, 28)) {
		t.Fatal("29 Feb must be greeted on 28 Feb in a common year")
	}
	if birthdayToday(2, 29, day(2028, 2, 28)) || !birthdayToday(2, 29, day(2028, 2, 29)) {
		t.Fatal("29 Feb in a leap year")
	}
}

func TestYearsWord(t *testing.T) {
	for n, want := range map[int]string{1: "год", 2: "года", 4: "года", 5: "лет", 11: "лет", 12: "лет", 21: "год", 22: "года", 100: "лет", 111: "лет"} {
		if got := yearsWord(n); got != want {
			t.Errorf("yearsWord(%d) = %s, want %s", n, got, want)
		}
	}
}

func TestBirthdayStore(t *testing.T) {
	s, _ := OpenStore(":memory:")
	s.Join(-1, 1, "Вася")
	s.Join(-2, 1, "Вася")
	s.Join(-1, 2, "Петя")
	s.SetBirthday(1, 3, 15, false)
	s.SetBirthday(1, 4, 20, false)
	if m, d, ok, _ := s.GetBirthday(1); !ok || m != 4 || d != 20 {
		t.Fatalf("profile date must update profile date: %d.%d %v", d, m, ok)
	}
	s.SetBirthday(1, 5, 5, true)
	s.SetBirthday(1, 6, 6, false)
	if m, d, _, _ := s.GetBirthday(1); m != 5 || d != 5 {
		t.Fatalf("profile must not overwrite manual date: %d.%d", d, m)
	}
	s.SetBirthday(1, 7, 7, true)
	if m, d, _, _ := s.GetBirthday(1); m != 7 || d != 7 {
		t.Fatalf("manual must overwrite manual: %d.%d", d, m)
	}
	if _, _, ok, _ := s.GetBirthday(2); ok {
		t.Fatal("no date for user 2")
	}
	list, _ := s.Birthdays()
	if len(list) != 2 {
		t.Fatalf("birthday must appear in both chats, got %d", len(list))
	}
	s.Leave(-2, 1)
	if list, _ = s.Birthdays(); len(list) != 1 {
		t.Fatalf("inactive player must not be greeted, got %d", len(list))
	}
}

func TestDrawExcept(t *testing.T) {
	s, _ := OpenStore(":memory:")
	for u := int64(1); u <= 3; u++ {
		s.Join(-1, u, "игрок")
	}
	for d := 1; d <= 40; d++ {
		day := time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		if u, _, _, ok, _ := s.DrawExcept(-1, day, []int64{1}); !ok || u == 1 {
			t.Fatalf("excluded user won or no draw: u=%d ok=%v", u, ok)
		}
	}

	s2, _ := OpenStore(":memory:")
	s2.Join(-2, 1, "Вася")
	s2.Join(-2, 2, "Петя")
	if u, _, _, ok, _ := s2.DrawExcept(-2, "2026-01-01", []int64{1}); !ok || u != 2 {
		t.Fatalf("with two players the other one must win: u=%d ok=%v", u, ok)
	}
	if u, _, fresh, ok, _ := s2.DrawExcept(-2, "2026-01-02", []int64{1, 2}); !ok || !fresh || u == 0 {
		t.Fatalf("everyone excluded must fall back to everyone: u=%d ok=%v", u, ok)
	}
	s3, _ := OpenStore(":memory:")
	s3.Join(-3, 1, "Вася")
	if _, _, _, ok, _ := s3.DrawExcept(-3, "2026-01-01", []int64{1}); ok {
		t.Fatal("a single player never makes a draw")
	}
}

func TestGreetOncePerYear(t *testing.T) {
	s, _ := OpenStore(":memory:")
	if ok, _ := s.Greet(-1, 1, "birthday", 2026); !ok {
		t.Fatal("first greeting")
	}
	if ok, _ := s.Greet(-1, 1, "birthday", 2026); ok {
		t.Fatal("same year must not repeat")
	}
	if ok, _ := s.Greet(-1, 1, "birthday", 2027); !ok {
		t.Fatal("next year is a new greeting")
	}
}

func TestQuotes(t *testing.T) {
	s, _ := OpenStore(":memory:")
	if _, ok, _ := s.RandomQuote(-1); ok {
		t.Fatal("empty chat has no quotes")
	}
	if fresh, _ := s.AddQuote(-1, 10, "Вася", "Привет", "2026-11-08"); !fresh {
		t.Fatal("first quote")
	}
	if fresh, _ := s.AddQuote(-1, 10, "Вася", "Привет", "2026-11-08"); fresh {
		t.Fatal("same message must not be saved twice")
	}
	q, ok, _ := s.RandomQuote(-1)
	if !ok || q.Author != "Вася" || q.Text != "Привет" {
		t.Fatalf("quote = %+v ok=%v", q, ok)
	}
	if _, ok, _ := s.RandomQuote(-2); ok {
		t.Fatal("quotes must not leak between chats")
	}
}

func TestFirstDrawAndChats(t *testing.T) {
	s, _ := OpenStore(":memory:")
	s.Join(-1, 1, "Вася")
	s.Join(-1, 2, "Петя")
	if d, _ := s.FirstDraw(-1); d != "" {
		t.Fatalf("no draws yet, got %q", d)
	}
	s.Draw(-1, "2026-05-02")
	s.Draw(-1, "2026-03-01")
	if d, _ := s.FirstDraw(-1); d != "2026-03-01" {
		t.Fatalf("first draw = %q", d)
	}
	if chats, _ := s.Chats(); len(chats) != 1 || chats[0] != -1 {
		t.Fatalf("chats = %v", chats)
	}
}
