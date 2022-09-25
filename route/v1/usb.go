package v1

import (
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
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
	c.ShouldBind(&js)
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
