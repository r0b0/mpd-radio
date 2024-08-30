package main

import (
    "fmt"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/widget"
)

func main() {
	a := app.New()
	w := a.NewWindow("MPD Radio")
	w.Resize(fyne.NewSize(800, 600))
	refreshableWidgets := make([]widget.Widget, 0)
	serverLabel := widget.NewLabel("Player")
	serverDropdown := widget.NewSelect([]string{"Add New..."}, func(s string) {
        fmt.Printf("player selected: %s\n", s)
    })
	urlLabel := widget.NewLabel("Radio")
	urlList := []string{"Add New..."}
	urlDropdown := widget.NewSelect(urlList, func(s string) {
        fmt.Printf("radio selected: %s\n", s)
        switch s {
        case "Add New...":
            AddNewRadio(w, func(radio Radio) {
                urlList = append(urlList, radio.name)
				refreshAllWidgets(refreshableWidgets)
            })
        }
    })
	// TODO buttons
	w.SetContent(container.NewVBox(
		serverLabel, serverDropdown, urlLabel, urlDropdown))
	refreshableWidgets = append(refreshableWidgets, urlDropdown)
	w.ShowAndRun()
}

type Radio struct {
	name string
	url	 string
}

func AddNewRadio(parent fyne.Window, cb func(Radio)) {
	nameEntry := widget.NewEntry()
	urlEntry := widget.NewEntry()
	formItems := []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("URL", urlEntry),
	}
	dialog.ShowForm("Add new radion",
		"OK",
		"Cancel",
		formItems,
        func(b bool) {
			cb(Radio{nameEntry.Text, urlEntry.Text})
        },
		parent)
}

func refreshAllWidgets(widgets []widget.BaseWidget) {
	for _, w := range widgets {
		w.Refresh()
	}
}