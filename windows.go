package main

import (
	"fmt"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/getlantern/elevate"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func WindowsOpenFolder(path string) {
	cmd := exec.Command("explorer", path)
	cmd.Run()
}

func WindowsMountVolume(volumeId string) (bool, string) {
	letter := windowsGenerateLetter()
	cmd := elevate.Command("mountvol", letter, volumeId)
	err := cmd.Run()
	mountpoint := fmt.Sprintf("%s:\\", letter)
	WindowsOpenFolder(mountpoint)
	return err == nil, mountpoint
}

func windowsGenerateLetter() string {
	letter := windowsRandomLetter()
	for pair := Volumes.Oldest(); pair != nil; pair = pair.Next() {
		if pair.Value.MountPoint == letter {
			letter = windowsRandomLetter()
		}
	}
	return letter
}

func windowsRandomLetter() string {
	rand.Seed(time.Now().UnixNano())
	randomNum := rand.Intn(26) + 65
	return fmt.Sprintf("%c:\\", rune(randomNum))
}

func windowsCommand(command string) (string, error) {
	cmd := exec.Command("cmd.exe", "/C", command)
	output, err := cmd.Output()
	return string(output), err
}

func windowsParseListDisk(output string) *orderedmap.OrderedMap[string, Disk] {
	ls := strings.Split(output, "\n")
	disks := orderedmap.New[string, Disk]()
	for i, l := range ls {
		if i == 0 {
			continue
		}
		vals := strings.Split(l, "  ")
		values := make([]string, 0)
		if len(vals) < 3 {
			continue
		}
		for _, r := range vals {
			if strings.TrimSpace(r) != "" {
				values = append(values, strings.TrimSpace(r))
			}
		}
		size, _ := strconv.Atoi(values[2])
		id := strings.TrimSpace(values[0])
		disks.Set(id, Disk{
			Name:       values[1],
			ID:         id,
			Partitions: orderedmap.New[string, Partition](),
			Size:       uint64(size),
		})
	}
	return disks
}

func windowsGetDiskNumbers() map[string]string {
	d, _ := windowsPowershellCommand("Get-Partition | Select-Object DiskNumber, AccessPaths")
	data := make(map[string]string)
	for i, l := range strings.Split(strings.TrimSpace(d), "\n") {
		if i < 2 {
			continue
		}
		l = strings.TrimSpace(l)
		sp := strings.Split(l, " ")
		if len(sp) < 2 {
			continue
		}
		index := len(sp) - 1
		if len(sp) > 2 {
			data[sp[index][:len(sp[index])-1]] = sp[0]
		} else {
			data[sp[index][1:len(sp[index])-1]] = sp[0]
		}
	}
	return data
}

func WindowsGetDisks() (*orderedmap.OrderedMap[string, Disk], error) {
	pdata, _ := windowsCommand("wmic volume get DeviceID, Capacity, Label, DriveLetter")
	ddata, _ := windowsCommand("wmic diskdrive get Model, Size, Index")
	disks := windowsParseListDisk(ddata)
	volus := strings.Split(strings.TrimSpace(pdata), "\n")
	dnums := windowsGetDiskNumbers()
	for i, vol := range volus {
		if i == 0 {
			continue
		}
		data := make(map[string]string)
		vals := make([]string, 0)
		for _, e := range strings.Split(vol, "  ") {
			tr := strings.TrimSpace(e)
			if tr != "" {
				vals = append(vals, tr)
			}
		}
		data["Capacity"] = vals[0]
		data["DeviceID"] = vals[1]
		switch len(vals) {
		case 3:
			{
				data["Label"] = vals[2]
			}
		case 4:
			{
				data["DriveLetter"] = vals[2]
				data["Label"] = vals[3]
			}
		}
		disk, _ := disks.Get(dnums[data["DeviceID"]])
		size, _ := strconv.Atoi(data["Capacity"])
		mountPoint := ""
		if data["DriveLetter"] != "" {
			mountPoint = fmt.Sprintf("%s\\", data["DriveLetter"])
		}
		partition := Partition{
			ID:         data["DeviceID"],
			Name:       data["Label"],
			Size:       uint64(size),
			MountPoint: mountPoint,
		}
		disk.Partitions.Set(data["DeviceID"], partition)
	}
	return disks, nil
}

func windowsPowershellCommand(command string) (string, error) {
	cmd := exec.Command("powershell.exe", "/C", command)

	i, err := cmd.Output()
	if err != nil {
		fmt.Println("Command execution failed:", err)
		return "", err
	}
	return string(i), nil
}
