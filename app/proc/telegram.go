package proc

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/umputun/feed-master/app/feed"
)

// TelegramClient client
type TelegramClient struct {
	Bot     *tb.Bot
	Timeout time.Duration
}

// NewTelegramClient init telegram client
func NewTelegramClient(token, apiURL string, timeout time.Duration) (*TelegramClient, error) {
	if timeout == 0 {
		timeout = time.Second * 60
	}

	if token == "" {
		return &TelegramClient{
			Bot:     nil,
			Timeout: timeout,
		}, nil
	}

	bot, err := tb.NewBot(tb.Settings{
		URL:    apiURL,
		Token:  token,
		Client: &http.Client{Timeout: timeout * time.Second},
	})
	if err != nil {
		return nil, err
	}

	result := TelegramClient{
		Bot:     bot,
		Timeout: timeout,
	}
	return &result, err
}

// Send message, skip if telegram token empty
func (client TelegramClient) Send(channelID string, item feed.Item) (err error) {
	if client.Bot == nil || channelID == "" {
		return nil
	}

	message, err := client.sendAudio(channelID, item)
	if err != nil && strings.HasSuffix(err.Error(), "Request Entity Too Large") {
		message, err = client.sendText(channelID, item)
	}

	if err != nil {
		return errors.Wrapf(err, "can't send to telegram for %+v", item.Enclosure)
	}

	log.Printf("[DEBUG] telegram message sent: \n%s", message.Text)
	return nil
}

func (client TelegramClient) sendText(channelID string, item feed.Item) (*tb.Message, error) {
	message, err := client.Bot.Send(
		recipient{chatID: channelID},
		client.getMessageHTML(item, true),
		tb.ModeHTML,
		tb.NoPreview,
	)

	return message, err
}

func (client TelegramClient) sendAudio(channelID string, item feed.Item) (*tb.Message, error) {
	httpBody, err := client.downloadAudio(item.Enclosure.URL)
	if err != nil {
		return nil, err
	}
	defer httpBody.Close()

	audio := tb.Audio{
		File:     tb.FromReader(httpBody),
		FileName: client.getFilenameByURL(item.Enclosure.URL),
		MIME:     "audio/mpeg",
		Caption:  client.getMessageHTML(item, false),
		Title:    item.Title,
	}

	message, err := audio.Send(
		client.Bot,
		recipient{chatID: channelID},
		&tb.SendOptions{
			ParseMode: tb.ModeHTML,
		},
	)

	return message, err
}

func (client TelegramClient) downloadAudio(url string) (io.ReadCloser, error) {
	clientHTTP := &http.Client{Timeout: client.Timeout * time.Second}

	resp, err := clientHTTP.Get(url)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] start download audio: %s", url)

	return resp.Body, err
}

// https://core.telegram.org/bots/api#html-style
func (client TelegramClient) tagLinkOnlySupport(htmlText string) string {
	p := bluemonday.NewPolicy()
	p.AllowAttrs("href").OnElements("a")
	return html.UnescapeString(p.Sanitize(htmlText))
}

// getMessageHTML generates HTML message from provided feed.Item
func (client TelegramClient) getMessageHTML(item feed.Item, withMp3Link bool) string {
	description := string(item.Description)

	description = strings.TrimPrefix(description, "<![CDATA[")
	description = strings.TrimSuffix(description, "]]>")

	// apparently bluemonday doesn't remove escaped HTML tags
	description = client.tagLinkOnlySupport(html.UnescapeString(description))
	description = strings.TrimSpace(description)

	messageHTML := description

	title := strings.TrimSpace(item.Title)
	if title != "" {
		switch {
		case item.Link == "":
			messageHTML = fmt.Sprintf("%s\n\n", title) + messageHTML
		case item.Link != "":
			messageHTML = fmt.Sprintf("<a href=\"%s\">%s</a>\n\n", item.Link, title) + messageHTML
		}
	}

	if withMp3Link {
		messageHTML += fmt.Sprintf("\n\n%s", item.Enclosure.URL)
	}

	return messageHTML
}

func (client TelegramClient) getFilenameByURL(url string) string {
	_, filename := path.Split(url)
	return filename
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
