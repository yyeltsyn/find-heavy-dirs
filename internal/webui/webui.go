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

var serverStartTime time.Time
var lastRequestTime time.Time

var serverClosed chan int

func Start(dir string, limit int) error {
	serverStartTime = time.Now()
	serverClosed = make(chan int)
	defer close(serverClosed)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		w.Write(indexHtml)
	})

	http.HandleFunc("/api/top", func(w http.ResponseWriter, r *http.Request) {
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

		top, rest, total := core.Top(dir, limit)
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

	listener, err := net.Listen("tcp", ":0") // any available port
	if err != nil {
		return err
	}

	serverPort := listener.Addr().(*net.TCPAddr).Port
	err = startBrowser(serverPort, dir, limit)
	if err != nil {
		return err
	}

	return http.Serve(listener, nil)
}

func startBrowser(port int, dir string, limit int) error {
	values := url.Values{}
	values.Set("startDir", dir)
	values.Set("startLimit", strconv.Itoa(limit))
	url := fmt.Sprintf("http://localhost:%d/?%s", port, values.Encode())

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

func Active() bool {
	if serverClosed == nil {
		return true // server is not started yet
	}
	select {
	case <-serverClosed:
		return false // server is already closed
	default:
		// server is starting or serving, continue
	}
	if time.Since(serverStartTime) < warmupDuration {
		return true
	}
	if lastRequestTime.IsZero() {
		return false
	}

	return time.Since(lastRequestTime) < waitClientDuration
}
