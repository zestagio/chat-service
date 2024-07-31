package websocketstream

import (
	"context"
	"errors"
	"fmt"
	"time"

	gorillaws "github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/zestagio/chat-service/internal/middlewares"
	eventstream "github.com/zestagio/chat-service/internal/services/event-stream"
	"github.com/zestagio/chat-service/internal/types"
)

const (
	writeTimeout = time.Second
)

type eventStream interface {
	Subscribe(ctx context.Context, userID types.UserID) (<-chan eventstream.Event, error)
}

//go:generate options-gen -out-filename=handler_options.gen.go -from-struct=Options
type Options struct {
	pingPeriod time.Duration `default:"3s" validate:"omitempty,min=100ms,max=30s"`

	logger       *zap.Logger     `option:"mandatory" validate:"required"`
	eventStream  eventStream     `option:"mandatory" validate:"required"`
	eventAdapter EventAdapter    `option:"mandatory" validate:"required"`
	eventWriter  EventWriter     `option:"mandatory" validate:"required"`
	upgrader     Upgrader        `option:"mandatory" validate:"required"`
	shutdownCh   <-chan struct{} `option:"mandatory" validate:"required"`
}

type HTTPHandler struct {
	Options
	pingPeriod time.Duration
	pongWait   time.Duration
}

func NewHTTPHandler(opts Options) (*HTTPHandler, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}
	opts.logger = opts.logger.Named("websocket")

	return &HTTPHandler{
		Options:    opts,
		pingPeriod: opts.pingPeriod,
		pongWait:   pongWait(opts.pingPeriod),
	}, nil
}

func (h *HTTPHandler) Serve(eCtx echo.Context) error {
	ws, err := h.upgrader.Upgrade(eCtx.Response(), eCtx.Request(), nil)
	if err != nil {
		return fmt.Errorf("upgrade request: %v", err)
	}

	ctx, cancel := context.WithCancel(eCtx.Request().Context())
	defer cancel()

	wsCloser := newWsCloser(h.logger, ws)
	uid := middlewares.MustUserID(eCtx)

	events, err := h.eventStream.Subscribe(ctx, uid)
	if err != nil {
		h.logger.Error("cannot subscribe for events", zap.Error(err))
		wsCloser.Close(gorillaws.CloseInternalServerErr)
		return nil
	}

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return h.writeLoop(ctx, ws, events)
	})

	eg.Go(func() error {
		return h.readLoop(ctx, ws)
	})

	eg.Go(func() error {
		select {
		case <-ctx.Done():
		case <-h.shutdownCh:
			wsCloser.Close(gorillaws.CloseNormalClosure)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		if !errors.Is(err, gorillaws.ErrCloseSent) {
			h.logger.Error("unexpected error", zap.Error(err))
			wsCloser.Close(gorillaws.CloseInternalServerErr)
		}
		return nil
	}

	wsCloser.Close(gorillaws.CloseNormalClosure)
	return nil
}

// readLoop listen PONGs.
func (h *HTTPHandler) readLoop(_ context.Context, ws Websocket) error {
	ws.SetPongHandler(func(string) error {
		h.logger.Debug("pong")
		return ws.SetReadDeadline(time.Now().Add(h.pongWait))
	})

	if err := ws.SetReadDeadline(time.Now().Add(h.pongWait)); err != nil {
		return fmt.Errorf("set first read deadline: %v", err)
	}
	for {
		_, _, err := ws.NextReader()
		if gorillaws.IsCloseError(err, gorillaws.CloseNormalClosure) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("get next reader: %w", err)
		}
	}
}

// writeLoop listen events and writes them into Websocket.
func (h *HTTPHandler) writeLoop(ctx context.Context, ws Websocket, events <-chan eventstream.Event) error {
	pingTicker := time.NewTicker(h.pingPeriod)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-pingTicker.C:
			if err := ws.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
				return fmt.Errorf("set write deadline: %w", err)
			}
			if err := ws.WriteMessage(gorillaws.PingMessage, nil); err != nil {
				return fmt.Errorf("write ping message: %w", err)
			}
			h.logger.Debug("ping")

		case event, ok := <-events:
			if !ok {
				return errors.New("events stream was closed")
			}

			adapted, err := h.eventAdapter.Adapt(event)
			if err != nil {
				h.logger.With(zap.Error(err)).Error("cannot adapt event to out stream")
				continue
			}

			if err := ws.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
				return fmt.Errorf("set write deadline: %w", err)
			}

			wr, err := ws.NextWriter(gorillaws.TextMessage)
			if err != nil {
				return fmt.Errorf("get next writer: %w", err)
			}

			if err := h.eventWriter.Write(adapted, wr); err != nil {
				return fmt.Errorf("write data to connection: %w", err)
			}

			if err := wr.Close(); err != nil {
				return fmt.Errorf("flush writer: %w", err)
			}
		}
	}
}

func pongWait(ping time.Duration) time.Duration {
	return ping * 3 / 2
}
