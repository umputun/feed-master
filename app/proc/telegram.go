package proc

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/umputun/feed-master/app/feed"
)

//go:generate moq -out mocks/tg_sender.go -pkg mocks -skip-ensure -fmt goimports . TelegramSender
//go:generate moq -out mocks/dutation.go -pkg mocks -skip-ensure -fmt goimports . DurationService

// TelegramClient client
type TelegramClient struct {
	Bot             *tb.Bot
	Timeout         time.Duration
	DurationService DurationService
	TelegramSender  TelegramSender
}

// TelegramSender is the interface for sending messages to telegram
type TelegramSender interface {
	Send(tb.Audio, *tb.Bot, tb.Recipient, *tb.SendOptions) (*tb.Message, error)
}

// DurationService is the interface for reading duration from files
type DurationService interface {
	File(fname string) int
}

// NewTelegramClient init telegram client
func NewTelegramClient(token, apiURL string, timeout time.Duration, ds DurationService, tgs TelegramSender) (*TelegramClient, error) {
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
		DurationService: ds,
		TelegramSender:  tgs,
	}
	return &result, err
}

// Send message, skip if telegram token empty
func (client TelegramClient) Send(channelID string, item feed.Item) (err error) {
	if client.Bot == nil || channelID == "" {
		return nil
	}

	message, err := client.sendAudio(channelID, item)
	if err != nil && strings.Contains(err.Error(), "Request Entity Too Large") {
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
	// download audio to the temp file
	tmpFile, err := ioutil.TempFile(os.TempDir(), "feed-master-*.mp3")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	if _, err = io.Copy(tmpFile, httpBody); err != nil {
		return nil, err
	}
	if closeErr := tmpFile.Close(); closeErr != nil {
		return nil, closeErr
	}

	var dur int
	if item.Duration != "" { // item may have duration published, if not, try to get it from mp3 file
		if dur, err = strconv.Atoi(item.Duration); err != nil {
			dur = client.DurationService.File(tmpFile.Name())
		}
	} else {
		dur = client.DurationService.File(tmpFile.Name())
	}

	audio := tb.Audio{
		File:      tb.FromDisk(tmpFile.Name()),
		FileName:  item.GetFilename(),
		MIME:      "audio/mpeg",
		Caption:   client.getMessageHTML(item, htmlMessageParams{TrimCaption: true}),
		Title:     item.Title,
		Performer: item.Author,
		Duration:  dur,
	}

	return client.TelegramSender.Send(audio, client.Bot, recipient{chatID: channelID}, &tb.SendOptions{ParseMode: tb.ModeHTML})
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
	if _, err := strconv.ParseInt(r.chatID, 10, 64); err != nil && !strings.HasPrefix(r.chatID, "@") {
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

// TelegramSenderImpl is a TelegramSender implementation that sends messages to Telegram for real
type TelegramSenderImpl struct{}

// Send sends a message to Telegram
func (tg *TelegramSenderImpl) Send(audio tb.Audio, bot *tb.Bot, rcp tb.Recipient, opts *tb.SendOptions) (*tb.Message, error) {
	return audio.Send(bot, rcp, opts)
}
