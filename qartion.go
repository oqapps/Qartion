package main

import (
	"fmt"
	"os"
	"runtime"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	"fyne.io/fyne/v2/layout"

	"fyne.io/fyne/v2/widget"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var app = fyneapp.New()

var diskIcon widget.Icon

type Disk struct {
	ID         string
	Name       string
	Size       uint64
	Type       string
	Partitions *orderedmap.OrderedMap[string, Partition]
}

type Partition struct {
	ID         string
	Type       string
	Name       string
	Size       uint64
	Device     string
	Partitions *orderedmap.OrderedMap[string, Partition]
	MountPoint string
}
type Data struct {
	Disk      Disk
	Partition Partition
}

func parseSize(size uint64) string {
	gigabyte := size / 1e+9
	if gigabyte > 0 {
		return fmt.Sprintf("%dGB", gigabyte)
	}
	megabyte := size / 1000000
	if megabyte > 0 {
		return fmt.Sprintf("%dMB", megabyte)
	}
	kilobyte := size / 1000
	if kilobyte > 0 {
		return fmt.Sprintf("%dKB", kilobyte)
	}
	return fmt.Sprintf("%dB", size)
}

var Disks = orderedmap.New[string, Disk]()
var Volumes = orderedmap.New[string, Partition]()
var VolumeType = float64(0)

func LoadData(c *fyne.Container) {
	button := c.Objects[0]
	c.RemoveAll()
	c.Add(button)
	switch runtime.GOOS {
	case "darwin":
		{
			Disks = DarwinGetPartitions()
		}
	case "windows":
		{
			Disks, _ = WindowsGetDisks()
		}
	}
	for pair := Disks.Oldest(); pair != nil; pair = pair.Next() {
		disk := pair.Value
		diskName := widget.NewRichTextFromMarkdown(fmt.Sprintf("# %s", disk.Name))
		diskSize := widget.NewRichTextFromMarkdown(fmt.Sprintf("# %s", parseSize(disk.Size)))
		co := container.NewHBox(&diskIcon, diskName, layout.NewSpacer(), diskSize)
		diskContainer := container.NewVBox(co)
		for pair := disk.Partitions.Oldest(); pair != nil; pair = pair.Next() {
			partition := pair.Value
			partitionName := widget.NewRichTextFromMarkdown(fmt.Sprintf("## %s", partition.Name))
			partitionSize := widget.NewRichTextFromMarkdown(fmt.Sprintf("## %s", parseSize(partition.Size)))
			mount := widget.NewButton("Mount", func() {
				if partition.MountPoint != "" {
					switch runtime.GOOS {
					case "darwin":
						{
							DarwinOpenFolder(partition.MountPoint)
						}
					case "windows":
						{
							WindowsOpenFolder(partition.MountPoint)
						}
					}
				} else {
					switch runtime.GOOS {
					case "darwin":
						{
							success := DarwinMountPartition(partition)
							if success {
								LoadData(c)
							}
						}
					case "windows":
						{
							success := WindowsMountVolume(partition.ID)
							if success {
								LoadData(c)
							}
						}
					}
				}
			})
			if partition.MountPoint != "" {
				mount.Importance = widget.LowImportance
				mount.SetText(partition.MountPoint)
			} else {
				mount.Importance = widget.HighImportance
			}
			partitionContainer := container.New(layout.NewHBoxLayout(), partitionName, layout.NewSpacer(), partitionSize, mount)
			diskContainer.Add(partitionContainer)
		}
		card := widget.NewCard("", "", diskContainer)
		c.Add(card)
	}
}

func main() {
	_, _ = os.ReadFile("C:\\Users\\Amirb\\Downloads\\transparent-qartion.png")
	app.SetIcon(fyne.NewStaticResource("logo", Logo))
	w := app.NewWindow("Qartion")
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		card := widget.NewCard("Unsupported Platform", "Qartion does not support the platform you are using.", widget.NewButton("Exit", func() {
			w.Close()
		}))
		w.SetContent(card)
	} else {
		c := container.NewVBox()
		buttons := container.NewHBox(widget.NewCard("", "", container.New(layout.NewGridLayout(2), widget.NewButton("Refresh", func() {
			LoadData(c)
		}), widget.NewButton("Settings", func() {
			LaunchSettings(app)
		}))))
		i, _ := getIconTheme()

		var di []byte
		switch i {
		case "windows":
			di = WindowsDiskIcon
		case "darwin":
			di = DarwinDiskIcon
		}
		diskIcon = *widget.NewIcon(fyne.NewStaticResource("disk-icon", di))
		c.Add(buttons)
		LoadData(c)
		w.SetContent(c)
	}

	w.ShowAndRun()
}
