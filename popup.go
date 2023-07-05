package main

import "fyne.io/fyne/v2"

func CreatePopup(a fyne.App, title string, content fyne.CanvasObject) {
	w := a.NewWindow(title)
	w.CenterOnScreen()
	w.SetContent(content)
	w.Show()
}
