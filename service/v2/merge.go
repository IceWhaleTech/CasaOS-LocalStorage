package v2

import (
	"errors"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/mergerfs"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrMergeMountPointAlreadyExists  = errors.New("merge mount point already exists")
	ErrMergeMountPointDoesNotExist   = errors.New("merge mount point does not exist")
	ErrMergeMountPointSourceConflict = errors.New("source mount point should not be a child path of the merge mount point")
	ErrNilReference                  = errors.New("reference is nil")
)

// Make sure the serial disk is removed from the merge list when it is deleted from database, to keep the database consistent.
func hookAfterDeleteVolume(db *gorm.DB, model interface{}) {
	var targetVolumes []model2.Volume

	switch t := model.(type) {
	case model2.Volume:
		targetVolumes = []model2.Volume{t}
	case *model2.Volume:
		targetVolumes = []model2.Volume{*t}
	case []model2.Volume:
		targetVolumes = t
	case *[]model2.Volume:
		targetVolumes = *t
	default:
		return
	}

	var merges []model2.Merge

	if err := db.Model(&model2.Merge{}).Preload(model2.MergeSourceVolumes).Find(&merges).Error; err != nil {
		logger.Error("failed to get merge list from database", zap.Error(err))
		return
	}

	for i := range merges {
		updatedVolumes := make([]*model2.Volume, 0)
		for _, sourceVolume := range merges[i].SourceVolumes {
			for _, targetVolume := range targetVolumes {
				if sourceVolume.ID == targetVolume.ID {
					break // skip including the volume to be deleted
				}
				updatedVolumes = append(updatedVolumes, sourceVolume)
			}
		}

		if err := db.Model(&merges[i]).Association(model2.MergeSourceVolumes).Error; err != nil {
			logger.Error("failed to enter association mode between merges and volumes", zap.Error(err), zap.Any("merge", merges[i]))
			return
		}

		if err := db.Model(&merges[i]).Association(model2.MergeSourceVolumes).Replace(updatedVolumes); err != nil {
			logger.Error("failed to update merge source volumes", zap.Error(err), zap.Any("merge", merges[i]), zap.Any("updatedVolumes", updatedVolumes))
			return
		}
	}
}

func (s *LocalStorageService) CreateMerge(merge *model2.Merge) error {
	if merge == nil {
		logger.Error("`merge` should not be nil")
		return ErrNilReference
	}

	if err := file.IsNotExistMkDir(merge.MountPoint); err != nil {
		return err
	}

	sources, err := buildSources(merge)
	if err != nil {
		return err
	}

	// check if the mount point is empty before creating a new mergerfs mount
	if bool, err := file.IsDirEmpty(merge.MountPoint); err != nil {
		return err
	} else if !bool {
		return ErrMountPointIsNotEmpty
	}

	// create a new merge by mounting mergerfs
	source := strings.Join(sources, ":")
	if _, err := s.Mount(codegen.Mount{
		MountPoint: merge.MountPoint,
		Fstype:     &merge.FSType,
		Source:     &source,
	}); err != nil {
		return err
	}

	return nil
}

func (s *LocalStorageService) UpdateMerge(merge *model2.Merge) error {
	if merge == nil {
		logger.Error("`merge` should not be nil")
		return ErrNilReference
	}

	if !file.Exists(merge.MountPoint) {
		return ErrMergeMountPointDoesNotExist
	}

	sources, err := buildSources(merge)
	if err != nil {
		return err
	}

	// if it is already a merge point, check if the mount point is a mergerfs mount with the same sources
	existingSources, err := mergerfs.GetSource(merge.MountPoint)
	if err != nil {
		return err
	}

	if !utils.CompareStringSlices(sources, existingSources) {
		// update the mergerfs sources if different sources
		if err := mergerfs.SetSource(merge.MountPoint, sources); err != nil {
			return err
		}
	}

	return nil
}

func (s *LocalStorageService) CheckMergeMount() {
	logger.Info("checking merge mount...")

	mergesFromDB, err := s.GetMergeAllFromDB(nil)
	if err != nil {
		logger.Error("failed to get merge list from database", zap.Error(err))
		return
	}

	mounts, err := s.GetMounts(codegen.GetMountsParams{})
	if err != nil {
		logger.Error("failed to get mount list from system", zap.Error(err))
		return
	}

	for i := range mergesFromDB {

		isMergeExist := false

		// for each merge from database by mount point, check if it already mounted, i.e. a mergerfs mount
		for _, mount := range mounts {
			if mount.MountPoint == mergesFromDB[i].MountPoint {
				if *mount.Fstype == mergesFromDB[i].FSType {
					logger.Info("merge already exists", zap.Any("merge", mergesFromDB[i]))
					isMergeExist = true
					break
				}
				logger.Error("not a mergerfs mount point", zap.Any("mount", mount))
			}
		}

		if isMergeExist {
			if err := s.UpdateMerge(&mergesFromDB[i]); err != nil {
				logger.Error("failed to update merge", zap.Error(err), zap.Any("merge", mergesFromDB[i]))
			}
			continue
		} else {
			if err := s.CreateMerge(&mergesFromDB[i]); err != nil {
				logger.Error("failed to create merge", zap.Error(err), zap.Any("merge", mergesFromDB[i]))
			}
		}
	}
}

func buildSources(merge *model2.Merge) ([]string, error) {
	sources := make([]string, 0)

	if merge.SourceBasePath != nil && *merge.SourceBasePath != "" {
		// check if sourceBasePath is under mount point
		if strings.HasPrefix(*merge.SourceBasePath, merge.MountPoint) {
			logger.Error(
				"source base path should not be a child path of the merge mount point",
				zap.String("sourceBasePath", *merge.SourceBasePath),
				zap.String("merge.MountPoint", merge.MountPoint),
			)
			return nil, ErrMergeMountPointSourceConflict
		}

		// create source path if it does not exists
		if err := file.IsNotExistMkDir(*merge.SourceBasePath); err != nil {
			return nil, err
		}

		sources = append(sources, *merge.SourceBasePath)
	}

	for _, sourceVolume := range merge.SourceVolumes {
		if sourceVolume == nil {
			logger.Error("one of the source volumes is nil", zap.Any("sourceVolumes", merge.SourceVolumes))
			return nil, ErrNilReference
		}

		// check if sourceBasePath is under mount point
		if strings.HasPrefix(sourceVolume.MountPoint, merge.MountPoint) {
			logger.Error(
				"mount point of source volume should not be a child path of the mount point",
				zap.Any("sourceVolume.MountPoint", sourceVolume.MountPoint),
				zap.Any("merge.MountPoint", merge.MountPoint),
			)
			return nil, ErrMergeMountPointSourceConflict
		}

		// TODO - append only when the volume with the same UUID is already attached, so we don't incorrectly merge the wrong volume (log this)

		sources = append(sources, sourceVolume.MountPoint)
	}

	return sources, nil
}
