package wsupgrader

import (
	"net"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/tomas-qstarrs/nano/env"
)

type Upgrader struct {
	*websocket.Upgrader
}

func NewUpgrader() *Upgrader {
	return &Upgrader{
		Upgrader: &websocket.Upgrader{
			ReadBufferSize:    10240,
			WriteBufferSize:   10240,
			CheckOrigin:       func(_ *http.Request) bool { return true },
			EnableCompression: env.WebSocketCompression,
		},
	}
}

func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request, params map[string]string) (net.Conn, error) {
	conn, err := u.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	return NewConn(r, conn), nil
}
