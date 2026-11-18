package main

import (
	"slices"
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

func TestStreak(t *testing.T) {
	s, _ := OpenStore(":memory:")
	const chat = -100
	for day, user := range map[string]int64{"2026-02-27": 1, "2026-02-28": 1, "2026-03-01": 1, "2026-03-02": 2, "2026-03-04": 2} {
		if _, err := s.db.Exec(`INSERT INTO draws (chat_id, day, user_id) VALUES (?, ?, ?)`, chat, day, user); err != nil {
			t.Fatal(err)
		}
	}
	for user, want := range map[int64][]int{1: {3}, 2: {1, 1}, 3: nil} {
		days, err := s.WinDays(chat, user)
		if got := runs(days); err != nil || !slices.Equal(got, want) {
			t.Errorf("runs(user %d) = %v, %v; want %v", user, got, err, want)
		}
	}
}

func TestAwards(t *testing.T) {
	if got := newAwards(stats{}, stats{Wins: 1, BestStreak: 1}); !slices.Equal(got, []string{"🎬 Дебютант"}) {
		t.Errorf("first win: %v", got)
	}
	if got := newAwards(stats{Wins: 1, BestStreak: 1}, stats{Wins: 2, BestStreak: 2}); !slices.Equal(got, []string{"🎯 Дубль"}) {
		t.Errorf("second day in a row: %v", got)
	}
	if got := newAwards(stats{Wins: 3, BestStreak: 2}, stats{Wins: 4, BestStreak: 2}); len(got) != 0 {
		t.Errorf("a repeated streak must not award again: %v", got)
	}
	if got := earned(stats{Wins: 10, BestStreak: 3, Beauty: 1}); len(got) != 6 {
		t.Errorf("earned: %v", got)
	}
}

func TestDeal(t *testing.T) {
	s, _ := OpenStore(":memory:")
	pool := []string{"a", "b", "c", "d"}
	for round := 0; round < 3; round++ {
		seen := map[string]bool{}
		for range pool {
			p, err := s.Deal(-1, "test", pool)
			if err != nil {
				t.Fatal(err)
			}
			seen[p] = true
		}
		if len(seen) != len(pool) {
			t.Fatalf("round %d: dealt %d of %d phrases", round, len(seen), len(pool))
		}
	}
}

func TestYearEndText(t *testing.T) {
	lead := yearEndText(2026, []Score{{"Вася", 5}, {"Петя", 3}}, nil)
	tie := yearEndText(2026, []Score{{"Вася", 5}, {"Петя", 5}}, []Score{{"Коля", 2}})
	if !strings.Contains(strings.ToLower(lead), "побед: 5") {
		t.Errorf("no champion line: %q", lead)
	}
	if strings.Contains(strings.ToLower(tie), "побед: 5") {
		t.Errorf("a tie must not name a champion: %q", tie)
	}
	for _, out := range []string{lead, tie} {
		if strings.ContainsAny(out, "{}") {
			t.Errorf("unrendered placeholder: %q", out)
		}
	}
}

func TestPickDealsWholePool(t *testing.T) {
	pool := []string{"a", "b", "c", "d", "e"}
	prev := ""
	for round := 0; round < 20; round++ {
		seen := map[string]bool{}
		for range pool {
			p := pick(pool)
			if p == prev {
				t.Fatalf("round %d: %q dealt twice in a row", round, p)
			}
			seen[p], prev = true, p
		}
		if len(seen) != len(pool) {
			t.Fatalf("round %d: dealt %d of %d phrases", round, len(seen), len(pool))
		}
	}
}

func TestPhrasesRender(t *testing.T) {
	pools := [][]string{Openings, Spins, Sectors, Gifts, Letters, Reveals, Winners, Already, Joined, Beauties, AlreadyBeauties,
		AlreadyJoined, LeftGame, NotJoined, LeftChat, NotEnough, PrivateOnly, EmptyTop, MyScore, {Help},
		BirthdayGreets, BirthdayRest, Anniversary, BirthdayBad, BirthdaySaved, BirthdayFromProfile, BirthdayKnown,
		BirthdayNotFound, QuoteSaved, QuoteDup, QuoteNoText, QuoteTooLong, QuoteEmpty}
	pools = append(pools, Streaks, Prizes, YearEnd, YearChampion, YearTie, AwardNew, NoAwards)
	for _, j := range Jubilees {
		pools = append(pools, j)
	}
	for _, pool := range pools {
		for _, p := range pool {
			out := render(p, "{name}", "X", "{n}", "3", "{year}", "1", "{all}", "2", "{byear}", "1", "{ball}", "2", "{date}", "01.01", "{years}", "1", "{word}", "год", "{count}", "5", "{award}", "Y")
			if strings.ContainsAny(out, "{}<&") {
				t.Errorf("unrendered placeholder or raw HTML char: %q", out)
			}
		}
	}
}
