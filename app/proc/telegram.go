package proc

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pkg/errors"
	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/umputun/feed-master/app/feed"
)

const maxTelegramFileSize = 50_000_000

// TelegramClient client
type TelegramClient struct {
	Bot            *tb.Bot
	Timeout        time.Duration
	uploaderConfig *TelegramUploaderConfig
}

// TelegramUploaderConfig struct used to configure
// experimental uploader to Telegram for large audio files
type TelegramUploaderConfig struct {
	Enabled      bool   `long:"uploader_enabled" env:"UPLOADER_ENABLED" description:"Enables experimental Telegram large files uploader"`
	PathToScript string `long:"path" env:"PATH_TO_SCRIPT" description:"Path to Python uploader script"`
	APIID        string `long:"api_id" env:"API_ID" description:"Telegram APP API ID in format like 0000000"`
	APIHash      string `long:"api_hash" env:"API_HASH" description:"Telegram APP API Hash in format like 0123456789acbdefghijklmnopqrstuw"`
	Session      string `long:"session" env:"SESSION" description:"Path to Telegram client session file (created by uploader/auth.py script)"`
}

// NewTelegramClient init telegram client
func NewTelegramClient(token string, timeout time.Duration, uploaderConfig *TelegramUploaderConfig) (*TelegramClient, error) {
	if timeout == 0 {
		timeout = time.Duration(60 * 10)
	}

	if token == "" {
		return &TelegramClient{
			Bot:     nil,
			Timeout: timeout,
		}, nil
	}

	bot, err := tb.NewBot(tb.Settings{
		Token:  token,
		Client: &http.Client{Timeout: timeout * time.Second},
	})
	if err != nil {
		return nil, err
	}

	result := TelegramClient{
		Bot:            bot,
		Timeout:        timeout,
		uploaderConfig: uploaderConfig,
	}
	return &result, err
}

// Send message, skip if telegram token empty
func (client TelegramClient) Send(channelID string, item feed.Item) (err error) {

	if client.Bot == nil || channelID == "" {
		return nil
	}

	contentLength := item.Enclosure.Length
	if contentLength <= 0 {
		if contentLength, err = getContentLength(item.Enclosure.URL); err != nil {
			return errors.Wrapf(err, "can't get length for %s", item.Enclosure.URL)
		}
	}

	message, err := client.send(channelID, item, contentLength)
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

func (client TelegramClient) send(channelID string, item feed.Item, contentLength int) (*tb.Message, error) {
	if contentLength < maxTelegramFileSize {
		return client.sendAudio(channelID, item)
	}

	if client.uploaderConfig.Enabled {
		return client.sendAudioWithExternalUploader(channelID, item)
	}

	return client.sendText(channelID, item)
}

func (client TelegramClient) sendText(channelID string, item feed.Item) (*tb.Message, error) {
	return client.Bot.Send(
		recipient{chatID: channelID},
		client.getMessageHTML(item, true),
		tb.ModeHTML,
		tb.NoPreview,
	)
}

func (client TelegramClient) sendAudio(channelID string, item feed.Item) (*tb.Message, error) {
	httpBody, err := client.downloadAudio(item.Enclosure.URL)
	defer httpBody.Close() //nolint:staticcheck
	if err != nil {
		return nil, err
	}

	audio := tb.Audio{
		File:     tb.FromReader(httpBody),
		FileName: client.getFilenameByURL(item.Enclosure.URL),
		MIME:     "audio/mpeg",
		Caption:  client.getMessageHTML(item, false),
		Title:    item.Title,
	}

	return audio.Send(
		client.Bot,
		recipient{chatID: channelID},
		&tb.SendOptions{
			ParseMode: tb.ModeHTML,
		},
	)
}

func (client TelegramClient) sendAudioWithExternalUploader(channelID string, item feed.Item) (*tb.Message, error) {
	httpBody, err := client.downloadAudio(item.Enclosure.URL)
	defer httpBody.Close() //nolint:staticcheck
	if err != nil {
		return nil, err
	}

	tmpFile, err := ioutil.TempFile(os.TempDir(), "feed-master-")
	if err != nil {
		return nil, errors.Wrap(err, "create temoprary file")
	}
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, httpBody)
	if err != nil {
		return nil, errors.Wrap(err, "write to temporary file")
	}

	output, err := execute(
		"python3 "+client.uploaderConfig.PathToScript,
		map[string]string{
			"API_ID":            client.uploaderConfig.APIID,
			"API_HASH":          client.uploaderConfig.APIHash,
			"SESSION":           client.uploaderConfig.Session,
			"SEND_TO":           channelID,
			"FILE_PATH":         tmpFile.Name(),
			"CAPTION":           client.getMessageHTML(item, false),
			"PARSE_MODE":        "html",
			"SHOW_PROGRESS_BAR": "false",
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "execute experimental audio uploader")
	}

	return &tb.Message{
		Text: output,
	}, nil
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

func execute(command string, env map[string]string) (string, error) {
	args := strings.Split(command, " ")

	var stdout, stderr bytes.Buffer

	cmd := exec.Command(args[0], args[1:]...) // #nosec
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmdEnv := []string{}
	for key, value := range env {
		cmdEnv = append(cmdEnv, key+"="+value)
	}
	cmd.Env = cmdEnv

	err := cmd.Run()
	if err != nil {
		return "", errors.Wrapf(err, command)
	}

	if stderr.Len() > 0 {
		return "", errors.New(stderr.String())
	}
	return stdout.String(), nil
}
