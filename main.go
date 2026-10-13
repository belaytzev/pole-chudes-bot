package main

import (
	"html"
	"log"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	tele "gopkg.in/telebot.v3"
)

type app struct {
	bot   *tele.Bot
	store *Store
	loc   *time.Location
}

func main() {
	loc, err := time.LoadLocation(env("TZ", "Europe/Moscow"))
	if err != nil {
		log.Fatal(err)
	}
	store, err := OpenStore(env("DB_PATH", "bot.db"))
	if err != nil {
		log.Fatal(err)
	}
	bot, err := tele.NewBot(tele.Settings{
		Token:     os.Getenv("BOT_TOKEN"),
		Poller:    &tele.LongPoller{Timeout: 30 * time.Second},
		ParseMode: tele.ModeHTML,
		OnError:   func(err error, c tele.Context) { log.Printf("update error: %v", err) },
	})
	if err != nil {
		log.Fatal(err)
	}
	a := &app{bot, store, loc}

	bot.Handle("/start", a.help)
	bot.Handle("/help", a.help)
	g := bot.Group()
	g.Use(groupOnly)
	g.Handle("/igrok", a.join)
	g.Handle("/vyhod", a.leave)
	g.Handle("/baraban", a.spin)
	g.Handle("/kapital", a.yearTop)
	g.Handle("/zalslavy", a.allTop)
	g.Handle("/moi", a.mine)
	g.Handle(tele.OnUserLeft, a.userLeft)

	err = bot.SetCommands([]tele.Command{
		{Text: "igrok", Description: "Записаться в игроки"},
		{Text: "vyhod", Description: "Выйти из игры"},
		{Text: "baraban", Description: "Вращать барабан"},
		{Text: "kapital", Description: "Итоги года"},
		{Text: "zalslavy", Description: "Зал славы"},
		{Text: "moi", Description: "Мой счёт"},
	})
	if err != nil {
		log.Printf("set commands: %v", err)
	}
	log.Printf("started as @%s", bot.Me.Username)
	bot.Start()
}

func groupOnly(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if c.Chat().Type == tele.ChatPrivate {
			return c.Send(PrivateOnly)
		}
		return next(c)
	}
}

func (a *app) help(c tele.Context) error { return c.Send(Help) }

func (a *app) join(c tele.Context) error {
	u := c.Sender()
	fresh, err := a.store.Join(c.Chat().ID, u.ID, displayName(u))
	if err != nil {
		return err
	}
	if fresh {
		return c.Send(render(pick(Joined), "{name}", esc(u)))
	}
	return c.Send(render(AlreadyJoined, "{name}", esc(u)))
}

func (a *app) leave(c tele.Context) error {
	u := c.Sender()
	left, err := a.store.Leave(c.Chat().ID, u.ID)
	if err != nil {
		return err
	}
	if left {
		return c.Send(render(LeftGame, "{name}", esc(u)))
	}
	return c.Send(render(NotJoined, "{name}", esc(u)))
}

func (a *app) userLeft(c tele.Context) error {
	u := c.Message().UserLeft
	left, err := a.store.Leave(c.Chat().ID, u.ID)
	if err != nil || !left {
		return err
	}
	return c.Send(render(LeftChat, "{name}", esc(u)))
}

func (a *app) spin(c tele.Context) error {
	chat := c.Chat()
	players, err := a.store.Players(chat.ID)
	if err != nil {
		return err
	}
	user, name, fresh, ok, err := a.store.Draw(chat.ID, today(time.Now(), a.loc))
	if err != nil {
		return err
	}
	if !ok {
		return c.Send(NotEnough)
	}
	name = html.EscapeString(name)
	if !fresh {
		return c.Send(render(pick(Already), "{name}", name))
	}
	wins, err := a.store.Wins(chat.ID, user, "")
	if err != nil {
		return err
	}

	mention := `<a href="tg://user?id=` + strconv.FormatInt(user, 10) + `">` + name + `</a>`
	final := "П🟦🟦🟦Р дня\n\n" + pick(Reveals) + "\n\n" + render(pick(Winners), "{name}", mention)
	if j, ok := Jubilees[wins]; ok {
		final += "\n\n" + render(j, "{name}", name)
	}
	frames := []string{
		render(pick(Openings), "{n}", strconv.Itoa(players)),
		pick(Spins),
		pick(Sectors),
		pick(Gifts),
		pick(Letters),
	}

	msg, err := a.bot.Send(chat, frames[0])
	if err != nil {
		return err
	}
	for _, f := range frames[1:] {
		pause()
		if _, err := a.bot.Edit(msg, f); err != nil {
			log.Printf("edit frame: %v", err)
		}
	}
	pause()
	if _, err := a.bot.Edit(msg, final); err != nil {
		log.Printf("edit final: %v", err)
		return c.Send(final)
	}
	return nil
}

func (a *app) yearTop(c tele.Context) error {
	now := time.Now().In(a.loc)
	return a.top(c, "🏆 Капитал-шоу "+strconv.Itoa(now.Year()), now.Format("2006")+"-01-01")
}

func (a *app) allTop(c tele.Context) error { return a.top(c, "🏛 Зал славы", "") }

func (a *app) top(c tele.Context, header, since string) error {
	top, err := a.store.Top(c.Chat().ID, since)
	if err != nil {
		return err
	}
	if len(top) == 0 {
		return c.Send(EmptyTop)
	}
	medals := []string{"🥇", "🥈", "🥉"}
	var b strings.Builder
	b.WriteString(header + "\n" + Title + "\n")
	for i, s := range top {
		place := strconv.Itoa(i+1) + "."
		if i < len(medals) {
			place = medals[i]
		}
		b.WriteString("\n" + place + " " + html.EscapeString(s.Name) + " — " + strconv.Itoa(s.Count))
	}
	return c.Send(b.String())
}

func (a *app) mine(c tele.Context) error {
	u := c.Sender()
	year, err := a.store.Wins(c.Chat().ID, u.ID, time.Now().In(a.loc).Format("2006")+"-01-01")
	if err != nil {
		return err
	}
	all, err := a.store.Wins(c.Chat().ID, u.ID, "")
	if err != nil {
		return err
	}
	return c.Send(render(MyScore, "{name}", esc(u), "{year}", strconv.Itoa(year), "{all}", strconv.Itoa(all)))
}

func today(t time.Time, loc *time.Location) string { return t.In(loc).Format("2006-01-02") }

func pause() { time.Sleep(1500*time.Millisecond + rand.N(time.Second)) }

func displayName(u *tele.User) string {
	if name := strings.TrimSpace(u.FirstName + " " + u.LastName); name != "" {
		return name
	}
	if u.Username != "" {
		return u.Username
	}
	return strconv.FormatInt(u.ID, 10)
}

func esc(u *tele.User) string { return html.EscapeString(displayName(u)) }

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
