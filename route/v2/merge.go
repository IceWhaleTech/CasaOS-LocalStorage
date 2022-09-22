package v2

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"

	"github.com/labstack/echo/v4"
)

func (s *LocalStorage) GetMerges(ctx echo.Context, params codegen.GetMergesParams) error {
	merges, err := service.MyService.LocalStorage().GetMergeAll(params.MountPoint)
	if err != nil {
		message := err.Error()
		response := codegen.BaseResponse{
			Message: &message,
		}
		return ctx.JSON(http.StatusInternalServerError, response)
	}

	data := make([]codegen.Merge, 0, len(merges))
	for _, merge := range merges {
		data = append(data, MergeAdapter(merge))
	}

	return ctx.JSON(http.StatusOK, codegen.GetMergesResponseOK{
		Data: &data,
	})
}

func (s *LocalStorage) SetMerge(ctx echo.Context, params codegen.SetMergeParams) error {
	var request codegen.Merge
	if err := ctx.Bind(&request); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	// TODO

	return nil
}

func MergeAdapter(m model2.Merge) codegen.Merge {
	id := int(m.ID)

	sourceVolumePaths := make([]string, 0, len(m.SourceVolumes))
	for _, volume := range m.SourceVolumes {
		sourceVolumePaths = append(sourceVolumePaths, volume.Path)
	}

	return codegen.Merge{
		Id:                &id,
		Fstype:            &m.FSType,
		MountPoint:        m.MountPoint,
		SourceBasePath:    m.SourceBasePath,
		SourceVolumePaths: &sourceVolumePaths,
		CreatedAt:         &m.CreatedAt,
		UpdatedAt:         &m.UpdatedAt,
	}
}
