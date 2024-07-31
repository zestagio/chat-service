package websocketstream

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	gorillaws "github.com/gorilla/websocket"
)

type Websocket interface {
	SetWriteDeadline(t time.Time) error
	NextWriter(messageType int) (io.WriteCloser, error)
	WriteMessage(messageType int, data []byte) error
	WriteControl(messageType int, data []byte, deadline time.Time) error

	SetPongHandler(h func(appData string) error)
	SetReadDeadline(t time.Time) error
	NextReader() (messageType int, r io.Reader, err error)

	Close() error
}

type Upgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (Websocket, error)
}

type upgraderImpl struct {
	upgrader *gorillaws.Upgrader
}

func NewUpgrader(allowOrigins []string, secWsProtocol string) Upgrader {
	upgrader := &gorillaws.Upgrader{
		HandshakeTimeout: 1 * time.Second,           // Slow clients should be oppressed.
		ReadBufferSize:   125,                       // Max control frame payload size.
		WriteBufferSize:  1,                         // To save memory on each connection, because app doesn't frame message.
		CheckOrigin:      checkOrigin(allowOrigins), // Simple check origin func.
		Subprotocols:     []string{secWsProtocol},
	}
	return &upgraderImpl{
		upgrader: upgrader,
	}
}

func (u *upgraderImpl) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (Websocket, error) {
	return u.upgrader.Upgrade(w, r, responseHeader)
}

// Based on echomdlwr.CORSWithConfig used in server/server.go.
func checkOrigin(allowOrigins []string) func(*http.Request) bool {
	allowOriginPatterns := make([]string, 0, len(allowOrigins))
	for _, origin := range allowOrigins {
		pattern := regexp.QuoteMeta(origin)
		pattern = strings.ReplaceAll(pattern, "\\*", ".*")
		pattern = strings.ReplaceAll(pattern, "\\?", ".")
		pattern = "^" + pattern + "$"
		allowOriginPatterns = append(allowOriginPatterns, pattern)
	}

	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false
		}

		for _, o := range allowOrigins { // Small O(N).
			if o == origin {
				return true
			}
		}

		// To avoid regex cost by invalid (long) domains (253 is domain name max limit).
		if len(origin) > (253+3+5) || !strings.Contains(origin, "://") {
			return false
		}

		for _, re := range allowOriginPatterns { // Small O(N).
			if match, _ := regexp.MatchString(re, origin); match {
				return true
			}
		}

		return false
	}
}
