package proc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/microcosm-cc/bluemonday"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/umputun/feed-master/app/feed"
)

const (
	maxTelegramFileSize = 50_000_000
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
		Token:  token,
		Client: &http.Client{Timeout: 60 * 10 * time.Second},
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

	contentLength, err := getContentLength(item.Enclosure.URL)
	if err != nil {
		return err
	}

	var message *tb.Message

	if contentLength < maxTelegramFileSize {
		message, err = client.sendAudio(channelID, item)
	} else {
		message, err = client.sendText(channelID, item)
	}

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] send telegram message: \n%s", message.Text)
	return err
}

func getContentLength(url string) (int64, error) {
	resp, err := http.Head(url) //nolint:gosec
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("resp.StatusCode: %d, not equal 200", resp.StatusCode)
	}

	log.Printf("[DEBUG] Content-Length: %d, url: %s", resp.ContentLength, url)
	return resp.ContentLength, err
}

func (client TelegramClient) sendText(channelID string, item feed.Item) (*tb.Message, error) {
	message, err := client.Bot.Send(
		recipient{chatID: channelID},
		client.getMessageHTML(item),
		tb.ModeHTML,
		tb.NoPreview,
	)

	return message, err
}

func (client TelegramClient) sendAudio(channelID string, item feed.Item) (*tb.Message, error) {
	file, err := client.downloadAudio(item.Enclosure.URL)
	if err != nil {
		return nil, err
	}

	audio := tb.Audio{
		File:     tb.FromReader(bytes.NewReader(*file)),
		FileName: "1.mp3",
		MIME:     "audio/mpeg",
		Caption:  client.getMessageHTML(item),
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

func (client TelegramClient) downloadAudio(url string) (*[]byte, error) {
	clientHTTP := &http.Client{Timeout: 60 * 10 * time.Second}

	resp, err := clientHTTP.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("[DEBUG] start download audio: %s", url)

	file, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] finish download audio: %s", url)
	return &file, err
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
