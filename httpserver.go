package main

import (
    "context"
    "embed"
    "encoding/json"
    "fmt"
    "html/template"
    "log"
    "log/slog"
    "net"
    "net/http"
    "os"
    "slices"
)

type Radio struct {
	Name string
	Url  string
}

type Context struct {
	PlayerList []*MpdClient
	RadioList  []Radio
	template   *template.Template
	ctx        context.Context
}

//go:embed template.html
var templateFile embed.FS

func (c *Context) indexHandler(w http.ResponseWriter, r *http.Request) {
	err := c.template.Execute(w, c)
	if err != nil {
		http.Error(w, fmt.Sprintf("Execute: %v", err), 500)
		return
	}
}

func (c *Context) selectHandler(w http.ResponseWriter, r *http.Request) {
	// slog.Debug("request", "r", r)
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("selectHandler: %v", err), 500)
		return
	}
	w.Header().Add("Content-Type", "text/html")
	for k, v := range r.Form {
		slog.Debug("select form", "k", k, "v", v)
        switch k {
        case "player":
			// TODO process the value
			err = c.template.ExecuteTemplate(w, "PlayerSelect", c)
        case "radio":
			err = c.template.ExecuteTemplate(w, "RadioSelect", c)
        }
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to execute template: %s", err), 500)
			return
		}
	}
}

func (c *Context) commonHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("commonHandler: %v", err), 500)
		return
	}
	//	slog.Debug("request", "r", r)
	w.Header().Add("Content-Type", "text/html")
	if r.Method == "PUT" && r.URL.Path == "/player" {
		player, err := NewMpdClient(c.ctx, r.Form.Get("playerHost"), r.Form.Get("playerPort"))
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to connect to player: %v", err), 500)
			return
		}
		c.PlayerList = append(c.PlayerList, player)
		err = c.Store()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to add player: %v", err), 500)
			return
		}
	} else if r.Method == "DELETE" && r.URL.Path == "/player" {
		err := c.RemovePlayer(r.Form.Get("playerHost"), r.Form.Get("playerPort"))
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to remove player: %v", err), 500)
			return
		}
	} else if r.Method == "PUT" && r.URL.Path == "/radio" {
		radio := Radio{
            Name: r.Form.Get("radioName"),
            Url:  r.Form.Get("radioUrl"),
        }
		c.RadioList = append(c.RadioList, radio)
		err := c.Store()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to add radio: %v", err), 500)
			return
		}
	} else if r.Method == "DELETE" && r.URL.Path == "/radio" {
		err := c.RemoveRadio(r.Form.Get("radio"))
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to remove radio: %v", err), 500)
			return
		}
	} else {
		http.Error(w, fmt.Sprintf("unknown combination of method %s and url %s", r.Method, r.URL.Path), 400)
		return
	}

	if r.URL.Path == "/player" {
		err = c.template.ExecuteTemplate(w, "PlayerSelect", c)
	} else if r.URL.Path == "/radio" {
		err = c.template.ExecuteTemplate(w, "RadioSelect", c)
	} else {
		http.Error(w, fmt.Sprintf("unknown request uri %s", r.URL.Path), 400)
	}
}

func (c *Context) commandHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("commonHandler: %v", err), 500)
		return
	}
	//	slog.Debug("request", "r", r)
	w.Header().Add("Content-Type", "text/html")
	playerUrl := r.Form.Get("player")
	player, err := c.FindPlayer(playerUrl)
	if err != nil {
		http.Error(w, fmt.Sprintf("player not found %s", playerUrl), 400)
		return
	}
	radioUrl := r.Form.Get("radio")
	if r.Form.Has("play") {
		err = c.Play(player, radioUrl)
	} else if r.Form.Has("stop") {
		err = c.Stop(player)
	} else if r.Form.Has("pause") {
		err = c.Pause(player)
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 500)
    }
    _, err = w.Write([]byte("ok"))
    if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 500)
    }
}

func (c *Context) RemoveRadio(name string) error {
    for i, r := range c.RadioList {
		if r.Url == name {
			c.RadioList = slices.Delete(c.RadioList, i, i+1)
            err := c.Store()
            if err != nil {
                return err
            }
			return nil
		}
	}
	return fmt.Errorf("radio not found")
}

func (c *Context) RemovePlayer(host string, port string) error {
	address := net.JoinHostPort(host, port)
	for i, p := range c.PlayerList {
		if p.Address == address {
			c.PlayerList = slices.Delete(c.PlayerList, i, i+1)
			err := c.Store()
            if err != nil {
                return err
            }
			return nil
		}
	}
	return fmt.Errorf("player not found")
}

func (c *Context) FindPlayer(url string) (*MpdClient, error) {
	for _, p := range c.PlayerList {
		if p.Address == url {
			return p, nil
		}
	}
	return nil, fmt.Errorf("player not found")
}

func (c *Context) Play(player *MpdClient, url string) error {
	_, err := player.Command("clear")
	if err != nil {
		return err
	}
	addIdData, err := player.Command(fmt.Sprintf("addid \"%s\" 0", url))
	if err != nil {
		return err
	}
	addIdData.Print()
	id, ok := addIdData.Response["Id"]
	if !ok {
		return fmt.Errorf("failed to get id of added song")
	}
	_, err = player.Command(fmt.Sprintf("playid %s", id))
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) Stop(player *MpdClient) error {
	_, err := player.Command("stop")
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) Pause(player *MpdClient) error {
	_, err := player.Command("pause")
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) Store() error {
	j, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return saveConfig(j)
}

func (c *Context) connectPlayer(p *MpdClient) {
    err := p.Connect(c.ctx)
	if err != nil {
		slog.Error("Failed to connect player %s: %s", p.Address, err)
	}
}

func Load() *Context {
	j, err := loadConfig()
	c := Context{}
	if err != nil {
		return &c
	}
    err = json.Unmarshal(j, &c)
    if err != nil {
        return &c
    }
	return &c
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

	http.HandleFunc("/", c.indexHandler)
	http.HandleFunc("/command", c.commandHandler)
	http.HandleFunc("/player", c.commonHandler)
	http.HandleFunc("/radio", c.commonHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}