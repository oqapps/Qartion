package main

import (
	"fmt"
	"os/exec"

	"path/filepath"

	"github.com/getlantern/elevate"
	"github.com/google/uuid"
	"github.com/oq-x/go-plist"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func getDiskPartitions() (*orderedmap.OrderedMap[string, Disk], error) {
	cmd := exec.Command("diskutil", "list", "-plist")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute diskutil command: %s", err)
	}
	var data = plist.OrderedDict{}
	_, err = plist.Unmarshal(output, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse diskutil output: %s", err)
	}

	disks := orderedmap.New[string, Disk]()
	all := data.Values[1].([]interface{})
	for _, d := range all {
		data := d.(plist.OrderedDict)
		if len(data.Keys) == 5 {
			// disk
			id := uuid.NewString()
			partitions := orderedmap.New[string, Partition]()
			partitionIDs := make([]string, 0)
			pars := data.Values[3].([]interface{})
			for _, p := range pars {
				par := p.(plist.OrderedDict)
				switch len(par.Keys) {
				case 6:
					{
						id := par.Values[2].(string)
						mountPoint := ""
						if filepath.IsAbs(id) {
							mountPoint = id
							id = par.Values[5].(string)
						}
						partitions.Set(par.Values[2].(string), Partition{
							Name:       par.Values[4].(string),
							Device:     par.Values[1].(string),
							ID:         par.Values[2].(string),
							Size:       par.Values[3].(uint64),
							MountPoint: mountPoint,
						})
						partitionIDs = append(partitionIDs, par.Values[2].(string))
					}
				case 7:
					{
						partitions.Set(par.Values[2].(string), Partition{
							Name:       par.Values[5].(string),
							Device:     par.Values[1].(string),
							ID:         par.Values[2].(string),
							Size:       par.Values[4].(uint64),
							MountPoint: par.Values[3].(string),
						})
						partitionIDs = append(partitionIDs, par.Values[2].(string))
					}
				}
			}
			info, _ := GetInfo(data.Values[1].(string))
			disks.Set(info["MediaName"].(string), Disk{
				ID:           id,
				Name:         info["MediaName"].(string),
				Size:         data.Values[4].(uint64),
				Partitions:   *partitions,
				PartitionIDs: partitionIDs,
			})
		} else if len(data.Keys) == 7 {
			// container
			info, _ := GetInfo(data.Values[3].(string))
			disk, e := disks.Get(info["MediaName"].(string))
			if e {
				pars := data.Values[1].([]interface{})
				for _, i := range pars {
					par := i.(plist.OrderedDict)
					switch len(par.Keys) {
					case 8:
						{
							if par.Keys[3] == "MountPoint" {
								disk.Partitions.Set(par.Values[2].(string), Partition{
									Name:       par.Values[6].(string),
									Device:     par.Values[1].(string),
									ID:         par.Values[2].(string),
									Size:       par.Values[5].(uint64),
									MountPoint: par.Values[3].(string),
								})
								disk.PartitionIDs = append(disk.PartitionIDs, par.Values[2].(string))
							}
						}
					case 9:
						{
							disk.Partitions.Set(par.Values[2].(string), Partition{
								Name:       par.Values[7].(string),
								Device:     par.Values[1].(string),
								ID:         par.Values[2].(string),
								Size:       par.Values[6].(uint64),
								MountPoint: par.Values[3].(string),
							})
							disk.PartitionIDs = append(disk.PartitionIDs, par.Values[2].(string))
						}
					}
				}
			}
		} else {
		}
	}
	return disks, nil
}

func GetInfo(name string) (map[string]interface{}, error) {
	cmd := exec.Command("diskutil", "info", "-plist", name)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute diskutil command: %s", err)
	}
	var data = plist.OrderedDict{}
	d := make(map[string]interface{})
	_, err = plist.Unmarshal(output, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse diskutil output: %s", err)
	}
	for i, key := range data.Keys {
		d[key] = data.Values[i]
	}
	return d, nil
}

func LoopPartitions(partitions orderedmap.OrderedMap[string, Partition]) {
	for pair := partitions.Oldest(); pair != nil; pair = pair.Next() {
		partition := pair.Value
		Entries[partition.ID] = Data{Partition: partition}
		for pair := partition.Partitions.Oldest(); pair != nil; pair = pair.Next() {
			LoopPartitions(pair.Value.Partitions)
		}
	}
}

func DarwinGetPartitions() (Disks *orderedmap.OrderedMap[string, Disk], DiskIDs []string) {
	disks, err := getDiskPartitions()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	Disks = disks

	for pair := disks.Oldest(); pair != nil; pair = pair.Next() {
		DiskIDs = append(DiskIDs, pair.Value.ID)
		Entries[pair.Value.ID] = Data{Disk: pair.Value}
		LoopPartitions(pair.Value.Partitions)
	}
	return
}

func DarwinOpenFolder(path string) {
	cmd := exec.Command("open", path)
	cmd.Output()
}

func DarwinMountPartition(partition Partition) bool {
	cmd := elevate.Command("sudo", "diskutil", "mount", partition.Device)
	_, e := cmd.Output()
	if e != nil {
		return false
	}
	info, e := GetInfo(partition.ID)
	if e != nil {
		return false
	}
	exec.Command("open", info["MountPoint"].(string)).Output()
	return true
}
