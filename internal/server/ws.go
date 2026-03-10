package server

import (
	"io"

	"github.com/go-kratos/kratos/v2/transport/http"
	"golang.org/x/net/websocket"
)

func registerWSServer(srv *http.Server) {
	srv.Handle("/api/v1/ws/start", websocket.Handler(handleWS))
}

func handleWS(conn *websocket.Conn) {
	defer conn.Close()

	for {
		var in string
		if err := websocket.Message.Receive(conn, &in); err != nil {
			if err == io.EOF {
				return
			}
			_ = websocket.Message.Send(conn, "error: "+err.Error())
			return
		}
		if err := websocket.Message.Send(conn, in); err != nil {
			return
		}
	}
}
