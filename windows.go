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

func WindowsMountVolume(volumeId string) bool {
	letter := windowsGenerateLetter()
	cmd := elevate.Command("mountvol", letter, volumeId)
	err := cmd.Run()
	return err == nil
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

func windowsParseListDisk(output string) map[string]map[string]interface{} {
	ls := strings.Split(output, "\n")
	disks := make(map[string]map[string]interface{}, 0)
	for i, l := range ls {
		if i == 0 {
			continue
		}
		vals := strings.Split(l, "  ")
		values := make([]string, 0)
		disk := make(map[string]interface{})
		if len(vals) < 3 {
			continue
		}
		for _, r := range vals {
			if strings.TrimSpace(r) != "" {
				values = append(values, strings.TrimSpace(r))
			}
		}
		disk["Index"] = values[0]
		disk["Model"] = values[1]
		disk["Size"] = values[2]
		disk["Volumes"] = orderedmap.New[string, Partition]()
		disks[strings.TrimSpace(values[0])] = disk
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
	pdata, _ := windowsPowershellCommand("Get-CimInstance -Class Win32_Volume | Select-Object DriveLetter, DeviceID, Label, FileSystem, Capacity, FreeSpace")
	ddata, _ := windowsCommand("wmic diskdrive get Model, Size, Index")
	ddisks := windowsParseListDisk(ddata)
	disks := orderedmap.New[string, Disk]()
	volus := strings.Split(strings.TrimSpace(pdata), "\r\n\r\n")
	dnums := windowsGetDiskNumbers()
	for _, vol := range volus {
		data := make(map[string]string)
		for _, l := range strings.Split(vol, "\n") {
			fsp := make([]string, 0)
			for _, d := range strings.Split(strings.TrimSpace(l), "") {
				if strings.TrimSpace(d) != "" {
					fsp = append(fsp, strings.TrimSpace(d))
				}
			}
			l = strings.Join(fsp, "")
			if fsp[len(fsp)-1] == ":" {
				str := strings.Split(l, ":")[0]
				if str == "DriveLetter" {
					l = l[:len(l)-1]
				}
			}
			l = strings.TrimSpace(l)
			sp := strings.Split(l, ":")
			if len(sp) == 2 {
				data[strings.TrimSpace(sp[0])] = strings.TrimSpace(sp[1])
			}
		}
		disk := ddisks[dnums[data["DeviceID"]]]
		size, _ := strconv.Atoi(data["Capacity"])
		mountPoint := ""
		if data["DriveLetter"] != "" {
			mountPoint = fmt.Sprintf("%s:\\", data["DriveLetter"])
		}
		partition := Partition{
			ID:         data["DeviceID"],
			Name:       data["Label"],
			Size:       uint64(size),
			MountPoint: mountPoint,
		}
		vols := disk["Volumes"].(*orderedmap.OrderedMap[string, Partition])
		vols.Set(data["DeviceID"], partition)
	}
	for _, disk := range ddisks {
		size, _ := strconv.Atoi(disk["Size"].(string))
		d := Disk{
			ID:   disk["Index"].(string),
			Name: disk["Model"].(string),
			Size: uint64(size),
		}
		if disk["Volumes"] != nil {
			d.Partitions = disk["Volumes"].(*orderedmap.OrderedMap[string, Partition])
		}
		disks.Set(disk["Index"].(string), d)
	}
	return disks, nil
}

func windowsPowershellCommand(command string) (string, error) {
	cmd := exec.Command("powershell.exe", "-Command", command)

	i, err := cmd.Output()
	if err != nil {
		fmt.Println("Command execution failed:", err)
		return "", err
	}
	return string(i), nil
}
