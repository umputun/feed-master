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
	"golang.org/x/net/html"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/umputun/feed-master/app/duration"

	"github.com/umputun/feed-master/app/feed"
)

// TelegramClient client
type TelegramClient struct {
	Bot             *tb.Bot
	Timeout         time.Duration
	DurationService *duration.Service
}

// NewTelegramClient init telegram client
func NewTelegramClient(token, apiURL string, timeout time.Duration, durSvc *duration.Service) (*TelegramClient, error) {
	log.Printf("[INFO] create telegram client for %s, timeout: %s", apiURL, timeout)
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
		Bot:             bot,
		Timeout:         timeout,
		DurationService: durSvc,
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
		client.getMessageHTML(item, htmlMessageParams{WithMp3Link: true}),
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
		File:      tb.FromReader(&httpBodyCopy),
		FileName:  item.GetFilename(),
		MIME:      "audio/mpeg",
		Caption:   client.getMessageHTML(item, htmlMessageParams{TrimCaption: true}),
		Title:     item.Title,
		Performer: item.Author,
		Duration:  client.DurationService.Reader(tee),
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

type htmlMessageParams struct{ WithMp3Link, TrimCaption bool }

// getMessageHTML generates HTML message from provided feed.Item
func (client TelegramClient) getMessageHTML(item feed.Item, params htmlMessageParams) string {
	var header, footer string
	title := strings.TrimSpace(item.Title)
	if title != "" && item.Link == "" {
		header = fmt.Sprintf("%s\n\n", title)
	} else if title != "" && item.Link != "" {
		header = fmt.Sprintf("<a href=%q>%s</a>\n\n", item.Link, title)
	}

	if params.WithMp3Link {
		footer += fmt.Sprintf("\n\n%s", item.Enclosure.URL)
	}

	description := string(item.Description)
	description = strings.TrimPrefix(description, "<![CDATA[")
	description = strings.TrimSuffix(description, "]]>")
	// apparently bluemonday doesn't remove escaped HTML tags
	description = client.tagLinkOnlySupport(html.UnescapeString(description))
	description = strings.TrimSpace(description)

	// https://limits.tginfo.me/en 1024 symbol limit for caption
	if params.TrimCaption && len(header+description+footer) > 1024 {
		description = CropText(description, 1024-len(header+footer))
	}

	return header + description + footer
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

// CropText shrinks the provided string, removing HTML tags in case it's exceeding the limit
func CropText(inp string, max int) string {
	if len([]rune(inp)) > max {
		return CleanText(inp, max)
	}
	return inp
}
