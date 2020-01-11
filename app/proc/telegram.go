package proc

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	tb "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pkg/errors"

	"github.com/umputun/feed-master/app/feed"
)

// TelegramClient client
type TelegramClient struct {
	Bot     *tb.BotAPI
	Timeout time.Duration
}

// NewTelegramClient init telegram client
func NewTelegramClient(token string, timeout time.Duration) (*TelegramClient, error) {
	if timeout == 0 {
		timeout = time.Duration(60 * 10)
	}

	if token == "" {
		return &TelegramClient{
			Bot:     nil,
			Timeout: timeout,
		}, nil
	}

	bot, err := tb.NewBotAPIWithClient(token, &http.Client{Timeout: timeout * time.Second})
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

	if err != nil {
		return errors.Wrapf(err, "can't send to telegram for %+v", item.Enclosure)
	}

	log.Printf("[DEBUG] telegram message sent: \n%s", message.Text)
	return nil
}

// getContentLength uses HEAD request and called as a fallback in case of item.Enclosure.Length not populated
func getContentLength(url string) (int, error) {
	resp, err := http.Head(url) //nolint:gosec
	if err != nil {
		return 0, errors.Wrapf(err, "can't HEAD %s", url)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, errors.Errorf("non-200 status, %d", resp.StatusCode)
	}

	log.Printf("[DEBUG] Content-Length: %d, url: %s", resp.ContentLength, url)
	return int(resp.ContentLength), err
}

func (client TelegramClient) sendAudio(channelID string, item feed.Item) (*tb.Message, error) {
	channel, _ := strconv.ParseInt(channelID, 10, 64)

	httpBody, err := client.downloadAudio(item.Enclosure.URL)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(httpBody)
	defer httpBody.Close() //nolint:staticcheck
	if err != nil {
		return nil, err
	}

	audioConfig := tb.NewAudioUpload(channel, tb.FileBytes{Bytes: data, Name: item.Title})
	audioConfig.Caption = client.getMessageHTML(item, false)
	audioConfig.Title = client.getFilenameByURL(item.Enclosure.URL)
	audioConfig.ParseMode = tb.ModeHTML

	message, err := client.Bot.Send(audioConfig)
	return &message, err
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
func (client TelegramClient) tagLinkOnlySupport(html string) string {
	p := bluemonday.NewPolicy()
	p.AllowAttrs("href").OnElements("a")
	return p.Sanitize(html)
}

func (client TelegramClient) getMessageHTML(item feed.Item, withMp3Link bool) string {
	title := strings.TrimSpace(item.Title)

	description := client.tagLinkOnlySupport(string(item.Description))
	description = strings.TrimSpace(description)

	messageHTML := fmt.Sprintf("<a href=\"%s\">%s</a>\n\n%s", item.Link, title, description)
	if withMp3Link {
		messageHTML = fmt.Sprintf("<a href=\"%s\">%s</a>\n\n%s\n\n%s", item.Link, title, description, item.Enclosure.URL)
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
