package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/net/websocket"

	"momoko/pkg/servercore"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:18080", "临时 WS 服务监听地址")
	flag.Parse()

	// exePath, workDir, err := resolveBedrockPath()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	exePath := "cmd.exe"
	workDir := ""

	server, err := servercore.NewServer(servercore.ServerConfig{
		ID:               "bedrock-manual",
		Command:          exePath,
		Dir:              workDir,
		LogLimit:         1000,
		SubscriberBuffer: 256,
	})
	if err != nil {
		log.Fatalf("创建服务端实例失败: %v", err)
	}
	if err := server.Start(); err != nil {
		log.Fatalf("启动 bedrock_server.exe 失败: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/ws", websocket.Handler(func(conn *websocket.Conn) {
		handleWS(conn, server)
	}))

	httpServer := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	log.Printf("临时 WS 服务已启动: ws://%s/ws", *addr)
	log.Printf("目标程序: %s", exePath)
	log.Printf("已主动启动目标进程")
	log.Printf("WS 连接建立后将直接接入控制台")

	defer func() {
		_ = server.Stop()
	}()

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("启动临时 WS 服务失败: %v", err)
	}
}

func resolveBedrockPath() (string, string, error) {
	exeRelPath := filepath.Join("test", "bedrock-server", "bedrock_server.exe")
	exeAbsPath, err := filepath.Abs(exeRelPath)
	if err != nil {
		return "", "", fmt.Errorf("解析 bedrock_server.exe 路径失败: %w", err)
	}

	if _, err := os.Stat(exeAbsPath); err != nil {
		if os.IsNotExist(err) {
			return "", "", errors.New("未找到 test/bedrock-server/bedrock_server.exe")
		}
		return "", "", fmt.Errorf("检查 bedrock_server.exe 失败: %w", err)
	}

	return exeAbsPath, filepath.Dir(exeAbsPath), nil
}

func handleWS(conn *websocket.Conn, server *servercore.Server) {
	defer conn.Close()

	var writeMu sync.Mutex
	send := func(text string) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return websocket.Message.Send(conn, text)
	}

	done := make(chan struct{})
	defer close(done)

	// 新连接建立后，先把最近日志补发给客户端，再进入实时推送。
	for _, entry := range server.RecentLogs() {
		if err := send(entry.Text); err != nil {
			return
		}
	}

	logCh, cancel := server.Subscribe()
	defer cancel()

	go func() {
		for {
			select {
			case <-done:
				return
			case entry := <-logCh:
				if err := send(entry.Text); err != nil {
					return
				}
			}
		}
	}()

	for {
		var input string
		if err := websocket.Message.Receive(conn, &input); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			return
		}
		if err := server.Send(input); err != nil {
			_ = send("error: " + err.Error())
		}
	}
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}
