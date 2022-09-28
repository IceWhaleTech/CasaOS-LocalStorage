package v1

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-Common/utils/jwt"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	model1 "github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/utils/encryption"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const messagePathStorageStatus = "storage_status"

var diskMap = make(map[string]string)

type StorageMessage struct {
	Type   string `json:"type"`   // sata,usb
	Action string `json:"action"` // remove add
	Path   string `json:"path"`
	Volume string `json:"volume"`
	Size   uint64 `json:"size"`
}

// @Summary disk list
// @Produce  application/json
// @Accept application/json
// @Tags disk
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /disk/list [get]
func GetDiskList(c *gin.Context) {
	blkList := service.MyService.Disk().LSBLK(false)
	foundSystem := false

	dbList := service.MyService.Disk().GetSerialAll()
	part := make(map[string]int64, len(dbList))
	for _, v := range dbList {
		part[v.MountPoint] = v.CreatedAt
	}

	disks := []model1.Drive{}
	avail := []model1.Drive{}

	for _, currentDisk := range blkList {
		disk := model1.Drive{
			Serial:         currentDisk.Serial,
			Name:           currentDisk.Name,
			Size:           currentDisk.Size,
			Path:           currentDisk.Path,
			Model:          currentDisk.Model,
			ChildrenNumber: len(currentDisk.Children),
		}

		if currentDisk.Rota {
			disk.DiskType = "HDD"
		} else {
			disk.DiskType = "SSD"
		}

		if len(currentDisk.Children) > 0 && !foundSystem {
			for _, blkChild := range currentDisk.Children {
				if len(blkChild.Children) > 0 {
					for _, blkGrandChild := range blkChild.Children {
						if blkGrandChild.MountPoint != "/" {
							continue
						}

						disk.Model = "System"
						if strings.Contains(blkGrandChild.SubSystems, "mmc") {
							disk.DiskType = "MMC"
						} else if strings.Contains(blkGrandChild.SubSystems, "usb") {
							disk.DiskType = "USB"
						}
						disk.Health = "true"

						disks = append(disks, disk)
						foundSystem = true
						break
					}

					continue
				}

				if blkChild.MountPoint == "/" {
					continue
				}

				disk.Model = "System"
				if strings.Contains(blkChild.SubSystems, "mmc") {
					disk.DiskType = "MMC"
				} else if strings.Contains(blkChild.SubSystems, "usb") {
					disk.DiskType = "USB"
				}
				disk.Health = "true"

				disks = append(disks, disk)
				foundSystem = true

				break
			}

			if foundSystem {
				continue
			}
		}

		if !isDiskSupported(currentDisk) {
			continue
		}

		temp := service.MyService.Disk().SmartCTL(currentDisk.Path)
		if reflect.DeepEqual(temp, model1.SmartctlA{}) {
			temp.SmartStatus.Passed = true
		}

		isAvail := true
		for _, v := range currentDisk.Children {
			if v.MountPoint != "" {
				isAvail = false
			}
		}

		if isAvail {
			disk.NeedFormat = false
			avail = append(avail, disk)
		}

		disk.Temperature = temp.Temperature.Current
		disk.Health = strconv.FormatBool(temp.SmartStatus.Passed)

		disks = append(disks, disk)
	}

	data := map[string]interface{}{
		"disks": disks,
		"avail": avail,
	}

	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

// @Summary disk list
// @Produce  application/json
// @Accept application/json
// @Tags disk
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /disk/list [get]

func DeleteDisksUmount(c *gin.Context) {
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

	if len(path) == 0 {
		c.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
		return
	}

	if _, ok := diskMap[path]; ok {
		c.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.DISK_BUSYING, Message: common_err.GetMsg(common_err.DISK_BUSYING)})
		return
	}

	diskInfo := service.MyService.Disk().GetDiskInfo(path)
	for _, v := range diskInfo.Children {
		if output, err := service.MyService.Disk().UmountPointAndRemoveDir(v.Path); err != nil {
			c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.REMOVE_MOUNT_POINT_ERROR, Message: output})
			return
		}

		// delete data
		service.MyService.Disk().DeleteMountPoint(v.Path, v.MountPoint)

		if err := service.MyService.Shares().DeleteShare(v.MountPoint); err != nil {
			logger.Error("error when deleting share by mount point", zap.Error(err), zap.String("mount point", v.MountPoint))
		}
	}

	service.MyService.Disk().RemoveLSBLKCache()

	// send notify to client
	go func() {
		message := map[string]interface{}{
			"data": StorageMessage{
				Action: "REMOVED",
				Path:   path,
				Volume: "",
				Size:   0,
				Type:   "",
			},
		}

		if err := service.MyService.Notify().SendNotify(messagePathStorageStatus, message); err != nil {
			logger.Error("error when sending notification", zap.Error(err), zap.String("message path", messagePathStorageStatus), zap.Any("message", message))
		}
	}()

	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: path})
}

func isDiskSupported(d model1.LSBLKModel) bool {
	return d.Tran == "sata" ||
		d.Tran == "nvme" ||
		d.Tran == "spi" ||
		d.Tran == "sas" ||
		strings.Contains(d.SubSystems, "virtio") ||
		strings.Contains(d.SubSystems, "block:scsi:vmbus:acpi") || // Microsoft Hyper-V
		(d.Tran == "ata" && d.Type == "disk")
}
