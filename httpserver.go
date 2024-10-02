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

	if r.Method == "GET" && r.URL.Path == "/" {
		templateName = "template.html"
	} else if r.URL.Path == "/status" {
		templateName = "Status"
		err = c.updateStatus(r.Form.Get("player"))
		if err != nil {
			httpError(w, 500, "failed to get player status", "error", err)
			return
		}
	} else if r.URL.Path == "/command" {
		templateName = "Status"
		playerUrl := r.Form.Get("player")
		player, err := c.FindPlayer(playerUrl)
		if err != nil {
			httpError(w, 400, "player not found", "error", err, "url", playerUrl)
			return
		}
		radioUrl := r.Form.Get("radio")
		if r.Form.Has("play") {
			c.Status = "playing"
			err = c.Play(player, radioUrl)
		} else if r.Form.Has("stop") {
			c.Status = "stopping"
			err = c.Stop(player)
		} else if r.Form.Has("pause") {
			c.Status = "pausing"
			err = c.Pause(player)
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
		player, err := NewMpdClient(c.ctx, r.Form.Get("playerHost"), r.Form.Get("playerPort"))
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
		err := c.RemovePlayer(r.Form.Get("player"))
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
		err := c.RemoveRadio(r.Form.Get("radio"))
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
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	listenFlag := flag.String("p", "127.0.0.1:6680", "listen address and port")
	flag.Parse()

	c := Load()

	t, err := template.ParseFS(templateFile, "*.*")
	if err != nil {
		slog.Error("failed to parse template %v", err)
		return
	}
	c.template = t
	c.ctx = context.Background()

	for _, p := range c.PlayerList {
		go c.connectPlayer(p)
	}

	http.HandleFunc("/", c.commonHandler)
	http.HandleFunc("/player", c.commonHandler)
	http.HandleFunc("/radio", c.commonHandler)
	http.HandleFunc("/command", c.commonHandler)

	static, _ := fs.Sub(staticFiles, ".")
	http.Handle("/static/", http.FileServerFS(static))

	log.Fatal(http.ListenAndServe(*listenFlag, nil))
}
