package main

import (
    "fmt"
    "html/template"
    "log"
    "log/slog"
    "net/http"
    "os"
)

type Context struct {
	PlayerList []*MpdClient
	RadioList  []Radio
	template   *template.Template
}

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
			err = c.template.ExecuteTemplate(w, "PlayerSelect", c)
        }
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to execute template: %s", err), 500)
			return
		}
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	
	t, err := template.ParseFiles("index.tmpl")
	if err != nil {
		panic("Failed to parse template")
	}
	c := Context{template: t}
	http.HandleFunc("/", c.indexHandler)
	http.HandleFunc("/player_selected", c.selectHandler)
	http.HandleFunc("/radio_selected", c.selectHandler)
	
	log.Fatal(http.ListenAndServe(":8080", nil))
}