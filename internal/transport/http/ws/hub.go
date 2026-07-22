package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/labstack/echo/v4"

	"github.com/Jagerente/matcharr/internal/apperr"
	"github.com/Jagerente/matcharr/internal/service/session"
	"github.com/Jagerente/matcharr/internal/transport/http/api"
)

const writeTimeout = 5 * time.Second

type SessionGateway interface {
	Authenticate(code, token string) (session.You, error)
	State(code, token string) (session.PersonalView, error)
	SetPresence(code, participantID string, online bool)
}

// Limits cap the event sockets held open per participant and per session.
// Zero leaves one unbounded. A phone that reconnects before the old socket
// is reaped needs more than one per client.
type Limits struct {
	ConnsPerClient  int
	ConnsPerSession int
}

type Hub struct {
	logger   *slog.Logger
	sessions SessionGateway
	limits   Limits

	mu    sync.Mutex
	rooms map[string]map[*client]struct{}
}

type client struct {
	participantID string
	conn          *websocket.Conn
}

func NewHub(logger *slog.Logger, sessions SessionGateway, limits Limits) *Hub {
	return &Hub{
		logger:   logger,
		sessions: sessions,
		limits:   limits,
		rooms:    make(map[string]map[*client]struct{}),
	}
}

func (h *Hub) HandleEvent(code string, ev session.Event) {
	payload, err := json.Marshal(mapEvent(ev))
	if err != nil {
		h.logger.Error("event serialization", "err", err)
		return
	}
	h.broadcast(code, payload)
}

func (h *Hub) broadcast(code string, payload []byte) {
	h.mu.Lock()
	clients := make([]*client, 0, len(h.rooms[code]))
	for c := range h.rooms[code] {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	for _, c := range clients {
		// A per-write timeout: one stuck client must not stall the rest.
		ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
		err := c.conn.Write(ctx, websocket.MessageText, payload)
		cancel()
		if err != nil {
			_ = c.conn.Close(websocket.StatusAbnormalClosure, "write failed")
		}
	}
}

func (h *Hub) Handle(c echo.Context) error {
	code := c.Param("code")
	token := c.QueryParam("token")

	you, err := h.sessions.Authenticate(code, token)
	if err != nil {
		return err
	}
	if !h.hasRoom(code, you.ID) {
		return apperr.New(apperr.KindTooMany, apperr.ReasonRateLimited, "too many open connections")
	}

	// The socket outlives any single request, so the server's ReadTimeout,
	// which is still armed on the connection at this point, must go.
	if err := http.NewResponseController(c.Response()).SetReadDeadline(time.Time{}); err != nil {
		h.logger.Warn("ws deadline", "code", code, "err", err)
	}

	conn, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{
		// The Origin check is deliberately off: the app lives behind a
		// proxy, on a direct port, or over Tailscale, the real
		// authentication here is the token, not the Origin.
		InsecureSkipVerify: true,
	})
	if err != nil {
		// The handshake hijacked the connection; there is nothing left for
		// the HTTP error handler to write to.
		h.logger.Warn("ws handshake failed", "code", code, "err", err)
		return nil
	}

	member := &client{participantID: you.ID, conn: conn}
	h.join(code, member)
	defer h.leave(code, member)

	h.sendSnapshot(c.Request().Context(), conn, code, token)

	// Clients never send anything over WS (all actions are REST); the read
	// loop only services control frames and detects the close.
	ctx := conn.CloseRead(c.Request().Context())
	<-ctx.Done()
	_ = conn.Close(websocket.StatusNormalClosure, "")
	return nil
}

func (h *Hub) sendSnapshot(ctx context.Context, conn *websocket.Conn, code, token string) {
	view, err := h.sessions.State(code, token)
	if err != nil {
		return
	}
	payload, err := json.Marshal(snapshotMessage{
		Type:  "snapshot",
		State: api.MapState(view.State),
		You:   api.MapYou(view.You),
	})
	if err != nil {
		return
	}
	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	_ = conn.Write(writeCtx, websocket.MessageText, payload)
}

func (h *Hub) Connections() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	total := 0
	for _, room := range h.rooms {
		total += len(room)
	}
	return total
}

func (h *Hub) hasRoom(code, participantID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if perSession := h.limits.ConnsPerSession; perSession > 0 && len(h.rooms[code]) >= perSession {
		return false
	}
	perClient := h.limits.ConnsPerClient
	return perClient <= 0 || h.connCountLocked(code, participantID) < perClient
}

func (h *Hub) join(code string, c *client) {
	h.mu.Lock()
	if h.rooms[code] == nil {
		h.rooms[code] = make(map[*client]struct{})
	}
	h.rooms[code][c] = struct{}{}
	first := h.connCountLocked(code, c.participantID) == 1
	h.mu.Unlock()

	if first {
		h.sessions.SetPresence(code, c.participantID, true)
	}
}

func (h *Hub) leave(code string, c *client) {
	h.mu.Lock()
	delete(h.rooms[code], c)
	if len(h.rooms[code]) == 0 {
		delete(h.rooms, code)
	}
	last := h.connCountLocked(code, c.participantID) == 0
	h.mu.Unlock()

	if last {
		h.sessions.SetPresence(code, c.participantID, false)
	}
}

func (h *Hub) connCountLocked(code, participantID string) int {
	count := 0
	for c := range h.rooms[code] {
		if c.participantID == participantID {
			count++
		}
	}
	return count
}
