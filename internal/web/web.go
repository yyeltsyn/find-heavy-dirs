package web

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

var serverStartTime *time.Time
var lastRequestTime *time.Time

var serverPort int
var serverStarted = make(chan int)

func StartServer() error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/html")
		w.Write(indexHtml)
	})

	http.HandleFunc("/top", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		lastRequestTime = &now

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

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}

	serverPort = listener.Addr().(*net.TCPAddr).Port

	now := time.Now()
	serverStartTime = &now

	close(serverStarted)

	return http.Serve(listener, nil)
}

func StartClient(dir string, limit int) error {
	<-serverStarted
	values := url.Values{}
	values.Set("startDir", dir)
	values.Set("startLimit", strconv.Itoa(limit))
	url := fmt.Sprintf("http://localhost:%d/?%s", serverPort, values.Encode())

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

func HasClients() bool {
	if serverStartTime == nil { // hmm, server is not started
		return false
	}
	if time.Since(*serverStartTime) < warmupDuration {
		return true
	}
	if lastRequestTime == nil {
		return false
	}

	return time.Since(*lastRequestTime) < waitClientDuration
}
