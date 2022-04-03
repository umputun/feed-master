package api

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_Run(t *testing.T) {
	s := Server{Version: "1.0", TemplLocation: "../webapp/templates/*"}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	port := rand.Intn(10000) + 4000
	go func() {
		time.Sleep(time.Millisecond * 100)
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/ping", port))
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		t.Logf("%+v", resp.Header)
		assert.Equal(t, "1.0", resp.Header.Get("App-Version"))
		assert.Equal(t, "feed-master", resp.Header.Get("App-Name"))
	}()
	s.Run(ctx, port)
}
