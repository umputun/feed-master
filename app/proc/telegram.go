package proc

import (
	log "github.com/go-pkgz/lgr"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/umputun/feed-master/app/feed"
)

// TelegramClient client
type TelegramClient struct {
	Bot *tb.Bot
}

// NewTelegramClient init telegram client
func NewTelegramClient(token string) (*TelegramClient, error) {
	bot, err := tb.NewBot(tb.Settings{
		Token: token,
	})
	if err != nil {
		return nil, err
	}

	result := TelegramClient{
		Bot: bot,
	}
	return &result, err
}

// Send message, skip if telegram token empty
func (client TelegramClient) Send(channelID string, item feed.Item) error {
	if channelID == "" {
		return nil
	}

	_, err := client.Bot.Send(
		recipient{chatID: channelID},
		item.Enclosure.URL,
	)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] send telegram message")
	return err
}

type recipient struct {
	chatID string
}

func (r recipient) Recipient() string {
	return r.chatID
}
