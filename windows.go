package main

import (
	"fmt"
	"math/rand"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var WindowsVolumeTypes = "Partition,Simple,Mirror,Stripe,RAID-5,Unknown,No Fs,Removable"
var WindowsFileSystems = "FAT32,NTFS,exFAT,UDF,ReFS,Fat,Raw,CDFS,DFS"
var WindowsHealthTypes = "Healthy,Healthy(System),Healthy(Active),Healthy(Boot),Failed,Failed(Errors),Failed(Offline),Failed(Validation),Failed(Other),Formatting,Resynching"
var WindowsInfoTypes = "Hidden,System,Active,Boot,Crash Dump,Page File,Primary Partition,Logical Drive,No Drive Letter,Read-only"

func WindowsGetPartitions() *orderedmap.OrderedMap[string, Disk] {
	disks, err := windowsGetDiskPartitions()
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	return disks
}

func WindowsOpenFolder(path string) {
	cmd := exec.Command("explorer", path)
	cmd.Run()
}

func WindowsMountPartition(partition Partition) bool {
	_, err := windowsDiskpartCommand(fmt.Sprintf("sel vol %s", partition.ID), "assign")
	return err == nil
}

func WindowsMountVolume(volumeId string) bool {
	letter := windowsGenerateLetter()
	cmd := exec.Command("mountvol", letter, volumeId)
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

func windowsParseSize(size string) uint64 {
	sizeNumber, _ := strconv.Atoi(strings.Split(size, " ")[0])
	if strings.HasSuffix(size, "GB") {
		return uint64(sizeNumber * 1e+9)
	} else if strings.HasSuffix(size, "MB") {
		return uint64(sizeNumber * 1000000)
	} else if strings.HasSuffix(size, "KB") {
		return uint64(sizeNumber * 1000)
	}
	return uint64(sizeNumber)
}

func windowsGetDiskPartitions() (*orderedmap.OrderedMap[string, Disk], error) {
	d, _ := windowsDiskpartCommand("list disk")
	ds := windowsParseListDisk(d)
	disks := orderedmap.New[string, Disk]()
	for i, d := range ds {
		detailCommand, _ := windowsDiskpartCommand(fmt.Sprintf("sel disk %s", i), "detail disk")
		detail := windowsParseDetailDisk(i, detailCommand)
		s := strings.Split(d["size"], " ")[0]
		size, _ := strconv.Atoi(s)
		partitions := orderedmap.New[string, Partition]()
		for _, p := range detail["Volumes"].([]map[string]interface{}) {
			mountPoint := ""
			if p["MountPoint"] != nil {
				mountPoint = p["MountPoint"].(string)
			}
			partitions.Set(p["ID"].(string), Partition{
				ID:         p["ID"].(string),
				Size:       p["Size"].(uint64),
				Name:       p["Name"].(string),
				MountPoint: mountPoint,
			})
		}
		disks.Set(detail["Disk ID"].(string), Disk{
			ID:         detail["Disk ID"].(string),
			Name:       detail["Name"].(string),
			Type:       detail["Type"].(string),
			Size:       uint64(size),
			Partitions: *partitions,
		})
	}
	return disks, nil
}

func windowsDiskpartCommand(commands ...string) (string, error) {
	c := ""
	for in, i := range commands {
		if in == len(commands)-1 {
			c += fmt.Sprintf("echo %s", i)
		} else {
			c += fmt.Sprintf("echo %s & ", i)
		}
	}
	cmd := exec.Command("cmd.exe", "/C", fmt.Sprintf("(%s) | diskpart", c))

	i, err := cmd.Output()
	if err != nil {
		fmt.Println("Command execution failed:", err)
		return "", err
	}
	return string(i), nil
}

func WindowsGetVolumes() *orderedmap.OrderedMap[string, Partition] {
	volumes := orderedmap.New[string, Partition]()
	vols := windowsListVolumeGUIDs()
	for _, i := range vols {
		if i["FileSystem"] == "" && i["Label"] == "" && i["Size"] == "" {
			continue
		}
		size, _ := strconv.Atoi(i["Capacity"])
		mountPoint := ""
		if i["DriveLetter"] != "" {
			mountPoint = fmt.Sprintf("%s:\\", i["DriveLetter"])
		}
		volumes.Set(i["DeviceID"], Partition{
			ID:         i["DeviceID"],
			Size:       uint64(size),
			MountPoint: mountPoint,
			Name:       i["Label"],
		})
	}
	return volumes
}

func windowsListVolumeGUIDs() []map[string]string {
	data, _ := windowsPowershellCommand("GWMI -namespace root\\cimv2 -class win32_volume | FL -property Label,DriveLetter,DeviceID,SystemVolume,Capacity,Freespace,FileSystem")
	vols := strings.Split(data, "\r\n\r\n")
	volumes := make([]map[string]string, 0)
	for _, vol := range vols {
		vol = strings.TrimSpace(vol)
		if vol == "" {
			continue
		}
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
		volumes = append(volumes, data)
	}
	return volumes
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

func windowsParseDetailDisk(index string, output string) map[string]interface{} {
	diskInfo := make(map[string]interface{})
	volumes := make([]map[string]interface{}, 0)

	lines := strings.Split(output, "\n")
	keyValuePattern := regexp.MustCompile(`([A-Za-z\s]+)\s*:\s*(.*)`)

	for i, line := range lines {
		if matches := keyValuePattern.FindStringSubmatch(line); len(matches) == 3 {
			key := strings.TrimSpace(matches[1])
			value := strings.TrimSpace(matches[2])

			diskInfo[key] = value
			if key == "Disk ID" {
				sline := lines[i-1]
				diskInfo["Name"] = sline
			}
		}

		if i > 25 && i < 30 && !strings.HasPrefix(line, "  Volume ###") && !strings.HasPrefix(line, "  ----------") && strings.TrimSpace(line) != "" {
			line = strings.TrimSpace(line)
			volume := make(map[string]interface{})
			for _, d := range strings.Split(line, "  ") {
				if strings.TrimSpace(d) == "" {
					continue
				}
				d = strings.TrimSpace(d)
				if strings.HasPrefix(d, "Volume") {
					volume["ID"] = strings.TrimSpace(strings.Split(d, " ")[1])
				}
				if len(strings.Split(d, "")) == 1 {
					volume["MountPoint"] = fmt.Sprintf("%s:\\", d)
				}
				if !strings.Contains(WindowsFileSystems, d) && !strings.Contains(WindowsHealthTypes, d) && !strings.Contains(WindowsInfoTypes, d) && !strings.Contains(WindowsVolumeTypes, d) {
					if strings.HasSuffix(d, "B") {
						volume["Size"] = windowsParseSize(d)
					} else {
						volume["Name"] = d
					}
				}
			}
			volumes = append(volumes, volume)
		}
	}

	diskInfo["Volumes"] = volumes
	return diskInfo
}

func windowsParseListDisk(output string) map[string]map[string]string {
	diskInfo := make(map[string]map[string]string)
	diskPattern := regexp.MustCompile(`Disk\s+(\d+)\s+([A-Za-z]+)\s+(\d+\s+[\w]+)\s+(\d+\s+[\w]+)`)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		matches := diskPattern.FindStringSubmatch(line)
		if len(matches) > 0 {
			diskNumber := matches[1]
			status := matches[2]
			size := matches[3]
			free := matches[4]

			diskDetails := make(map[string]string)
			diskDetails["status"] = status
			diskDetails["size"] = size
			diskDetails["free"] = free

			diskInfo[diskNumber] = diskDetails
		}
	}
	return diskInfo
}
