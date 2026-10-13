package main

import (
	"strings"
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

func TestTodayUsesTZ(t *testing.T) {
	msk, _ := time.LoadLocation("Europe/Moscow")
	if got := today(time.Date(2026, 10, 13, 22, 30, 0, 0, time.UTC), msk); got != "2026-10-14" {
		t.Fatalf("today = %s, want 2026-10-14", got)
	}
}

func TestPhrasesRender(t *testing.T) {
	pools := [][]string{Openings, Spins, Sectors, Gifts, Letters, Reveals, Winners, Already, Joined,
		{AlreadyJoined, LeftGame, NotJoined, LeftChat, NotEnough, PrivateOnly, EmptyTop, MyScore, Help}}
	for _, j := range Jubilees {
		pools = append(pools, []string{j})
	}
	for _, pool := range pools {
		for _, p := range pool {
			out := render(p, "{name}", "X", "{n}", "3", "{year}", "1", "{all}", "2")
			if strings.ContainsAny(out, "{}<&") {
				t.Errorf("unrendered placeholder or raw HTML char: %q", out)
			}
		}
	}
}
