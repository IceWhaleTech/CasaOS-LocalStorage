package v1

import (
	"strconv"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	model1 "github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	"github.com/gin-gonic/gin"
)

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
		usbList := service.MyService.Disk().LSBLK(false)
		usb := []model1.USBDriveStatus{}
		for _, v := range usbList {
			if v.Tran == "usb" {
				isMount := false
				temp := model1.USBDriveStatus{}
				temp.Model = v.Model
				temp.Name = v.Name
				temp.Size = v.Size
				for _, child := range v.Children {
					if len(child.MountPoint) > 0 {
						isMount = true
						avail, _ := strconv.ParseUint(child.FSAvail, 10, 64)
						temp.Avail += avail

					}
				}
				if isMount {
					usb = append(usb, temp)
				}
			}
		}

		// TODO - @tiger - implement proxy to notify service

		// service.MyService.Notify().SendUSBInfoBySocket(usb)
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
		usbList := service.MyService.Disk().LSBLK(false)
		usb := []model1.USBDriveStatus{}
		for _, v := range usbList {
			if v.Tran == "usb" {
				isMount := false
				temp := model1.USBDriveStatus{}
				temp.Model = v.Model
				temp.Name = v.Name
				temp.Size = v.Size
				for _, child := range v.Children {
					if len(child.MountPoint) > 0 {
						isMount = true
						avail, _ := strconv.ParseUint(child.FSAvail, 10, 64)
						temp.Avail += avail

					}
				}
				if isMount {
					usb = append(usb, temp)
				}
			}
		}
		// TODO - @tiger - implement proxy to notify service

		// service.MyService.Notify().SendUSBInfoBySocket(usb)
	}()
	c.JSON(common_err.SUCCESS,
		model.Result{
			Success: common_err.SUCCESS,
			Message: common_err.GetMsg(common_err.SUCCESS),
			Data:    state,
		})
}
