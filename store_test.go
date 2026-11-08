package main

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStore(t *testing.T) {
	s, err := OpenStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	const chat = -100

	if _, _, _, ok, _ := s.Draw(chat, "2026-10-13"); ok {
		t.Fatal("draw with no players must fail")
	}
	s.Join(chat, 1, "Вася")
	if _, _, _, ok, _ := s.Draw(chat, "2026-10-13"); ok {
		t.Fatal("draw with one player must fail")
	}
	s.Join(chat, 2, "Петя")
	s.Join(chat, 3, "Коля")
	s.Leave(chat, 3)

	u1, _, fresh1, ok, err := s.Draw(chat, "2026-10-13")
	if err != nil || !ok || !fresh1 {
		t.Fatalf("first draw: ok=%v fresh=%v err=%v", ok, fresh1, err)
	}
	u2, _, fresh2, _, _ := s.Draw(chat, "2026-10-13")
	if u1 != u2 || fresh2 {
		t.Fatalf("second draw same day must return same winner: %d vs %d fresh=%v", u1, u2, fresh2)
	}

	for d := 1; d <= 50; d++ {
		u, _, _, _, _ := s.Draw(chat, time.Date(2025, 1, d, 0, 0, 0, 0, time.UTC).Format("2006-01-02"))
		if u == 3 {
			t.Fatal("inactive player won")
		}
	}

	year, _ := s.Top(chat, "2026-01-01")
	if len(year) != 1 || year[0].Count != 1 {
		t.Fatalf("year top must count only 2026: %+v", year)
	}
	all, _ := s.Top(chat, "")
	total := 0
	for _, sc := range all {
		total += sc.Count
	}
	if total != 51 {
		t.Fatalf("all-time total = %d, want 51", total)
	}
}

func TestDrawConcurrent(t *testing.T) {
	s, _ := OpenStore(":memory:")
	const chat = -200
	s.Join(chat, 1, "Вася")
	s.Join(chat, 2, "Петя")
	var wg sync.WaitGroup
	var fresh, winner atomic.Int64
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, _, f, ok, err := s.Draw(chat, "2026-10-13")
			if err != nil || !ok {
				t.Errorf("draw: ok=%v err=%v", ok, err)
				return
			}
			if f {
				fresh.Add(1)
			}
			winner.CompareAndSwap(0, u)
			if winner.Load() != u {
				t.Errorf("different winners in one day")
			}
		}()
	}
	wg.Wait()
	if fresh.Load() != 1 {
		t.Fatalf("fresh draws = %d, want 1", fresh.Load())
	}
}

func TestJoinRefreshesName(t *testing.T) {
	s, _ := OpenStore(":memory:")
	const chat = -300
	if fresh, _ := s.Join(chat, 1, "Вася"); !fresh {
		t.Fatal("first join must be fresh")
	}
	if fresh, _ := s.Join(chat, 1, "Василий"); fresh {
		t.Fatal("repeat join must not be fresh")
	}
	var name string
	s.db.QueryRow(`SELECT name FROM players WHERE chat_id = ? AND user_id = 1`, chat).Scan(&name)
	if name != "Василий" {
		t.Fatalf("name = %q, want Василий", name)
	}
	s.Leave(chat, 1)
	if fresh, _ := s.Join(chat, 1, "Вася"); !fresh {
		t.Fatal("join after leave must be fresh")
	}
}

func TestMigrate(t *testing.T) {
	s, _ := OpenStore(":memory:")
	s.Join(-1, 1, "Вася")
	s.Join(-1, 2, "Петя")
	s.Draw(-1, "2026-10-13")
	if err := s.Migrate(-1, -1001); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.Players(-1); n != 0 {
		t.Fatalf("old chat keeps %d players", n)
	}
	if n, _ := s.Players(-1001); n != 2 {
		t.Fatalf("new chat has %d players, want 2", n)
	}
	if _, _, fresh, ok, _ := s.Draw(-1001, "2026-10-13"); !ok || fresh {
		t.Fatalf("migrated draw must be kept: ok=%v fresh=%v", ok, fresh)
	}
}

func TestTodayUsesTZ(t *testing.T) {
	msk, _ := time.LoadLocation("Europe/Moscow")
	if got := today(time.Date(2026, 10, 13, 22, 30, 0, 0, time.UTC), msk); got != "2026-10-14" {
		t.Fatalf("today = %s, want 2026-10-14", got)
	}
}

func TestBeauty(t *testing.T) {
	s, err := OpenStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	s.Join(-7, 1, "A")
	s.Join(-7, 2, "B")
	s.Join(-7, 3, "C")
	var first int64
	for d := 1; d <= 30; d++ {
		day := time.Date(2025, 1, d, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		u, _, ok, err := s.Beauty(-7, day, 1)
		if err != nil || !ok || u == 1 {
			t.Fatalf("beauty: u=%d ok=%v err=%v", u, ok, err)
		}
		if d == 1 {
			first = u
		}
	}
	if u, _, _, _ := s.Beauty(-7, "2025-01-01", 1); u != first {
		t.Fatal("same day must return the same beauty")
	}
	if top, _ := s.BeautyTop(-7, ""); len(top) == 0 {
		t.Fatal("empty beauty top")
	}
	s.Join(-8, 1, "A")
	if _, _, ok, _ := s.Beauty(-8, "2025-01-01", 1); ok {
		t.Fatal("no beauty without a second player")
	}
}

func TestPhrasesRender(t *testing.T) {
	pools := [][]string{Openings, Spins, Sectors, Gifts, Letters, Reveals, Winners, Already, Joined, Beauties, AlreadyBeauties,
		AlreadyJoined, LeftGame, NotJoined, LeftChat, NotEnough, PrivateOnly, EmptyTop, MyScore, {Help},
		BirthdayGreets, BirthdayRest, Anniversary, BirthdayBad, BirthdaySaved, BirthdayFromProfile, BirthdayKnown,
		BirthdayNotFound, QuoteSaved, QuoteDup, QuoteNoText, QuoteTooLong, QuoteEmpty}
	for _, j := range Jubilees {
		pools = append(pools, []string{j})
	}
	for _, pool := range pools {
		for _, p := range pool {
			out := render(p, "{name}", "X", "{n}", "3", "{year}", "1", "{all}", "2", "{byear}", "1", "{ball}", "2", "{date}", "01.01", "{years}", "1", "{word}", "год")
			if strings.ContainsAny(out, "{}<&") {
				t.Errorf("unrendered placeholder or raw HTML char: %q", out)
			}
		}
	}
}
