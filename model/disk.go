/*@Author: LinkLeong link@icewhale.com
 *@Date: 2022-07-13 10:43:45
 *@LastEditors: LinkLeong
 *@LastEditTime: 2022-08-03 14:45:35
 *@FilePath: /CasaOS/model/disk.go
 *@Description:
 *@Website: https://www.casaos.io
 *Copyright (c) 2022 by icewhale, All Rights Reserved.
 */
package model

import "encoding/json"

type LSBLKModel struct {
	Name        string       `json:"name"`
	FsType      string       `json:"fstype"`
	Size        uint64       `json:"size"`
	FSSize      json.Number  `json:"fssize"`
	Path        string       `json:"path"`
	Model       string       `json:"model"` // 设备标识符
	RM          bool         `json:"rm"`    // 是否为可移动设备
	RO          bool         `json:"ro"`    // 是否为只读设备
	State       string       `json:"state"`
	PhySec      int          `json:"phy-sec"` // 物理扇区大小
	Type        string       `json:"type"`
	Vendor      string       `json:"vendor"`  // 供应商
	Rev         string       `json:"rev"`     // 修订版本
	FSAvail     json.Number  `json:"fsavail"` // 可用空间
	FSUse       string       `json:"fsuse%"`  // 已用百分比
	MountPoint  string       `json:"mountpoint"`
	Format      string       `json:"format"`
	Health      string       `json:"health"`
	HotPlug     bool         `json:"hotplug"`
	UUID        string       `json:"uuid"`
	PTUUID      string       `json:"ptuuid"`
	PartUUID    string       `json:"partuuid"`
	FSUsed      json.Number  `json:"fsused"`
	Temperature int          `json:"temperature"`
	Tran        string       `json:"tran"`
	MinIO       uint64       `json:"min-io"`
	UsedPercent float64      `json:"used_percent"`
	Serial      string       `json:"serial"`
	Children    []LSBLKModel `json:"children"`
	SubSystems  string       `json:"subsystems"`
	Label       string       `json:"label"`
	// 详情特有
	StartSector uint64 `json:"start_sector,omitempty"`
	Rota        bool   `json:"rota"` // true(hhd) false(ssd)
	DiskType    string `json:"disk_type"`
	EndSector   uint64 `json:"end_sector,omitempty"`
}

type Drive struct {
	Name           string `json:"name"`
	Size           uint64 `json:"size"`
	Model          string `json:"model"`
	Health         string `json:"health"`
	Temperature    int    `json:"temperature"`
	DiskType       string `json:"disk_type"`
	NeedFormat     bool   `json:"need_format"`
	Serial         string `json:"serial"`
	Path           string `json:"path"`
	ChildrenNumber int    `json:"children_number"`
}

type USBDriveStatus struct {
	Name     string        `json:"name"`
	Size     uint64        `json:"size"`
	Model    string        `json:"model"`
	Avail    uint64        `json:"avail"`
	Children []USBChildren `json:"children"`
}
type USBChildren struct {
	Name       string `json:"name"`
	Size       uint64 `json:"size"`
	Avail      uint64 `json:"avail"`
	MountPoint string `json:"mount_point"`
}

type Storage struct {
	UUID        string `json:"uuid"`
	MountPoint  string `json:"mount_point"`
	Size        string `json:"size"`
	Avail       string `json:"avail"` // 可用空间
	Type        string `json:"type"`
	Path        string `json:"path"`
	DriveName   string `json:"drive_name"`
	Label       string `json:"label"`
	PersistedIn string `json:"persisted_in"` // none, fstab, casaos
}
type Storages struct {
	DiskName string    `json:"disk_name"`
	Size     uint64    `json:"size"`
	Path     string    `json:"path"`
	Children []Storage `json:"children"`
	Type     string    `json:"type"`
}

type DiskStatus struct {
	Size   uint64 `json:"size"`
	Avail  uint64 `json:"avail"` // 可用空间
	Health bool   `json:"health"`
	Used   uint64 `json:"used"`
}
