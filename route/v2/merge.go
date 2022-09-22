package v2

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/fs"

	"github.com/labstack/echo/v4"
)

func (s *LocalStorage) GetMerges(ctx echo.Context, params codegen.GetMergesParams) error {
	// TODO return 503 when merge is not enabled

	merges, err := service.MyService.LocalStorage().GetMergeAll(params.MountPoint)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
	}

	data := make([]codegen.Merge, 0, len(merges))
	for _, merge := range merges {
		data = append(data, MergeAdapterOut(merge))
	}

	return ctx.JSON(http.StatusOK, codegen.GetMergesResponseOK{Data: &data})
}

func (s *LocalStorage) SetMerge(ctx echo.Context) error {
	// TODO return 503 when merge is not enabled

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
	if m.SourceVolumePaths != nil {
		allVolumes := service.MyService.Disk().GetSerialAll()
		sourceVolumes = make([]*model2.Volume, 0, len(*m.SourceVolumePaths))
		for _, volumePath := range *m.SourceVolumePaths {
			for i := range allVolumes {
				if volumePath == allVolumes[i].Path {
					sourceVolumes = append(sourceVolumes, &allVolumes[i])
				}
			}
		}
	}

	// set merge
	if err := service.MyService.LocalStorage().SetMerge(&model2.Merge{
		FSType:         fstype,
		MountPoint:     m.MountPoint,
		SourceBasePath: m.SourceBasePath,
		SourceVolumes:  sourceVolumes,
	}); err != nil {
		// TODO - return different HTTP status code based on the error
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{Message: &message})
	}

	return nil
}

func MergeAdapterOut(m model2.Merge) codegen.Merge {
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
