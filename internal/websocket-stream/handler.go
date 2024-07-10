package websocketstream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
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
}

func NewHTTPHandler(opts Options) (*HTTPHandler, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %v", err)
	}

	return &HTTPHandler{Options: opts}, nil
}

func (h *HTTPHandler) Serve(eCtx echo.Context) error {
	ctx := eCtx.Request().Context()
	userID := middlewares.MustUserID(eCtx)

	ws, err := h.upgrader.Upgrade(eCtx.Response(), eCtx.Request(), nil)
	if err != nil {
		return fmt.Errorf("upgrade ws: %v", err)
	}

	closer := newWsCloser(h.logger, ws)

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		pongDeadline := h.pingPeriod + (h.pingPeriod / 2)

		if err := ws.SetReadDeadline(time.Now().Add(pongDeadline)); err != nil {
			if websocket.IsCloseError(err) {
				return nil
			}

			return fmt.Errorf("set deadline for read pong: %v", err)
		}

		ws.SetPongHandler(func(string) error {
			h.logger.Debug("pong")

			return ws.SetReadDeadline(time.Now().Add(pongDeadline))
		})

		return h.readLoop(ctx, ws)
	})

	eg.Go(func() error {
		events, err := h.eventStream.Subscribe(ctx, userID)
		if err != nil {
			return fmt.Errorf("subscribe to events for user %v: %v", userID, err)
		}

		return h.writeLoop(ctx, ws, events)
	})

	eg.Go(func() error {
		<-h.shutdownCh

		h.logger.Info("graceful shutdown websocket service")

		closer.Close(websocket.CloseNormalClosure)
		return nil
	})

	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		if websocket.IsCloseError(err, websocket.CloseNoStatusReceived) {
			h.logger.Warn("ws closed", zap.Error(err))
			return nil
		}

		return fmt.Errorf("wait ws stop: %v", err)
	}
	return nil
}

// readLoop listen PONGs.
func (h *HTTPHandler) readLoop(_ context.Context, ws Websocket) error {
	for {
		if _, _, err := ws.NextReader(); err != nil {
			if websocket.IsCloseError(err) {
				return nil
			}

			return fmt.Errorf("next read msg: %v", err)
		}
	}
}

// writeLoop listen events and writes them into Websocket.
func (h *HTTPHandler) writeLoop(_ context.Context, ws Websocket, events <-chan eventstream.Event) error {
	t := time.NewTicker(h.pingPeriod)

	for {
		select {
		case event, ok := <-events:
			if !ok {
				if err := ws.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					if websocket.IsCloseError(err) {
						return nil
					}

					return fmt.Errorf("write close msg: %v", err)
				}
			}

			if err := h.writeEvent(ws, event); err != nil {
				return err
			}
		case <-t.C:
			if err := h.writePing(ws); err != nil {
				return err
			}
		}
	}
}

func (h *HTTPHandler) writeEvent(ws Websocket, event eventstream.Event) error {
	if err := ws.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		if websocket.IsCloseError(err) {
			return nil
		}

		return fmt.Errorf("set write deadline: %v", err)
	}

	w, err := ws.NextWriter(websocket.TextMessage)
	if err != nil && !websocket.IsCloseError(err) {
		return fmt.Errorf("get next writer: %v", err)
	}

	ae, err := h.eventAdapter.Adapt(event)
	if err != nil {
		return fmt.Errorf("adapt event for send: %v", err)
	}

	if err := h.eventWriter.Write(&ae, w); err != nil {
		return fmt.Errorf("write event: %v", err)
	}

	w.Close() //nolint:revive // ignore unhandled error
	return nil
}

func (h *HTTPHandler) writePing(ws Websocket) error {
	if err := ws.SetWriteDeadline(time.Now().Add(h.pingPeriod)); err != nil {
		if websocket.IsCloseError(err) {
			return nil
		}

		return fmt.Errorf("set write deadline: %v", err)
	}

	if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
		if websocket.IsCloseError(err) {
			return nil
		}

		return fmt.Errorf("send ping msg: %v", err)
	}
	h.logger.Debug("ping")
	return nil
}
