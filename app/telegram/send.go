// Package telegram connecting Telegram API
package telegram

import (
	"os"

	log "github.com/go-pkgz/lgr"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/umputun/feed-master/app/feed"
)

var (
	telegramToken  = os.Getenv("TELEGRAM_TOKEN")
	telegramChatID = os.Getenv("TELEGRAM_CHAT_ID")
)

// Send message
func Send(item feed.Item) {
	bot, err := tb.NewBot(tb.Settings{
		Token: telegramToken,
	})

	if err != nil {
		log.Printf("[WARN] failed initilization telegram bot %s, %v", telegramChatID, err)
		return
	}

	_, err = bot.Send(
		recipient{chatID: telegramChatID},
		item.Enclosure.URL,
	)
	if err != nil {
		log.Printf("[WARN] failed send telegram message")
		return
	}
	log.Printf("[DEBUG] send telegram message")
}

type recipient struct {
	chatID string
}

func (r recipient) Recipient() string {
	return r.chatID
}
