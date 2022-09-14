package webui

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/yyeltsyn/find-heavy-dirs/internal/core"
)

const warmupDuration = 5 * time.Second
const waitClientDuration = 3 * time.Second

//go:embed index.html
var indexHtml []byte

var startTime time.Time
var lastRequestTime time.Time

func Start(core1 *core.Core, dir string, limit int) error {
	startTime = time.Now()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		w.Write(indexHtml)
	})

	mux.HandleFunc("/api/top", func(w http.ResponseWriter, r *http.Request) {
		lastRequestTime = time.Now()

		dir := r.URL.Query().Get("dir")
		if dir == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		top, rest, total := core1.Top(dir, limit)
		output, err := json.Marshal(map[string]interface{}{
			"top":   top,
			"rest":  rest,
			"total": total,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(output)
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0") // any available port
	if err != nil {
		return err
	}

	err = startBrowser(listener.Addr().String(), dir, limit)
	if err != nil {
		return err
	}

	srv := &http.Server{Handler: mux}

	go func() {
		for hasActiveClients() {
			time.Sleep(100 * time.Millisecond)
		}
		srv.Close() // Note: ignore errors
	}()

	err = srv.Serve(listener)
	if err != http.ErrServerClosed {
		return err
	}

	return fmt.Errorf("no active clients, server closed")
}

func startBrowser(addr string, dir string, limit int) error {
	values := url.Values{}
	values.Set("startDir", dir)
	values.Set("startLimit", strconv.Itoa(limit))
	url := fmt.Sprintf("http://%s/?%s", addr, values.Encode())

	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	return err
}

func hasActiveClients() bool {
	if time.Since(startTime) < warmupDuration {
		return true
	}
	if lastRequestTime.IsZero() {
		return false
	}

	return time.Since(lastRequestTime) < waitClientDuration
}
