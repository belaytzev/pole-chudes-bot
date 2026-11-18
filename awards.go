package main

import (
	"slices"
	"time"
)

type stats struct{ Wins, BestStreak, Beauty int }

type award struct {
	Name string
	Need stats
}

var Awards = []award{
	{"🎬 Дебютант", stats{Wins: 1}},
	{"🎟 Постоянный клиент", stats{Wins: 5}},
	{"🎖 Ветеран барабана", stats{Wins: 10}},
	{"🥈 Серебряный призёр", stats{Wins: 25}},
	{"🏛 Экспонат музея", stats{Wins: 50}},
	{"🔑 Хозяин студии", stats{Wins: 100}},
	{"🎯 Дубль", stats{BestStreak: 2}},
	{"🎩 Хет-трик", stats{BestStreak: 3}},
	{"🧲 Любимец барабана", stats{BestStreak: 5}},
	{"📺 Неделя в эфире", stats{BestStreak: 7}},
	{"🌹 Первый букет", stats{Beauty: 1}},
	{"📸 Лицо с обложки", stats{Beauty: 10}},
	{"👑 Краса студии", stats{Beauty: 50}},
}

func earned(s stats) []string {
	var out []string
	for _, a := range Awards {
		if s.Wins >= a.Need.Wins && s.BestStreak >= a.Need.BestStreak && s.Beauty >= a.Need.Beauty {
			out = append(out, a.Name)
		}
	}
	return out
}

func newAwards(before, after stats) []string {
	had := earned(before)
	return slices.DeleteFunc(earned(after), func(a string) bool { return slices.Contains(had, a) })
}

// runs turns sorted win days into lengths of consecutive-day series, oldest first.
func runs(days []string) []int {
	var out []int
	var prev time.Time
	for _, d := range days {
		t, err := time.Parse(time.DateOnly, d)
		if err != nil {
			continue
		}
		if len(out) > 0 && t.Equal(prev.AddDate(0, 0, 1)) {
			out[len(out)-1]++
		} else {
			out = append(out, 1)
		}
		prev = t
	}
	return out
}

func best(runs []int) int {
	if len(runs) == 0 {
		return 0
	}
	return slices.Max(runs)
}
