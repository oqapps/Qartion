package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/fstanis/screenresolution"
)

var icons = []string{"Windows", "macOS", "Linux"}
var iconsLow = []string{"windows", "darwin", "linux"}

func updateConfig(key string, value interface{}) {
	var settings map[string]interface{}
	settingsFile, _ := os.ReadFile("settings.json")
	json.Unmarshal(settingsFile, &settings)
	settings[key] = value
	data, _ := json.Marshal(settings)

	os.WriteFile("settings.json", data, fs.ModeAppend)
}

func getDefaultIconTheme() (string, int) {
	var settings map[string]interface{}
	settingsFile, e := os.ReadFile("settings.json")
	if e != nil {
		return "windows", 0
	}
	e = json.Unmarshal(settingsFile, &settings)
	if e != nil {
		return "windows", 0
	}
	if settings["iconTheme"] != nil {
		i := fmt.Sprintf("%v", settings["iconTheme"])
		a, e := strconv.Atoi(i)
		if e != nil {
			return "windows", 0
		}
		return iconsLow[a], a
	}
	switch runtime.GOOS {
	case "windows":
		return runtime.GOOS, 0
	case "darwin":
		return runtime.GOOS, 1
	case "linux":
		return runtime.GOOS, 2
	}
	return "windows", 0
}

func LaunchSettings(a fyne.App) {
	w := a.NewWindow("Partition Mounter")
	w.CenterOnScreen()
	iconThemeSelect := widget.NewSelect(icons, func(s string) {
		var index int
		for in, i := range icons {
			if s == i {
				index = in
				break
			}
		}
		low := iconsLow[index]
		updateConfig("iconTheme", index)
		di, _ := os.ReadFile(fmt.Sprintf("disk-icon-%s.png", low))
		diskIcon = *widget.NewIcon(fyne.NewStaticResource("disk-icon", di))
	})
	_, i := getDefaultIconTheme()
	iconThemeSelect.SetSelectedIndex(i)

	settingsContainer := container.NewVBox(container.NewHBox(widget.NewLabel("Icon Theme"), iconThemeSelect))
	card := widget.NewCard("Settings", "", settingsContainer)
	resolution := screenresolution.GetPrimary()
	w.Resize(fyne.NewSize(float32(resolution.Width)/2, float32(resolution.Height)/2))

	w.SetContent(card)
	w.Show()
}