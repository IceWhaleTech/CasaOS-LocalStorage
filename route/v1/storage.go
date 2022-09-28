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
	"net/http"
	"path/filepath"
	"reflect"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-Common/utils/jwt"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/zap"

	model1 "github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/utils/encryption"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	"github.com/gin-gonic/gin"
)

func GetStorageList(c *gin.Context) {
	system := c.Query("system")

	blkList := service.MyService.Disk().LSBLK(false)
	foundSystem := false

	storages := []model1.Storages{}

	for _, currentDisk := range blkList {
		if currentDisk.Tran == "usb" {
			continue
		}

		tempSystemDisk := false
		children := 1
		tempDisk := model1.Storages{
			DiskName: currentDisk.Model,
			Path:     currentDisk.Path,
			Size:     currentDisk.Size,
		}

		storageArr := []model1.Storage{}
		temp := service.MyService.Disk().SmartCTL(currentDisk.Path)
		if reflect.DeepEqual(temp, model1.SmartctlA{}) {
			temp.SmartStatus.Passed = true
		}

		for _, blkChild := range currentDisk.Children {

			if blkChild.MountPoint == "" {
				continue
			}

			if !foundSystem {
				if blkChild.MountPoint == "/" {
					tempDisk.DiskName = "System"
					foundSystem = true
					tempSystemDisk = true
				} else {
					for _, c := range blkChild.Children {
						if c.MountPoint == "/" {
							tempDisk.DiskName = "System"
							foundSystem = true
							tempSystemDisk = true
							break
						}
					}
				}
			}

			stor := model1.Storage{
				MountPoint:  blkChild.MountPoint,
				Size:        blkChild.FSSize,
				Avail:       blkChild.FSAvail,
				Path:        blkChild.Path,
				Type:        blkChild.FsType,
				DriveName:   blkChild.Name,
				PersistedIn: service.MyService.Disk().GetPersistentType(blkChild.Path),
			}

			if len(blkChild.Label) == 0 {
				if stor.MountPoint == "/" {
					stor.Label = "System"
				} else {
					stor.Label = filepath.Base(stor.MountPoint)
				}

				children++
			} else {
				stor.Label = blkChild.Label
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
		} else if !tempSystemDisk {
			tempDisk.Children = storageArr
			storages = append(storages, tempDisk)
		}
	}

	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: storages})
}

func PostAddStorage(c *gin.Context) {
	js := make(map[string]interface{})
	if err := c.ShouldBind(&js); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err.Error()})
		return
	}

	path := js["path"].(string)
	name := js["name"].(string)
	format := js["format"].(bool)

	if len(path) == 0 {
		c.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
		return
	}
	if _, ok := diskMap[path]; ok {
		c.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.DISK_BUSYING, Message: common_err.GetMsg(common_err.DISK_BUSYING)})
		return
	}

	diskMap[path] = "busying"

	defer delete(diskMap, path)

	currentDisk := service.MyService.Disk().GetDiskInfo(path)
	if format {

		output, err := service.MyService.Disk().AddPartition(path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: output})
			return
		}
	}

	currentDisk = service.MyService.Disk().GetDiskInfo(path)

	for _, blkChild := range currentDisk.Children {

		mountPoint := blkChild.GetMountPoint(name)
		if output, err := service.MyService.Disk().MountDisk(blkChild.Path, mountPoint); err != nil {
			c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: output})
			return
		}

		m := model2.Volume{
			MountPoint: mountPoint,
			Path:       blkChild.Path,
			UUID:       blkChild.UUID,
			State:      0,
			CreatedAt:  time.Now().Unix(),
		}

		service.MyService.Disk().SaveMountPoint(m)
		// mount dir
	}

	service.MyService.Disk().RemoveLSBLKCache()

	// send notify to client
	go func() {
		message := map[string]interface{}{
			"data": StorageMessage{
				Action: "ADDED",
				Path:   currentDisk.Children[0].Path,
				Volume: "/mnt/",
				Size:   currentDisk.Children[0].Size,
				Type:   currentDisk.Children[0].Tran,
			},
		}

		if err := service.MyService.Notify().SendNotify(messagePathStorageStatus, message); err != nil {
			logger.Error("error when sending notification", zap.Error(err), zap.String("message path", messagePathStorageStatus), zap.Any("message", message))
		}
	}()

	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

// @Param  pwd formData string true "user password"
// @Param  volume formData string true "mount point"
// @Success 200 {string} string "ok"
// @Router /disk/format [post]
func PutFormatStorage(c *gin.Context) {
	js := make(map[string]string)
	if err := c.ShouldBind(&js); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err.Error()})
		return
	}

	// requires password from user to confirm the action
	if claims, err := jwt.ParseToken(c.GetHeader("Authorization"), false); err != nil || encryption.GetMD5ByStr(js["password"]) != claims.Password {
		c.JSON(http.StatusUnauthorized, model.Result{Success: common_err.PWD_INVALID, Message: common_err.GetMsg(common_err.PWD_INVALID)})
		return
	}

	path := js["path"]
	t := "ext4"
	mountPoint := js["volume"]

	if len(path) == 0 || len(t) == 0 {
		c.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
		return
	}

	if _, ok := diskMap[path]; ok {
		c.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.DISK_BUSYING, Message: common_err.GetMsg(common_err.DISK_BUSYING)})
		return
	}
	diskMap[path] = "busying"
	if output, err := service.MyService.Disk().UmountPointAndRemoveDir(path); err != nil {
		c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.REMOVE_MOUNT_POINT_ERROR, Message: output})
		return
	}

	_, err := service.MyService.Disk().FormatDisk(path, t)
	if err != nil {
		delete(diskMap, path)
		c.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.FORMAT_ERROR, Message: common_err.GetMsg(common_err.FORMAT_ERROR)})
	}

	currentDisk := service.MyService.Disk().GetDiskInfo(path)

	if mountPoint == "" {
		mountPoint = currentDisk.GetMountPoint("")
	}

	if output, err := service.MyService.Disk().MountDisk(path, mountPoint); err != nil {
		c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: output})
		return
	}

	m := model2.Volume{
		MountPoint: mountPoint,
		Path:       currentDisk.Path,
		UUID:       currentDisk.UUID,
		State:      0,
		CreatedAt:  time.Now().Unix(),
	}

	service.MyService.Disk().SaveMountPoint(m)

	service.MyService.Disk().RemoveLSBLKCache()
	delete(diskMap, path)
	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

func DeleteStorage(c *gin.Context) {
	js := make(map[string]string)
	if err := c.ShouldBind(&js); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err.Error()})
		return
	}

	// requires password from user to confirm the action
	if claims, err := jwt.ParseToken(c.GetHeader("Authorization"), false); err != nil || encryption.GetMD5ByStr(js["password"]) != claims.Password {
		c.JSON(http.StatusUnauthorized, model.Result{Success: common_err.PWD_INVALID, Message: common_err.GetMsg(common_err.PWD_INVALID)})
		return
	}

	path := js["path"]
	mountPoint := js["volume"]

	if len(path) == 0 || len(mountPoint) == 0 {
		c.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
		return
	}

	if _, ok := diskMap[path]; ok {
		c.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.DISK_BUSYING, Message: common_err.GetMsg(common_err.DISK_BUSYING)})
		return
	}

	if output, err := service.MyService.Disk().UmountPointAndRemoveDir(path); err != nil {
		c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.REMOVE_MOUNT_POINT_ERROR, Message: output})
		return
	}

	// delete data
	service.MyService.Disk().DeleteMountPoint(path, mountPoint)
	service.MyService.Disk().RemoveLSBLKCache()

	// send notify to client
	go func() {
		message := map[string]interface{}{
			"data": StorageMessage{
				Action: "REMOVED",
				Path:   path,
				Volume: mountPoint,
				Size:   0,
				Type:   "",
			},
		}

		if err := service.MyService.Notify().SendNotify(messagePathStorageStatus, message); err != nil {
			logger.Error("error when sending notification", zap.Error(err), zap.String("message path", messagePathStorageStatus), zap.Any("message", message))
		}
	}()

	c.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}
