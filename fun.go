package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

const (
	greetHour   = 10
	quoteMaxLen = 500
)

func parseDM(s string) (month, day int, ok bool) {
	var d, m int
	if n, err := fmt.Sscanf(strings.TrimSpace(s), "%d.%d", &d, &m); err != nil || n != 2 {
		return 0, 0, false
	}
	t := time.Date(2024, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	if int(t.Month()) != m || t.Day() != d {
		return 0, 0, false
	}
	return m, d, true
}

func fmtDM(month, day int) string { return fmt.Sprintf("%02d.%02d", day, month) }

func isLeap(y int) bool { return y%4 == 0 && (y%100 != 0 || y%400 == 0) }

// 29 February is greeted on 28 February in non-leap years.
func birthdayToday(month, day int, now time.Time) bool {
	if int(now.Month()) == month && now.Day() == day {
		return true
	}
	return month == 2 && day == 29 && now.Month() == time.February && now.Day() == 28 && !isLeap(now.Year())
}

func mention(id int64, name string) string {
	return `<a href="tg://user?id=` + strconv.FormatInt(id, 10) + `">` + html.EscapeString(name) + `</a>`
}

// profileBirthday asks Telegram for the date in the user's profile; ok=false when it is unset or hidden from bots.
func (a *app) profileBirthday(user int64) (month, day int, ok bool) {
	raw, err := a.bot.Raw("getChat", map[string]string{"chat_id": strconv.FormatInt(user, 10)})
	if err != nil {
		return 0, 0, false
	}
	var r struct {
		Result struct {
			Birthdate *struct {
				Day   int `json:"day"`
				Month int `json:"month"`
			} `json:"birthdate"`
		} `json:"result"`
	}
	if json.Unmarshal(raw, &r) != nil || r.Result.Birthdate == nil {
		return 0, 0, false
	}
	b := r.Result.Birthdate
	return b.Month, b.Day, b.Month != 0 && b.Day != 0
}

func (a *app) birthdayExcl(chat int64, now time.Time) []int64 {
	list, err := a.store.Birthdays()
	if err != nil {
		log.Printf("birthdays: %v", err)
		return nil
	}
	var ids []int64
	for _, b := range list {
		if b.Chat == chat && birthdayToday(b.Month, b.Day, now) {
			ids = append(ids, b.User)
		}
	}
	return ids
}

func (a *app) syncBirthday(user int64) {
	if m, d, ok := a.profileBirthday(user); ok {
		if err := a.store.SetBirthday(user, m, d, false); err != nil {
			log.Printf("save profile birthday: %v", err)
		}
	}
}

func (a *app) birthday(c tele.Context) error {
	u := c.Sender()
	if arg := strings.TrimSpace(c.Message().Payload); arg != "" {
		m, d, ok := parseDM(arg)
		if !ok {
			return c.Send(BirthdayBad)
		}
		if err := a.store.SetBirthday(u.ID, m, d, true); err != nil {
			return err
		}
		return c.Send(render(BirthdaySaved, "{name}", esc(u), "{date}", fmtDM(m, d)))
	}
	if m, d, ok := a.profileBirthday(u.ID); ok {
		if err := a.store.SetBirthday(u.ID, m, d, false); err != nil {
			return err
		}
	}
	m, d, ok, err := a.store.GetBirthday(u.ID)
	if err != nil {
		return err
	}
	if !ok {
		return c.Send(render(BirthdayNotFound, "{name}", esc(u)))
	}
	return c.Send(render(BirthdayKnown, "{name}", esc(u), "{date}", fmtDM(m, d)))
}

func (a *app) quote(c tele.Context) error {
	chat := c.Chat().ID
	if r := c.Message().ReplyTo; r != nil {
		text := strings.TrimSpace(r.Text)
		if text == "" {
			text = strings.TrimSpace(r.Caption)
		}
		if text == "" || r.Sender == nil || r.Sender.IsBot {
			return c.Send(QuoteNoText)
		}
		if len([]rune(text)) > quoteMaxLen {
			return c.Send(QuoteTooLong)
		}
		fresh, err := a.store.AddQuote(chat, r.ID, displayName(r.Sender), text, today(time.Now(), a.loc))
		if err != nil {
			return err
		}
		if !fresh {
			return c.Send(QuoteDup)
		}
		return c.Send(QuoteSaved)
	}
	q, ok, err := a.store.RandomQuote(chat)
	if err != nil {
		return err
	}
	if !ok {
		return c.Send(QuoteEmpty)
	}
	year := q.SavedAt
	if len(year) >= 4 {
		year = year[:4]
	}
	return c.Send("«" + html.EscapeString(q.Text) + "»\n— " + html.EscapeString(q.Author) + ", " + year)
}

func (a *app) greetLoop() {
	for {
		a.greet(time.Now().In(a.loc))
		time.Sleep(15 * time.Minute)
	}
}

func (a *app) greet(now time.Time) {
	if now.Hour() < greetHour {
		return
	}
	list, err := a.store.Birthdays()
	if err != nil {
		log.Printf("birthdays: %v", err)
		return
	}
	for _, b := range list {
		if !birthdayToday(b.Month, b.Day, now) {
			continue
		}
		if ok, err := a.store.Greet(b.Chat, b.User, "birthday", now.Year()); err != nil || !ok {
			continue
		}
		a.say(b.Chat, render(pick(BirthdayGreets), "{name}", mention(b.User, b.Name)))
	}
	chats, err := a.store.Chats()
	if err != nil {
		log.Printf("chats: %v", err)
		return
	}
	for _, chat := range chats {
		first, err := a.store.FirstDraw(chat)
		if err != nil || first == "" {
			continue
		}
		t, err := time.Parse("2006-01-02", first)
		years := now.Year() - t.Year()
		if err != nil || years < 1 || t.Month() != now.Month() || t.Day() != now.Day() {
			continue
		}
		if ok, err := a.store.Greet(chat, 0, "anniversary", now.Year()); err != nil || !ok {
			continue
		}
		a.say(chat, render(Anniversary, "{years}", strconv.Itoa(years), "{word}", yearsWord(years)))
	}
}

func (a *app) say(chat int64, text string) {
	if _, err := a.bot.Send(&tele.Chat{ID: chat}, text); err != nil {
		log.Printf("greet chat %d: %v", chat, err)
	}
}
