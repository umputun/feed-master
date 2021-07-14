package proc

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pkg/errors"
	"github.com/tcolgate/mp3"
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
		Client: &http.Client{Timeout: timeout},
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
	httpBody, err := item.DownloadAudio(client.Timeout)
	if err != nil {
		return nil, err
	}
	defer httpBody.Close()

	var httpBodyCopy bytes.Buffer
	tee := io.TeeReader(httpBody, &httpBodyCopy)

	audio := tb.Audio{
		File:     tb.FromReader(&httpBodyCopy),
		FileName: item.GetFilename(),
		MIME:     "audio/mpeg",
		Caption:  client.getMessageHTML(item, false),
		Title:    item.Title,
		Duration: client.duration(tee),
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

// duration scans MP3 file from provided io.Reader and returns it's duration in seconds, ignoring possible errors
func (client TelegramClient) duration(r io.Reader) int {
	d := mp3.NewDecoder(r)
	var f mp3.Frame
	var skipped int
	var duration float64
	var err error

	for err == nil {
		if err = d.Decode(&f, &skipped); err != nil && err != io.EOF {
			return 0
		}
		duration += f.Duration().Seconds()
	}
	return int(duration)
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
