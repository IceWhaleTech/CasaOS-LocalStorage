package v1

import (
	"os"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/drivers/dropbox"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/drivers/google_drive"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/utils/httper"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	"github.com/labstack/echo/v4"
)

func ListStorages(ctx echo.Context) error {
	// var req model.PageReq
	// if err := ctx.Bind(&req); err != nil {
	// 	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.CLIENT_ERROR, Message: common_err.GetMsg(common_err.CLIENT_ERROR), Data: err.Error()})
	// 	return
	// }
	// req.Validate()

	// logger.Info("ListStorages", zap.Any("req", req))
	// storages, total, err := service.MyService.Storage().GetStorages(req.Page, req.PerPage)
	// if err != nil {
	// 	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: err.Error()})
	// 	return
	// }
	// return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: model.PageResp{
	// 	Content: storages,
	// 	Total:   total,
	// }})
	r, err := service.MyService.Storage().GetStorages()
	if err != nil {
		return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: err.Error()})
	}

	for i := 0; i < len(r.MountPoints); i++ {
		t := service.MyService.Storage().GetAttributeValueByName(r.MountPoints[i].Fs, "type")
		if t == "drive" {
			r.MountPoints[i].Icon = google_drive.ICONURL
		}
		if t == "dropbox" {
			r.MountPoints[i].Icon = dropbox.ICONURL
		}
		r.MountPoints[i].Name = service.MyService.Storage().GetAttributeValueByName(r.MountPoints[i].Fs, "username")
	}
	list := []httper.MountPoint{}

	for _, v := range r.MountPoints {
		list = append(list, httper.MountPoint{
			Fs:         v.Fs,
			Icon:       v.Icon,
			MountPoint: v.MountPoint,
			Name:       v.Name,
		})
	}

	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: list})
}

func UmountStorage(ctx echo.Context) error {
	json := make(map[string]string)
	ctx.Bind(&json)
	mountPoint := json["mount_point"]
	if mountPoint == "" {
		return ctx.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.CLIENT_ERROR, Message: common_err.GetMsg(common_err.CLIENT_ERROR), Data: "mount_point is empty"})
	}
	err := service.MyService.Storage().UnmountStorage(mountPoint)
	if err != nil {
		return ctx.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: err.Error()})
	}
	service.MyService.Storage().DeleteConfigByName(strings.ReplaceAll(mountPoint, "/mnt/", ""))
	if fs, err := os.ReadDir(mountPoint); err == nil && len(fs) == 0 {
		os.RemoveAll(mountPoint)
	}
	return ctx.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: "success"})
}
