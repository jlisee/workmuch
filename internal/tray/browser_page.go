package tray

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const browserPageLifetime = time.Minute

func serveBrowserPage(page []byte) (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("listen for browser page: %w", err)
	}

	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		_ = listener.Close()
		return "", fmt.Errorf("create browser page URL: %w", err)
	}
	pagePath := "/" + hex.EncodeToString(token[:])
	served := make(chan struct{})
	var servedOnce sync.Once

	mux := http.NewServeMux()
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	mux.HandleFunc(pagePath, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writer.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		writer.Header().Set("Cache-Control", "no-store")
		writer.Header().Set("Content-Length", strconv.Itoa(len(page)))
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		if request.Method == http.MethodHead {
			return
		}
		_, _ = writer.Write(page)
		servedOnce.Do(func() { close(served) })
	})

	go func() {
		_ = server.Serve(listener)
	}()
	go func() {
		timer := time.NewTimer(browserPageLifetime)
		defer timer.Stop()
		select {
		case <-served:
		case <-timer.C:
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	return "http://" + listener.Addr().String() + pagePath, nil
}
