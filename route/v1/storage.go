/*@Author: LinkLeong link@icewhale.com
 *@Date: 2022-07-11 16:02:29
 *@LastEditors: LinkLeong
 *@LastEditTime: 2022-08-17 19:14:50
 *@FilePath: /CasaOS/route/v1/storage.go
 *@Description:
 *@Website: https://www.casaos.io
 *Copyright (c) 2022 by icewhale, All Rights Reserved.
 */
package v1

import (
	"path/filepath"
	"reflect"

	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"

	model1 "github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	"github.com/gin-gonic/gin"
)

func GetStorageList(c *gin.Context) {
	system := c.Query("system")
	storages := []model1.Storages{}
	disks := service.MyService.Disk().LSBLK(false)
	diskNumber := 1
	foundSystem := false
	for _, d := range disks {
		if d.Tran == "usb" {
			continue
		}

		tempSystemDisk := false
		children := 1
		tempDisk := model1.Storages{
			DiskName: d.Model,
			Path:     d.Path,
			Size:     d.Size,
		}

		storageArr := []model1.Storage{}
		temp := service.MyService.Disk().SmartCTL(d.Path)
		if reflect.DeepEqual(temp, model1.SmartctlA{}) {
			temp.SmartStatus.Passed = true
		}

		for _, v := range d.Children {

			if v.MountPoint == "" {
				continue
			}

			if !foundSystem {
				if v.MountPoint == "/" {
					tempDisk.DiskName = "System"
					foundSystem = true
					tempSystemDisk = true
				}

				for _, c := range v.Children {
					if c.MountPoint == "/" {
						tempDisk.DiskName = "System"
						foundSystem = true
						tempSystemDisk = true
						break
					}
				}
			}

			stor := model1.Storage{
				MountPoint:  v.MountPoint,
				Size:        v.FSSize,
				Avail:       v.FSAvail,
				Path:        v.Path,
				Type:        v.FsType,
				DriveName:   v.Name,
				PersistedIn: service.MyService.Disk().GetPersistentType(v.Path),
			}

			if len(v.Label) == 0 {
				if stor.MountPoint == "/" {
					stor.Label = "System"
				} else {
					stor.Label = filepath.Base(stor.MountPoint)
				}

				children++
			} else {
				stor.Label = v.Label
			}
			storageArr = append(storageArr, stor)
		}

		if len(storageArr) == 0 {
			continue
		}

		if tempSystemDisk && len(system) > 0 {
			tempStorageArr := []model1.Storage{}
			for i := 0; i < len(storageArr); i++ {
				if storageArr[i].MountPoint != "/boot/efi" && storageArr[i].Type != "swap" {
					tempStorageArr = append(tempStorageArr, storageArr[i])
				}
			}
			tempDisk.Children = tempStorageArr
			storages = append(storages, tempDisk)
			diskNumber++
		} else if !tempSystemDisk {
			tempDisk.Children = storageArr
			storages = append(storages, tempDisk)
			diskNumber++
		}
	}

	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: storages})
}
