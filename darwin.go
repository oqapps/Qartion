package main

import (
	"fmt"
	"os/exec"
	"strings"

	"path/filepath"

	"github.com/getlantern/elevate"
	"github.com/google/uuid"
	"github.com/oq-x/go-plist"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func darwinGetDiskPartitions() (*orderedmap.OrderedMap[string, Disk], error) {
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
			pars := data.Values[3].([]interface{})
			for _, p := range pars {
				par := p.(plist.OrderedDict)
				switch len(par.Keys) {
				case 6:
					{
						if par.Values[0] == "Microsoft Basic Data" {
							id := par.Values[2].(string)
							namesp := strings.Split(par.Values[3].(string), "/")
							name := namesp[len(namesp)-1]
							partitions.Set(id, Partition{
								Name:       name,
								Device:     par.Values[1].(string),
								ID:         id,
								Size:       par.Values[4].(uint64),
								MountPoint: par.Values[3].(string),
							})
						} else {
							id := par.Values[2].(string)
							mountPoint := ""
							if filepath.IsAbs(id) {
								mountPoint = id
								id = par.Values[5].(string)
							}
							partitions.Set(id, Partition{
								Name:       par.Values[4].(string),
								Device:     par.Values[1].(string),
								ID:         id,
								Size:       par.Values[3].(uint64),
								MountPoint: mountPoint,
							})
						}
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
					}
				}
			}
			info, _ := GetInfo(data.Values[1].(string))
			disks.Set(info["MediaName"].(string), Disk{
				ID:         id,
				Name:       info["MediaName"].(string),
				Size:       data.Values[4].(uint64),
				Partitions: partitions,
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
						}
					}
				}
			}
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

func DarwinGetPartitions() *orderedmap.OrderedMap[string, Disk] {
	disks, err := darwinGetDiskPartitions()
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	return disks
}

func DarwinOpenFolder(path string) {
	cmd := exec.Command("open", path)
	cmd.Run()
}

func DarwinMountPartition(partition Partition) (bool, Partition) {
	cmd := elevate.Command("diskutil", "mount", partition.Device)
	_, e := cmd.Output()
	if e != nil {
		return false, partition
	}
	info, e := GetInfo(partition.ID)
	if e != nil {
		return false, partition
	}
	e = exec.Command("open", info["MountPoint"].(string)).Run()
	partition.MountPoint = info["MountPoint"].(string)
	return e == nil, partition
}
