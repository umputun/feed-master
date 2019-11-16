package proc

import (
	"fmt"
	"strings"

	log "github.com/go-pkgz/lgr"
	"github.com/microcosm-cc/bluemonday"
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

	message, err := client.Bot.Send(
		recipient{chatID: channelID},
		client.getMessageHTML(item),
		tb.ModeHTML,
		tb.NoPreview,
	)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] send telegram message: \n%s", message.Text)
	return err
}

// https://core.telegram.org/bots/api#html-style
func (client TelegramClient) tagLinkOnlySupport(html string) string {
	p := bluemonday.NewPolicy()
	p.AllowAttrs("href").OnElements("a")
	return p.Sanitize(html)
}

func (client TelegramClient) getMessageHTML(item feed.Item) string {
	title := strings.TrimSpace(item.Title)

	description := client.tagLinkOnlySupport(string(item.Description))
	description = strings.TrimSpace(description)

	messageHTML := fmt.Sprintf("%s\n\n%s\n\n%s", title, description, item.Enclosure.URL)

	return messageHTML
}

type recipient struct {
	chatID string
}

func (r recipient) Recipient() string {
	return r.chatID
}
