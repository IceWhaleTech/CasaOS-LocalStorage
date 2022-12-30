package v2

import (
	"net/http"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/utils/constants"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/utils/merge"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/fs"

	"github.com/labstack/echo/v4"
)

var MessageMergerFSNotEnabled = "mergerfs is not enabled - either it is not enabled in configuration file; merge point is not empty before mounting; or mergerfs is not installed"

func (s *LocalStorage) GetMerges(ctx echo.Context, params codegen.GetMergesParams) error {
	if strings.ToLower(config.ServerInfo.EnableMergerFS) != "true" {
		return ctx.JSON(http.StatusServiceUnavailable, codegen.ResponseServiceUnavailable{Message: &MessageMergerFSNotEnabled})
	}

	merges, err := service.MyService.LocalStorage().GetMerges(params.MountPoint)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
	}

	message := "ok"

	data := make([]codegen.Merge, 0, len(merges))
	for _, merge := range merges {
		data = append(data, MergeAdapterOut(merge))
	}

	return ctx.JSON(http.StatusOK, codegen.GetMergesResponseOK{Data: &data, Message: &message})
}

func (s *LocalStorage) SetMerge(ctx echo.Context) error {
	if strings.ToLower(config.ServerInfo.EnableMergerFS) != "true" {

		file.MoveFile("/DATA", constants.DefaultFilePath)
		file.RMDir("/DATA")

		if !merge.IsMergerFSInstalled() {
			config.ServerInfo.EnableMergerFS = "false"
			logger.Info("mergerfs is disabled")
		}

		if !service.MyService.Disk().EnsureDefaultMergePoint() {
			config.ServerInfo.EnableMergerFS = "false"
			logger.Info("mergerfs is disabled")
		}

		service.MyService.LocalStorage().CheckMergeMount()

		config.Cfg.Section("server").Key("EnableMergerFS").SetValue("true")
		config.ServerInfo.EnableMergerFS = "true"

		config.Cfg.SaveTo(config.LocalStorageConfigFilePath)
	}

	var m codegen.Merge
	if err := ctx.Bind(&m); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	// default to mergerfs if fstype is not specified
	fstype := fs.MergerFSFullName
	if m.Fstype != nil {
		fstype = *m.Fstype
	}

	// expand source volume paths to source volumes
	var sourceVolumes []*model2.Volume
	if m.SourceVolumeUuids != nil {
		volumesFromDB, err := service.MyService.Disk().GetSerialAllFromDB()
		if err != nil {
			message := err.Error()
			return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
		}

		sourceVolumes = make([]*model2.Volume, 0, len(*m.SourceVolumeUuids))
		for _, volumeUUID := range *m.SourceVolumeUuids {
			volumeFound := false
			for i := range volumesFromDB {
				if volumeUUID == volumesFromDB[i].UUID {
					volumeFound = true
					sourceVolumes = append(sourceVolumes, &volumesFromDB[i])
					break
				}
			}

			if !volumeFound {
				message := "volume " + volumeUUID + " not found, or it is not a CasaOS storage. Consider adding it to CasaOS first."
				return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
			}
		}
	}

	merge, err := service.MyService.LocalStorage().GetFirstMergeFromDB(m.MountPoint)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
	}

	if merge == nil {
		merge = &model2.Merge{
			FSType:         fstype,
			MountPoint:     m.MountPoint,
			SourceBasePath: m.SourceBasePath,
			SourceVolumes:  sourceVolumes,
		}

		if err := service.MyService.LocalStorage().CreateMerge(merge); err != nil {
			message := err.Error()
			return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
		}

		if err := service.MyService.LocalStorage().CreateMergeInDB(merge); err != nil {
			message := err.Error()
			return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
		}
	} else {
		if m.SourceBasePath != nil {
			merge.SourceBasePath = m.SourceBasePath
		}

		if m.SourceVolumeUuids != nil {
			merge.SourceVolumes = sourceVolumes // which come from m.SourceVolumeUuids
		}

		if err := service.MyService.LocalStorage().UpdateMerge(merge); err != nil {
			message := err.Error()
			return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
		}

		if err := service.MyService.LocalStorage().UpdateMergeSourcesInDB(merge); err != nil {
			message := err.Error()
			return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
		}
	}

	result := MergeAdapterOut(*merge)

	return ctx.JSON(http.StatusOK, codegen.SetMergeResponseOK{
		Data: &result,
	})
}

func MergeAdapterOut(m model2.Merge) codegen.Merge {
	id := int(m.ID)

	sourceVolumeUUIDs := make([]string, 0, len(m.SourceVolumes))
	for _, volume := range m.SourceVolumes {
		sourceVolumeUUIDs = append(sourceVolumeUUIDs, volume.UUID)
	}

	return codegen.Merge{
		Id:                &id,
		Fstype:            &m.FSType,
		MountPoint:        m.MountPoint,
		SourceBasePath:    m.SourceBasePath,
		SourceVolumeUuids: &sourceVolumeUUIDs,
		CreatedAt:         &m.CreatedAt,
		UpdatedAt:         &m.UpdatedAt,
	}
}
