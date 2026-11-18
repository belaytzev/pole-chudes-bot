package main

import (
	"html"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	_ "time/tzdata"

	tele "gopkg.in/telebot.v3"
)

type app struct {
	bot     *tele.Bot
	store   *Store
	loc     *time.Location
	sticker string // optional file_id of the spinning-drum sticker
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
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN is not set")
	}
	bot, err := tele.NewBot(tele.Settings{
		Token:     token,
		Poller:    &tele.LongPoller{Timeout: 30 * time.Second},
		ParseMode: tele.ModeHTML,
		OnError:   func(err error, c tele.Context) { log.Printf("update error: %v", err) },
	})
	if err != nil {
		log.Fatal(err)
	}
	a := &app{bot, store, loc, os.Getenv("SPIN_STICKER")}

	var inflight sync.WaitGroup
	bot.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			inflight.Add(1)
			defer inflight.Done()
			return next(c)
		}
	})
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
	g.Handle("/dr", a.birthday)
	g.Handle("/citata", a.quote)
	g.Handle(tele.OnUserLeft, a.userLeft)
	g.Handle(tele.OnMigration, a.migrate)

	err = bot.SetCommands([]tele.Command{
		{Text: "igrok", Description: "Записаться в игроки"},
		{Text: "vyhod", Description: "Выйти из игры"},
		{Text: "baraban", Description: "Вращать барабан"},
		{Text: "kapital", Description: "Итоги года"},
		{Text: "zalslavy", Description: "Зал славы"},
		{Text: "moi", Description: "Мой счёт"},
		{Text: "dr", Description: "День рождения"},
		{Text: "citata", Description: "Цитата: сохранить или показать"},
	})
	if err != nil {
		log.Printf("set commands: %v", err)
	}
	log.Printf("started as @%s", bot.Me.Username)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-stop; bot.Stop() }()
	go a.greetLoop()
	bot.Start()
	inflight.Wait()
	if err := store.Close(); err != nil {
		log.Printf("close store: %v", err)
	}
}

func (a *app) migrate(c tele.Context) error {
	from, to := c.Migration()
	return a.store.Migrate(from, to)
}

func groupOnly(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if c.Chat().Type == tele.ChatPrivate {
			return c.Send(pick(PrivateOnly))
		}
		if u := c.Sender(); u != nil && u.IsBot { // anonymous group admins arrive as GroupAnonymousBot
			return nil
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
	go a.syncBirthday(u.ID)
	if fresh {
		return c.Send(render(pick(Joined), "{name}", esc(u)))
	}
	return c.Send(render(pick(AlreadyJoined), "{name}", esc(u)))
}

func (a *app) leave(c tele.Context) error {
	u := c.Sender()
	left, err := a.store.Leave(c.Chat().ID, u.ID)
	if err != nil {
		return err
	}
	if left {
		return c.Send(render(pick(LeftGame), "{name}", esc(u)))
	}
	return c.Send(render(pick(NotJoined), "{name}", esc(u)))
}

func (a *app) userLeft(c tele.Context) error {
	u := c.Message().UserLeft
	left, err := a.store.Leave(c.Chat().ID, u.ID)
	if err != nil || !left {
		return err
	}
	return c.Send(render(pick(LeftChat), "{name}", esc(u)))
}

func (a *app) spin(c tele.Context) error {
	chat := c.Chat()
	players, err := a.store.Players(chat.ID)
	if err != nil {
		return err
	}
	now := time.Now().In(a.loc)
	excl := a.birthdayExcl(chat.ID, now)
	day := today(now, a.loc)
	user, name, fresh, ok, err := a.store.DrawExcept(chat.ID, day, excl)
	if err != nil {
		return err
	}
	if !ok {
		return c.Send(pick(NotEnough))
	}
	bUser, bName, bOk, err := a.store.Beauty(chat.ID, day, user)
	if err != nil {
		return err
	}
	bName = html.EscapeString(bName)
	name = html.EscapeString(name)
	if !fresh {
		msg := render(pick(Already), "{name}", name)
		if bOk {
			msg += "\n" + render(pick(AlreadyBeauties), "{name}", bName)
		}
		return c.Send(msg)
	}
	days, err := a.store.WinDays(chat.ID, user)
	if err != nil {
		return err
	}
	series := runs(days)
	if len(series) == 0 {
		series = []int{1}
	}
	wins, streak := len(days), series[len(series)-1]
	before := slices.Clone(series)
	before[len(before)-1]--
	a.react(chat, c.Message(), "👀")

	mention := `<a href="tg://user?id=` + strconv.FormatInt(user, 10) + `">` + name + `</a>`
	final := "П🟦🟦🟦Р дня\n\n" + a.deal(chat.ID, "reveals", Reveals) + "\n\n" + render(a.deal(chat.ID, "winners", Winners), "{name}", mention)
	if j, ok := Jubilees[wins]; ok {
		final += "\n\n" + render(pick(j), "{name}", name)
	} else if streak >= 2 {
		final += "\n\n" + render(pick(Streaks), "{name}", name, "{n}", strconv.Itoa(streak))
	}
	final += awardLines(name, stats{Wins: wins - 1, BestStreak: best(before)}, stats{Wins: wins, BestStreak: best(series)})
	final += "\n\n" + a.deal(chat.ID, "prizes", Prizes)
	if bOk {
		bMention := `<a href="tg://user?id=` + strconv.FormatInt(bUser, 10) + `">` + bName + `</a>`
		final += "\n\n" + render(a.deal(chat.ID, "beauties", Beauties), "{name}", bMention)
		bWins, err := a.store.BeautyWins(chat.ID, bUser, "")
		if err != nil {
			return err
		}
		final += awardLines(bName, stats{Beauty: bWins - 1}, stats{Beauty: bWins})
	}
	if len(excl) > 0 && !slices.Contains(excl, user) {
		final += "\n\n" + pick(BirthdayRest)
	}
	frames := []string{
		render(pick(Openings), "{n}", strconv.Itoa(players)),
		pick(Spins),
		pick(Sectors),
		pick(Gifts),
		pick(Letters),
	}

	if a.sticker != "" {
		sm, err := a.bot.Send(chat, &tele.Sticker{File: tele.File{FileID: a.sticker}})
		if err != nil {
			log.Printf("send sticker: %v", err)
		} else {
			defer func() {
				if err := a.bot.Delete(sm); err != nil {
					log.Printf("delete sticker: %v", err)
				}
			}()
		}
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
	a.react(chat, msg, pick(WinReactions))
	return nil
}

func (a *app) deal(chat int64, name string, pool []string) string {
	p, err := a.store.Deal(chat, name, pool)
	if err != nil {
		log.Printf("deal %s: %v", name, err)
		return pick(pool)
	}
	return p
}

func awardLines(name string, before, after stats) string {
	var out string
	for _, aw := range newAwards(before, after) {
		out += "\n\n" + render(pick(AwardNew), "{name}", name, "{award}", aw)
	}
	return out
}

// Chats may restrict the allowed reactions, so a failure is only logged.
func (a *app) react(chat *tele.Chat, msg tele.Editable, emoji string) {
	err := a.bot.React(chat, msg, tele.ReactionOptions{Reactions: []tele.Reaction{{Type: "emoji", Emoji: emoji}}, Big: true})
	if err != nil {
		log.Printf("react: %v", err)
	}
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
	beauty, err := a.store.BeautyTop(c.Chat().ID, since)
	if err != nil {
		return err
	}
	if len(top) == 0 && len(beauty) == 0 {
		return c.Send(pick(EmptyTop))
	}
	return c.Send(topText(header, top, beauty))
}

func topText(header string, top, beauty []Score) string {
	var b strings.Builder
	b.WriteString(header)
	for _, sec := range []struct {
		title string
		top   []Score
	}{{Title, top}, {BeautyTitle, beauty}} {
		if len(sec.top) == 0 {
			continue
		}
		b.WriteString("\n\n" + sec.title + "\n")
		writeTop(&b, sec.top)
	}
	return b.String()
}

func yearEndText(year int, top, beauty []Score) string {
	text := topText(render(pick(YearEnd), "{year}", strconv.Itoa(year)), top, beauty)
	if len(top) == 0 {
		return text
	}
	if len(top) > 1 && top[0].Count == top[1].Count {
		return text + "\n\n" + pick(YearTie)
	}
	return text + "\n\n" + render(pick(YearChampion), "{name}", html.EscapeString(top[0].Name), "{count}", strconv.Itoa(top[0].Count))
}

func writeTop(b *strings.Builder, top []Score) {
	medals := []string{"🥇", "🥈", "🥉"}
	for i, s := range top {
		place := strconv.Itoa(i+1) + "."
		if i < len(medals) {
			place = medals[i]
		}
		b.WriteString("\n" + place + " " + html.EscapeString(s.Name) + " — " + strconv.Itoa(s.Count))
	}
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
	since := time.Now().In(a.loc).Format("2006") + "-01-01"
	byear, err := a.store.BeautyWins(c.Chat().ID, u.ID, since)
	if err != nil {
		return err
	}
	ball, err := a.store.BeautyWins(c.Chat().ID, u.ID, "")
	if err != nil {
		return err
	}
	days, err := a.store.WinDays(c.Chat().ID, u.ID)
	if err != nil {
		return err
	}
	awards := pick(NoAwards)
	if got := earned(stats{Wins: all, BestStreak: best(runs(days)), Beauty: ball}); len(got) > 0 {
		awards = "🏅 Ачивки: " + strings.Join(got, ", ")
	}
	return c.Send(render(pick(MyScore), "{name}", esc(u), "{year}", strconv.Itoa(year), "{all}", strconv.Itoa(all),
		"{byear}", strconv.Itoa(byear), "{ball}", strconv.Itoa(ball)) + "\n\n" + awards)
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
