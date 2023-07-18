package main

import (
	_ "embed"
	"fmt"
	"os"
	"runtime"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var Disks = orderedmap.New[string, Disk]()
var Volumes = orderedmap.New[string, Partition]()
var VolumeType = float64(0)

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
	terabyte := size / 1e+12
	if terabyte > 0 {
		return fmt.Sprintf("%dTB", terabyte)
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

func LoadData(l *widgets.QGridLayout) {
	for l.Count() > 0 {
		layoutItem := l.TakeAt(0)
		if layoutItem != nil {
			layoutItem.Widget().SetParent(nil)
			layoutItem.Widget().DestroyQWidget()
		}
	}
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
	var index = 0
	for pair := Disks.Oldest(); pair != nil; pair = pair.Next() {
		disk := pair.Value

		var (
			card          = widgets.NewQGroupBox2("", nil)
			diskFont      = gui.NewQFont()
			partitionFont = gui.NewQFont()
			diskName      = widgets.NewQLabel2(disk.Name, nil, 0)
			diskSize      = widgets.NewQLabel2(parseSize(disk.Size), nil, 0)
		)
		diskFont.SetPointSize(25)
		diskName.SetFont(diskFont)
		diskSize.SetFont(diskFont)

		partitionFont.SetPointSize(15)

		var layout = widgets.NewQGridLayout2()
		layout.AddWidget2(diskName, 0, 0, 0)
		layout.AddWidget2(diskSize, 0, 2, 0)

		var pindex = 1
		for pair := disk.Partitions.Oldest(); pair != nil; pair = pair.Next() {
			partition := pair.Value
			var (
				partitionName = widgets.NewQLabel2(partition.Name, nil, 0)
				partitionSize = widgets.NewQLabel2(parseSize(partition.Size), nil, 0)
				mountButton   = widgets.NewQPushButton2("Mount", nil)
			)
			partitionName.SetFont(partitionFont)
			partitionSize.SetFont(partitionFont)

			layout.AddWidget2(partitionName, pindex, 0, 0)
			layout.AddWidget2(partitionSize, pindex, 1, 0)
			layout.AddWidget3(mountButton, pindex, 2, 1, 2, 0)

			mountButton.ConnectClicked(func(bool) {
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
							success, partition := DarwinMountPartition(partition)
							if success {
								mountButton.SetText(partition.MountPoint)
							}
						}
					case "windows":
						{
							success, mountpoint := WindowsMountVolume(partition.ID)
							if success {
								mountButton.SetText(mountpoint)
							}
						}
					}
				}
			})

			if partition.MountPoint != "" {
				mountButton.SetText(partition.MountPoint)
			}
			pindex += 1
		}
		card.SetLayout(layout)
		l.AddWidget2(card, index, 0, 0)
		index += 1
	}
}

func main() {
	app := widgets.NewQApplication(len(os.Args), os.Args)
	core.QCoreApplication_SetOrganizationName("oqDev")
	core.QCoreApplication_SetApplicationName("Qartion")
	core.QCoreApplication_SetApplicationVersion("1.3.0")
	window := widgets.NewQMainWindow(nil, 0)
	wsize := window.Size()
	window.SetFixedSize2(wsize.Width(), wsize.Height())

	menuBar := window.MenuBar()
	menu := menuBar.AddMenu2("App")
	reloadButton := menu.AddAction("Reload")
	reloadShortcut := gui.NewQKeySequence2("Ctrl+R", gui.QKeySequence__NativeText)
	reloadButton.SetShortcut(reloadShortcut)

	var layout = widgets.NewQGridLayout2()
	var centralWidget = widgets.NewQWidget(window, 0)
	centralWidget.SetLayout(layout)
	window.SetCentralWidget(centralWidget)

	reloadButton.ConnectTriggered(func(checked bool) {
		LoadData(layout)
	})

	LoadData(layout)

	window.SetWindowTitle("Qartion")
	window.Show()
	app.Exec()
}
