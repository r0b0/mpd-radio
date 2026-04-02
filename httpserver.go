package main

import (
	"context"
	"embed"
	"flag"
	htmlTemplate "html/template"
	"io/fs"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	textTemplate "text/template"
	"time"
)

//go:embed template.html
var htmlTemplateFile embed.FS

//go:embed template.txt
var textTemplateFile embed.FS

//go:embed static
var staticFiles embed.FS

func httpError(w http.ResponseWriter, code int, message string, args ...any) {
	slog.Error(message, args...)
	http.Error(w, message, code)
}

func (a *Application) commonHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		httpError(w, 400, "failed to parse form", "error", err)
		return
	}
	//	slog.Debug("request", "r", r)

	var templateName string
	playerUrl := r.Form.Get("player")
	radioUrl := r.Form.Get("radio")
	player := a.FindPlayer(playerUrl)

	if r.Method == "GET" && r.URL.Path == "/" {
		templateName = "template.html"
	} else if r.Method == "GET" && r.URL.Path == "/player" {
		templateName = "PlayerSelect"
	} else if r.URL.Path == "/status" {
		templateName = "Status"
		err = a.UpdateStatus(playerUrl)
		if err != nil {
			httpError(w, 500, "failed to get player status", "error", err)
			return
		}
	} else if r.URL.Path == "/command" {
		templateName = "Status"
		if player == nil {
			httpError(w, 400, "player not found", "error", err, "url", playerUrl)
			return
		}

		if r.Form.Has("play") {
			a.Status = "playing"
			a.IsPlaying = true
			err = a.Play(player, radioUrl)
		} else if r.Form.Has("stop") {
			a.Status = "stopping"
			a.IsPlaying = false
			err = a.Stop(player)
		} else if r.Form.Has("pause") {
			a.Status = "pausing"
			a.IsPlaying = false
			err = a.Pause(player)
		} else if r.Form.Has("volume_up") {
			templateName = "VolumeRange"
			err = a.UpdateVolume(player, 10)
		} else if r.Form.Has("volume_down") {
			templateName = "VolumeRange"
			err = a.UpdateVolume(player, -10)
		} else {
			httpError(w, 400, "unknown command")
			return
		}
		if err != nil {
			httpError(w, 500, "failed processing command", "error", err)
			return
		}
	} else if r.Method == "PUT" && r.URL.Path == "/player" {
		templateName = "PlayerSelect"
		address := net.JoinHostPort(r.Form.Get("playerHost"), r.Form.Get("playerPort"))
		player = &MpdClient{
			Address: address,
			logger:  slog.Default().With("player address", address),
			lastUse: time.Now(),
		}
		a.PlayerList = append(a.PlayerList, player)
		err = a.Store()
		if err != nil {
			httpError(w, 500, "failed to store app status", "error", err)
			return
		}
	} else if r.Method == "DELETE" && r.URL.Path == "/player" {
		templateName = "PlayerSelect"
		err := a.RemovePlayer(playerUrl)
		if err != nil {
			httpError(w, 500, "failed to remove player", "error", err)
			return
		}
	} else if r.Method == "GET" && r.URL.Path == "/radio" {
		templateName = "RadioSelect"
	} else if r.Method == "PUT" && r.URL.Path == "/radio" {
		templateName = "RadioSelect"
		radio := Radio{
			Name: r.Form.Get("radioName"),
			Url:  r.Form.Get("radioUrl"),
		}
		a.RadioList = append(a.RadioList, radio)
		err := a.Store()
		if err != nil {
			httpError(w, 500, "failed to store app status", "error", err)
			return
		}
	} else if r.Method == "DELETE" && r.URL.Path == "/radio" {
		templateName = "RadioSelect"
		err := a.RemoveRadio(radioUrl)
		if err != nil {
			httpError(w, 500, "failed to remove radio", "error", err)
			return
		}
	} else {
		httpError(w, 404, "unknown combination of method and url", "method", r.Method, "url", r.URL.Path)
		return
	}

	accept := r.Header.Get("Accept")
	if strings.HasPrefix(accept, "text/plain") {
		w.Header().Add("Content-Type", "text/plain")
		err = a.textTemplate.ExecuteTemplate(w, templateName, a)
		if err != nil {
			httpError(w, 500, "failed to execute template", "error", err)
			return
		}
	} else if strings.HasPrefix(accept, "text/html") ||
		strings.HasPrefix(accept, "text/*") ||
		strings.HasPrefix(accept, "*/") {
		w.Header().Add("Content-Type", "text/html")
		err = a.htmlTemplate.ExecuteTemplate(w, templateName, a)
		if err != nil {
			httpError(w, 500, "failed to execute template", "error", err)
			return
		}
	} else {
		httpError(w, 404, "unknown request url", "url", r.URL.Path, "accept", accept)
		return
	}
}

func main() {
	listenFlag := flag.String("p", "127.0.0.1:6680", "listen address and port")
	quietFlag := flag.Bool("q", false, "skip debugging output")
	skipTimeStampFlag := flag.Bool("t", false, "skip timestamps in output")
	flag.Parse()

	handlerOptions := slog.HandlerOptions{}
	if !*quietFlag {
		handlerOptions.Level = slog.LevelDebug
	}
	if *skipTimeStampFlag {
		handlerOptions.ReplaceAttr = replaceAttrFuncRemoveTime
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &handlerOptions))
	slog.SetDefault(logger)

	c := Load()

	t, err := htmlTemplate.ParseFS(htmlTemplateFile, "*.*")
	if err != nil {
		slog.Error("failed to parse html template", slog.Any("error", err))
		return
	}
	c.htmlTemplate = t

	tt, err := textTemplate.ParseFS(textTemplateFile, "*.*")
	if err != nil {
		slog.Error("failed to parse txt template", slog.Any("error", err))
		return
	}
	c.textTemplate = tt
	c.ctx = context.Background()

	for _, p := range c.PlayerList {
		go c.ConnectPlayer(p)
	}

	http.HandleFunc("/", c.commonHandler)

	static, _ := fs.Sub(staticFiles, ".")
	http.Handle("/static/", http.FileServerFS(static))

	log.Fatal(http.ListenAndServe(*listenFlag, nil))
}

// helper function not to log timestamp
func replaceAttrFuncRemoveTime(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.Attr{}
	}

	return a
}
