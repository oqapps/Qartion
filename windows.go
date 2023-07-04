package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

var WindowsVolumeTypes = "Partition,Simple,Mirror,Stripe,RAID-5,Unknown,No Fs"
var WindowsFileSystems = "FAT32,NTFS,exFAT,UDF,ReFS,Fat,Raw,CDFS,DFS"
var WindowsHealthTypes = "Healthy,Healthy(System),Healthy(Active),Healthy(Boot),Failed,Failed(Errors),Failed(Offline),Failed(Validation),Failed(Other),Formatting,Resynching"
var WindowsInfoTypes = "Hidden,System,Active,Boot,Crash Dump,Page File,Primary Partition,Logical Drive,No Drive Letter,Read-only"

func WindowsGetPartitions() (Disks *orderedmap.OrderedMap[string, Disk], DiskIDs []string) {
	disks, err := windowsGetDiskPartitions()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	Disks = disks
	for pair := disks.Oldest(); pair != nil; pair = pair.Next() {
		DiskIDs = append(DiskIDs, pair.Value.ID)
		Entries[pair.Value.ID] = Data{Disk: pair.Value}
		loopPartitions(pair.Value.Partitions)
	}
	return
}

func WindowsOpenFolder(path string) {
	cmd := exec.Command("explorer", path)
	cmd.Run()
}

func WindowsMountPartition(partition Partition) bool {
	_, err := windowsDiskpartCommand(fmt.Sprintf("sel vol %s", partition.ID), "assign")
	return err != nil
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
		partitionIds := make([]string, 0)
		for _, p := range detail["Volumes"].([]map[string]interface{}) {
			partitionIds = append(partitionIds, p["ID"].(string))
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
			ID:           detail["Disk ID"].(string),
			Name:         detail["Name"].(string),
			Size:         uint64(size),
			Partitions:   *partitions,
			PartitionIDs: partitionIds,
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
	diskPattern := regexp.MustCompile(`Disk\s+(\d+)\s+([A-Za-z]+)\s+(\d+\s+[\w]+)\s+(\d+\s+[\w]+)\s+([*])`)
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
