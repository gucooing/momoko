package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"

	"momoko/pkg/shellcmd"
)

func main() {
	http.Handle("/ws", websocket.Handler(handleWS))
	fmt.Println("ws demo listen on :55555, path: /ws")
	log.Fatal(http.ListenAndServe(":55555", nil))
}

func handleWS(conn *websocket.Conn) {
	defer conn.Close()
	req := conn.Request()
	key := req.URL.Query().Get("key")
	if key != "123456" {
		return
	}

	session, err := shellcmd.Start(context.Background(), "cmd", []string{}, shellcmd.Options{})
	if err != nil {
		_ = websocket.Message.Send(conn, "error: "+err.Error())
		return
	}

	var writeMu sync.Mutex
	send := func(msg string) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return websocket.Message.Send(conn, msg)
	}

	var streamWG sync.WaitGroup
	streamWG.Add(2)
	go func() {
		defer streamWG.Done()
		for chunk := range session.Stdout() {
			if err := send(string(chunk)); err != nil {
				return
			}
		}
	}()
	go func() {
		defer streamWG.Done()
		for chunk := range session.Stderr() {
			if err := send(string(chunk)); err != nil {
				return
			}
		}
	}()

	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		result, waitErr := session.Wait()
		if waitErr != nil {
			_ = send(fmt.Sprintf("exit(%d): %v", result.ExitCode, waitErr))
			return
		}
		_ = send(fmt.Sprintf("exit(%d)", result.ExitCode))
	}()

	for {
		var in string
		if err := websocket.Message.Receive(conn, &in); err != nil {
			if err != io.EOF {
				_ = send("error: " + err.Error())
			}
			break
		}

		if in == "__close_stdin__" {
			_ = session.CloseStdin()
			continue
		}

		if _, err := session.WriteString(in); err != nil {
			_ = send("error: " + err.Error())
			break
		}
	}

	_ = session.Kill()
	<-waitDone
	streamWG.Wait()
}
