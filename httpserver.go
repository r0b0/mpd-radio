package main

import (
    "context"
    "embed"
    "fmt"
    "html/template"
    "log"
    "log/slog"
    "net/http"
    "os"
)

//go:embed template.html
var templateFile embed.FS

func (c *Context) commonHandler(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        slog.Error("failed to parse form", "error", err)
        http.Error(w, fmt.Sprintf("commonHandler: %v", err), 400)
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
            slog.Error("failed to get player status", "error", err)
            http.Error(w, "failed to get player status", 500)
            return
        }
    } else if r.Method == "PUT" && r.URL.Path == "/player" {
        templateName = "PlayerSelect"
        player, err := NewMpdClient(c.ctx, r.Form.Get("playerHost"), r.Form.Get("playerPort"))
        if err != nil {
            slog.Error("failed to connect to mpd server", "error", err)
            http.Error(w, fmt.Sprintf("failed to connect to player: %v", err), 500)
            return
        }
        c.PlayerList = append(c.PlayerList, player)
        err = c.Store()
        if err != nil {
            slog.Error("failed to store app status", "error", err)
            http.Error(w, fmt.Sprintf("failed to add player: %v", err), 500)
            return
        }
    } else if r.Method == "DELETE" && r.URL.Path == "/player" {
        templateName = "PlayerSelect"
        err := c.RemovePlayer(r.Form.Get("player"))
        if err != nil {
            slog.Error("failed to remove player", "error", err)
            http.Error(w, fmt.Sprintf("failed to remove player: %v", err), 500)
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
            slog.Error("failed to store app status", "error", err)
            http.Error(w, fmt.Sprintf("failed to add radio: %v", err), 500)
            return
        }
    } else if r.Method == "DELETE" && r.URL.Path == "/radio" {
        templateName = "RadioSelect"
        err := c.RemoveRadio(r.Form.Get("radio"))
        if err != nil {
            slog.Error("failed to remove radio", "error", err)
            http.Error(w, fmt.Sprintf("failed to remove radio: %v", err), 500)
            return
        }
    } else {
        slog.Error("unkonown combination of method and url", "method", r.Method, "url", r.URL.Path)
        http.Error(w, fmt.Sprintf("unknown combination of method %s and url %s", r.Method, r.URL.Path), 404)
        return
    }

    if templateName != "" {
        err = c.template.ExecuteTemplate(w, templateName, c)
        if err != nil {
            slog.Error("failed to execute templateName", "error", err)
            http.Error(w, fmt.Sprintf("failed to execute templateName: %s", err), 500)
            return
        }
    } else {
        slog.Error("unknown request url", "error", err)
        http.Error(w, fmt.Sprintf("unknown request uri %s", r.URL.Path), 400)
    }
}

func (c *Context) commandHandler(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        slog.Error("failed to parse form", "error", err)
        http.Error(w, fmt.Sprintf("commonHandler: %v", err), 400)
        return
    }
    //	slog.Debug("request", "r", r)
    w.Header().Add("Content-Type", "text/html")
    playerUrl := r.Form.Get("player")
    player, err := c.FindPlayer(playerUrl)
    if err != nil {
        slog.Error("player not found", "error", err)
        http.Error(w, fmt.Sprintf("player not found %s", playerUrl), 400)
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
        slog.Error("unknown command")
        http.Error(w, fmt.Sprintf("unkwnon command"), 400)
    }
    if err != nil {
        slog.Error("failed processing command", "error", err)
        http.Error(w, fmt.Sprintf("%s", err), 500)
        return
    }
    err = c.template.ExecuteTemplate(w, "Status", c)
    if err != nil {
        http.Error(w, fmt.Sprintf("%s", err), 500)
    }
}

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
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
        go c.connectPlayer(p)
    }

    http.HandleFunc("/", c.commonHandler)
    http.HandleFunc("/player", c.commonHandler)
    http.HandleFunc("/radio", c.commonHandler)

    http.HandleFunc("/command", c.commandHandler)

    log.Fatal(http.ListenAndServe(":8080", nil))
}
