package channel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannel_Get(t *testing.T) {
	feedXML, err := os.ReadFile("testdata/channel.xml")
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("req: %v", r.URL.String())
		require.Equal(t, "/blah?channel_id=123", r.URL.String())
		_, e := w.Write(feedXML)
		require.NoError(t, e)
	}))

	c := Feed{Client: &http.Client{Timeout: time.Second}, BaseURL: ts.URL + "/blah?channel_id="}

	res, err := c.Get(context.Background(), "123")
	require.NoError(t, err)
	assert.Equal(t, 15, len(res))

	first := res[0]
	assert.Equal(t, "UCPU28A9z_ka_R5dQfecHJlA", first.ChannelID)
	assert.Equal(t, "Hou7PjJR498", first.VideoID)
	assert.Equal(t, "2022-03-20T12:00:07Z", first.Published.Format(time.RFC3339))
	assert.Equal(t, "https://www.youtube.com/watch?v=Hou7PjJR498", first.Link.Href)
	assert.Equal(t, "https://www.youtube.com/channel/UCPU28A9z_ka_R5dQfecHJlA", first.Author.URI)
	assert.Equal(t, "RTVI Новости", first.Author.Name)
	assert.Equal(t, `«Мировая война была неизбежна» / Что это было, Максим Шевченко`, first.Title)
	assert.Equal(t, "https://i1.ytimg.com/vi/Hou7PjJR498/hqdefault.jpg", first.Media.Thumbnail.URL)
	assert.Contains(t, first.Media.Description, " Что это было")

	last := res[14]
	assert.Equal(t, "UCPU28A9z_ka_R5dQfecHJlA", last.ChannelID)
	assert.Equal(t, "zBwM0SU1vRk", last.VideoID)
	assert.Equal(t, "2022-03-16T08:00:29Z", last.Published.Format(time.RFC3339))
	assert.Equal(t, "https://www.youtube.com/watch?v=zBwM0SU1vRk", last.Link.Href)
	assert.Equal(t, "https://www.youtube.com/channel/UCPU28A9z_ka_R5dQfecHJlA", last.Author.URI)
	assert.Equal(t, "RTVI Новости", last.Author.Name)
	assert.Equal(t, `«Она показала пример». Константин Калачев — об антивоенной акции Овсянниковой в эфире Первого канала`, last.Title)
	assert.Equal(t, "https://i3.ytimg.com/vi/zBwM0SU1vRk/hqdefault.jpg", last.Media.Thumbnail.URL)
	assert.Contains(t, last.Media.Description, "за призыв к публичным несанкционированным акциям протеста")
}
