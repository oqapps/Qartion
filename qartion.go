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
	Partitions orderedmap.OrderedMap[string, Partition]
}

type Partition struct {
	ID         string
	Type       string
	Name       string
	Size       uint64
	Device     string
	Partitions orderedmap.OrderedMap[string, Partition]
	MountPoint string
}
type Data struct {
	Disk      Disk
	Partition Partition
}

func parseSize(size uint64, giga bool) string {
	if runtime.GOOS == "windows" && giga {
		return fmt.Sprintf("%dGB", size)
	}
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
			volumeType := FetchSetting("volumeMode")
			if volumeType != nil {
				VolumeType = volumeType.(float64)
			}
			switch VolumeType {
			case 1:
				{
					Volumes = WindowsGetVolumes()
				}
			default:
				{
					Disks = WindowsGetPartitions()
				}
			}
		}
	}
	if Disks.Len() > 0 {
		for pair := Disks.Oldest(); pair != nil; pair = pair.Next() {
			disk := pair.Value
			diskName := widget.NewRichTextFromMarkdown(fmt.Sprintf("# %s", disk.Name))
			diskSize := widget.NewRichTextFromMarkdown(fmt.Sprintf("# %s", parseSize(disk.Size, true)))
			co := container.NewHBox(&diskIcon, diskName, layout.NewSpacer(), diskSize)
			diskContainer := container.NewVBox(co)
			for pair := disk.Partitions.Oldest(); pair != nil; pair = pair.Next() {
				partition := pair.Value
				partitionName := widget.NewRichTextFromMarkdown(fmt.Sprintf("## %s", partition.Name))
				partitionSize := widget.NewRichTextFromMarkdown(fmt.Sprintf("## %s", parseSize(partition.Size, false)))
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
								if disk.Type == "USB" {
									button := widget.NewButton("Settings", func() {
										LaunchSettings(app)
									})
									box := container.NewHBox(layout.NewSpacer(), button, layout.NewSpacer())
									card := widget.NewCard("Unsupported Device", "Linked mode does not support mounting external partitions. Please use Standalone mode.", box)
									CreatePopup(app, "Unsupported Device", card)
									return
								}
								success := WindowsMountPartition(partition)
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
	} else if Volumes.Len() > 0 {
		for pair := Volumes.Oldest(); pair != nil; pair = pair.Next() {
			volume := pair.Value
			volumeName := widget.NewRichTextFromMarkdown(fmt.Sprintf("# %s", volume.Name))
			volumeSize := widget.NewRichTextFromMarkdown(fmt.Sprintf("# %s", parseSize(volume.Size, false)))

			mount := widget.NewButton("Mount", func() {
				if volume.MountPoint != "" {
					switch runtime.GOOS {
					case "darwin":
						{
							DarwinOpenFolder(volume.MountPoint)
						}
					case "windows":
						{
							WindowsOpenFolder(volume.MountPoint)
						}
					}
				} else {
					switch runtime.GOOS {
					case "windows":
						{
							success := WindowsMountVolume(volume.ID)
							if success {
								LoadData(c)
							}
						}
					}
				}
			})
			if volume.MountPoint != "" {
				mount.Importance = widget.LowImportance
				mount.SetText(volume.MountPoint)
			} else {
				mount.Importance = widget.HighImportance
			}
			co := container.NewHBox(&diskIcon, volumeName, layout.NewSpacer(), volumeSize, mount)
			volumeContainer := container.NewVBox(co)
			card := widget.NewCard("", "", volumeContainer)
			c.Add(card)
		}
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
