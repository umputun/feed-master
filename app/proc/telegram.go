package proc

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	if token == "" {
		return &TelegramClient{Bot: nil}, nil
	}

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
	if client.Bot == nil {
		return nil
	}

	if channelID == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response := make(chan *responseTelegram)
	go client.send(channelID, item, response)

	select {
	case <-ctx.Done():
		log.Printf("[WARN] timeout send telegram channel: [%s], title: [%s], url: [%s]", channelID, item.Title, item.Enclosure.URL)
		return nil
	case got := <-response:
		if got.err != nil {
			return got.err
		}

		log.Printf("[DEBUG] send telegram message: \n%s", got.message.Text)
		return got.err
	}
}

func (client TelegramClient) send(channelID string, item feed.Item, responseCh chan<- *responseTelegram) {
	message, err := client.Bot.Send(
		recipient{chatID: channelID},
		client.getMessageHTML(item),
		tb.ModeHTML,
		tb.NoPreview,
	)

	responseCh <- &responseTelegram{
		message: message,
		err:     err,
	}
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
	if !strings.HasPrefix(r.chatID, "@") {
		return "@" + r.chatID
	}

	return r.chatID
}

type responseTelegram struct {
	message *tb.Message
	err     error
}
