package main

import (
	"context"
	"embed"
	"flag"
	"html/template"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
)

//go:embed template.html
var templateFile embed.FS

//go:embed static
var staticFiles embed.FS

func httpError(w http.ResponseWriter, code int, message string, args ...any) {
	slog.Error(message, args)
	http.Error(w, message, code)
}

func (c *Context) commonHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		httpError(w, 400, "failed to parse form", "error", err)
		return
	}
	//	slog.Debug("request", "r", r)
	w.Header().Add("Content-Type", "text/html")
	var templateName string
	playerUrl := r.Form.Get("player")
	radioUrl := r.Form.Get("radio")
	player := c.FindPlayer(playerUrl)

	if r.Method == "GET" && r.URL.Path == "/" {
		templateName = "template.html"
	} else if r.Method == "GET" && r.URL.Path == "/player" {
		templateName = "PlayerSelect"
	} else if r.URL.Path == "/status" {
		templateName = "Status"
		err = c.UpdateStatus(playerUrl)
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
			c.Status = "playing"
			c.IsPlaying = true
			err = c.Play(player, radioUrl)
		} else if r.Form.Has("stop") {
			c.Status = "stopping"
			c.IsPlaying = false
			err = c.Stop(player)
		} else if r.Form.Has("pause") {
			c.Status = "pausing"
			c.IsPlaying = false
			err = c.Pause(player)
		} else if r.Form.Has("volume_up") {
			templateName = "VolumeRange"
			err = c.UpdateVolume(player, 10)
		} else if r.Form.Has("volume_down") {
			templateName = "VolumeRange"
			err = c.UpdateVolume(player, -10)
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
		player, err := NewMpdClient(c.ctx,
			r.Form.Get("playerHost"),
			r.Form.Get("playerPort"),
			slog.Default())
		if err != nil {
			httpError(w, 500, "failed to connect to mpd server", "error", err)
			return
		}
		c.PlayerList = append(c.PlayerList, player)
		err = c.Store()
		if err != nil {
			httpError(w, 500, "failed to store app status", "error", err)
			return
		}
	} else if r.Method == "DELETE" && r.URL.Path == "/player" {
		templateName = "PlayerSelect"
		err := c.RemovePlayer(playerUrl)
		if err != nil {
			httpError(w, 500, "failed to remove player", "error", err)
			return
		}
	} else if r.Method == "PUT" && r.URL.Path == "/radio" {
		templateName = "RadioSelect"
		radio := Radio{
			Name: r.Form.Get("radioName"),
			Url:  r.Form.Get("radioUrl"),
		}
		c.RadioList = append(c.RadioList, radio)
		err := c.Store()
		if err != nil {
			httpError(w, 500, "failed to store app status", "error", err)
			return
		}
	} else if r.Method == "DELETE" && r.URL.Path == "/radio" {
		templateName = "RadioSelect"
		err := c.RemoveRadio(radioUrl)
		if err != nil {
			httpError(w, 500, "failed to remove radio", "error", err)
			return
		}
	} else {
		httpError(w, 404, "uknonwn combination of method and url", "method", r.Method, "url", r.URL.Path)
		return
	}

	if templateName != "" {
		err = c.template.ExecuteTemplate(w, templateName, c)
		if err != nil {
			httpError(w, 500, "failed to execute template", "error", err)
			return
		}
	} else {
		httpError(w, 404, "uknonwn request url", "url", r.URL.Path)
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

	t, err := template.ParseFS(templateFile, "*.*")
	if err != nil {
		slog.Error("failed to parse template %v", err)
		return
	}
	c.template = t
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
	} else {
		return a
	}
}
