package v1

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/common"
	model1 "github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const messagePathStorageStatus = common.ServiceName + ":storage_status"

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

	dbList, err := service.MyService.Disk().GetSerialAllFromDB()
	if err != nil {
		logger.Error("error when getting all volumes from database", zap.Error(err))
		c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	part := make(map[string]int64, len(dbList))
	for _, v := range dbList {
		part[v.MountPoint] = v.CreatedAt
	}

	disks := []model1.Drive{}
	avail := []model1.Drive{}

	var systemDisk *model1.LSBLKModel

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

		temp := service.MyService.Disk().SmartCTL(currentDisk.Path)
		disk.Temperature = temp.Temperature.Current

		if systemDisk == nil {
			// go 5 level deep to look for system block device by mount point being "/"
			systemDisk := service.WalkDisk(currentDisk, 5, func(blk model1.LSBLKModel) bool { return blk.MountPoint == "/" })

			if systemDisk != nil {
				disk.Model = "System"
				if strings.Contains(systemDisk.SubSystems, "mmc") {
					disk.DiskType = "MMC"
				} else if strings.Contains(systemDisk.SubSystems, "usb") {
					disk.DiskType = "USB"
				}
				disk.Health = "true"

				disks = append(disks, disk)
				continue
			}
		}

		if !service.IsDiskSupported(currentDisk) {
			continue
		}

		if reflect.DeepEqual(temp, model1.SmartctlA{}) {
			temp.SmartStatus.Passed = true
		}

		isAvail := true
		if len(currentDisk.MountPoint) != 0 {
			isAvail = false
		} else {
			for _, v := range currentDisk.Children {
				if v.MountPoint != "" {
					isAvail = false
				}
			}
		}

		if isAvail {
			disk.NeedFormat = false
			avail = append(avail, disk)
		}

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
	// if claims, err := jwt.ParseToken(c.GetHeader("Authorization"), false); err != nil || encryption.GetMD5ByStr(js["password"]) != claims.Password {
	// 	c.JSON(http.StatusUnauthorized, model.Result{Success: common_err.PWD_INVALID, Message: common_err.GetMsg(common_err.PWD_INVALID)})
	// 	return
	// }

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
	if len(diskInfo.Children) == 0 && service.IsDiskSupported(diskInfo) {
		t := diskInfo
		t.Children = nil
		diskInfo.Children = append(diskInfo.Children, t)
	}
	for _, v := range diskInfo.Children {
		if err := service.MyService.Disk().UmountPointAndRemoveDir(v); err != nil {
			c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.REMOVE_MOUNT_POINT_ERROR, Message: err.Error()})
			return
		}

		// delete data
		if err := service.MyService.Disk().DeleteMountPointFromDB(v.Path, v.MountPoint); err != nil {
			logger.Error("error when deleting mount point from database", zap.Error(err), zap.String("path", v.Path), zap.String("mount point", v.MountPoint))
		}

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
