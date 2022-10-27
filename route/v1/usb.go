package v1

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	model1 "github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const messagePathSysUSB = "sys_usb"

// @Summary Turn off usb auto-mount
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/usb/off [put]
func PutSystemUSBAutoMount(c *gin.Context) {
	js := make(map[string]string)
	if err := c.ShouldBind(&js); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{Success: common_err.INVALID_PARAMS, Message: err.Error()})
		return
	}

	status := js["state"]
	if status == "on" {
		service.MyService.USB().UpdateUSBAutoMount("True")
		service.MyService.USB().ExecUSBAutoMountShell("True")
	} else {
		service.MyService.USB().UpdateUSBAutoMount("False")
		service.MyService.USB().ExecUSBAutoMountShell("False")
	}

	go func() {
		message := map[string]interface{}{
			"data": service.MyService.Disk().GetUSBDriveStatusList(),
		}

		if err := service.MyService.Notify().SendNotify(messagePathSysUSB, message); err != nil {
			logger.Error("failed to send notify", zap.Any("message", message), zap.Error(err))
		}
	}()

	c.JSON(common_err.SUCCESS,
		model.Result{
			Success: common_err.SUCCESS,
			Message: common_err.GetMsg(common_err.SUCCESS),
		})
}

// @Summary Turn off usb auto-mount
// @Produce  application/json
// @Accept application/json
// @Tags sys
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /sys/usb [get]
func GetSystemUSBAutoMount(c *gin.Context) {
	state := "True"
	if strings.ToLower(config.ServerInfo.USBAutoMount) != "true" {
		state = "False"
	}

	go func() {
		message := map[string]interface{}{
			"data": service.MyService.Disk().GetUSBDriveStatusList(),
		}

		if err := service.MyService.Notify().SendNotify(messagePathSysUSB, message); err != nil {
			logger.Error("failed to send notify", zap.Any("message", message), zap.Error(err))
		}
	}()

	c.JSON(common_err.SUCCESS,
		model.Result{
			Success: common_err.SUCCESS,
			Message: common_err.GetMsg(common_err.SUCCESS),
			Data:    state,
		})
}

func GetDisksUSBList(c *gin.Context) {
	list := service.MyService.Disk().LSBLK(false)
	data := []model1.USBDriveStatus{}
	for _, v := range list {
		if v.Tran == "usb" {
			temp := model1.USBDriveStatus{}
			temp.Model = v.Model
			temp.Name = v.Label
			if temp.Name == "" {
				temp.Name = v.Name
			}
			temp.Size = v.Size
			children := []model1.USBChildren{}
			for _, child := range v.Children {
				if len(child.MountPoint) > 0 {
					tempChildren := model1.USBChildren{}
					tempChildren.MountPoint = child.MountPoint
					tempChildren.Size, _ = strconv.ParseUint(child.FSSize.String(), 10, 64)
					tempChildren.Avail, _ = strconv.ParseUint(child.FSAvail.String(), 10, 64)
					tempChildren.Name = child.Label
					if len(tempChildren.Name) == 0 {
						tempChildren.Name = filepath.Base(child.MountPoint)
					}
					avail, _ := strconv.ParseUint(child.FSAvail.String(), 10, 64)
					children = append(children, tempChildren)
					temp.Avail += avail
				}
			}

			temp.Children = children
			data = append(data, temp)
		}
	}
	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

func DeleteDiskUSB(c *gin.Context) {
	js := make(map[string]string)
	if err := c.ShouldBind(&js); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err.Error()})
		return
	}
	mountPoint := js["mount_point"]
	if file.CheckNotExist(mountPoint) {
		c.JSON(http.StatusBadRequest, model.Result{Success: common_err.DIR_NOT_EXISTS, Message: common_err.GetMsg(common_err.DIR_NOT_EXISTS)})
		return
	}

	if err := service.MyService.Disk().UmountUSB(mountPoint); err != nil {
		c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: mountPoint})
}
