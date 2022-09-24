package v2

import (
	"errors"
	"os"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/mergerfs"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/sqlite"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrMergeMountPointAlreadyExists  = errors.New("merge mount point already exists")
	ErrMergeMountPointDoesNotExist   = errors.New("merge mount point does not exist")
	ErrMergeMountPointSourceConflict = errors.New("source mount point should not be a child path of the merge mount point")
)

func init() {
	// register the callback function to be called after a serial disk is deleted from database each time
	sqlite.Hooks[sqlite.HookAfterDelete] = append(sqlite.Hooks[sqlite.HookAfterDelete], hookAfterDeleteVolume)
}

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

func (s *LocalStorageService) GetMergeAll(mountPoint *string) ([]model2.Merge, error) {
	var merges []model2.Merge

	if mountPoint == nil {
		if err := s._db.Preload(model2.MergeSourceVolumes).Find(&merges).Error; err != nil {
			return nil, err
		}
		return merges, nil
	}

	if err := s._db.Preload(model2.MergeSourceVolumes).Where(&model2.Merge{MountPoint: *mountPoint}).Limit(1).Find(&merges).Error; err != nil {
		return nil, err
	}
	return merges, nil
}

func (s *LocalStorageService) SetMerge(merge *model2.Merge) error {
	// check if the mount point exists
	if _, err := os.Stat(merge.MountPoint); err != nil {
		return ErrMergeMountPointDoesNotExist
	}

	// check if a merge already exists in database by mount point
	var existingMerge model2.Merge

	mergeAlreadyExists := false
	if result := s._db.Where(&model2.Merge{MountPoint: merge.MountPoint}).Limit(1).Find(&existingMerge); result.Error != nil {
		return result.Error
	} else if result.RowsAffected > 0 {
		mergeAlreadyExists = true
	}

	// build sources
	sources := make([]string, 0)

	// source base path
	var sourceBasePath string

	if mergeAlreadyExists && existingMerge.SourceBasePath != nil {
		// default to the existing source base path if not specified in the request
		sourceBasePath = *existingMerge.SourceBasePath
	}

	if merge.SourceBasePath != nil {
		// override original source base path if specified in the request
		sourceBasePath = *merge.SourceBasePath
	}

	if sourceBasePath != "" {
		// check if sourceBasePath is under mount point
		if strings.HasPrefix(sourceBasePath, merge.MountPoint) {
			logger.Error(
				"source base path should not be a child path of the merge mount point",
				zap.String("sourceBasePath", sourceBasePath),
				zap.String("merge.MountPoint", merge.MountPoint),
			)
			return ErrMergeMountPointSourceConflict
		}

		// create source path if it does not exists
		if err := file.IsNotExistMkDir(sourceBasePath); err != nil {
			return err
		}

		sources = append(sources, sourceBasePath)
	}

	// source volumes
	var sourceVolumes []*model2.Volume

	if mergeAlreadyExists && existingMerge.SourceVolumes != nil {
		// default to the original source volumes if not specified in the request
		sourceVolumes = existingMerge.SourceVolumes
	}

	if merge.SourceVolumes != nil {
		// override original source volumes if specified in the request
		sourceVolumes = merge.SourceVolumes
	}

	for _, sourceVolume := range sourceVolumes {
		// check if sourceBasePath is under mount point
		if strings.HasPrefix(sourceVolume.MountPoint, merge.MountPoint) {
			logger.Error(
				"mount point of source volume should not be a child path of the mount point",
				zap.Any("sourceVolume.MountPoint", sourceVolume.MountPoint),
				zap.Any("merge.MountPoint", merge.MountPoint),
			)
			return ErrMergeMountPointSourceConflict
		}

		sources = append(sources, sourceVolume.MountPoint)
	}

	// check if the mount point is NOT a mergerfs mount
	if _, err := mergerfs.ListValues(merge.MountPoint); err != nil {
		// check if the mount point is empty before creating a new mergerfs mount
		if bool, err := file.IsDirEmpty(merge.MountPoint); err != nil {
			return err
		} else if !bool {
			return ErrMountPointIsNotEmpty
		}

		source := strings.Join(sources, ":")
		if _, err := s.Mount(codegen.Mount{
			MountPoint: merge.MountPoint,
			Fstype:     &merge.FSType,
			Source:     &source,
		}); err != nil {
			return err
		}
	} else {
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
	}

	if mergeAlreadyExists {
		// start association mode
		if err := s._db.Model(&existingMerge).Association(model2.MergeSourceVolumes).Error; err != nil {
			return err
		}

		if merge.SourceBasePath != nil && *merge.SourceBasePath != *existingMerge.SourceBasePath {
			existingMerge.SourceBasePath = merge.SourceBasePath
			if err := s._db.Model(&existingMerge).Update(model.MergeSourceBasePath, merge.SourceBasePath).Error; err != nil {
				return err
			}
		}

		if merge.SourceVolumes != nil {
			if err := s._db.Model(&existingMerge).Association(model2.MergeSourceVolumes).Replace(merge.SourceVolumes); err != nil {
				return err
			}
		}
	} else {
		if err := s._db.Create(merge).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *LocalStorageService) CheckMergeMount() {
	logger.Info("checking merge mount...")

	mergeList, err := s.GetMergeAll(nil)
	if err != nil {
		logger.Error("failed to get merge list from database", zap.Error(err))
		return
	}

	mounts, err := s.GetMounts(codegen.GetMountsParams{})
	if err != nil {
		logger.Error("failed to get mount list from system", zap.Error(err))
		return
	}

	for i := range mergeList {
		mountNeeded := true
		for _, mount := range mounts {
			if mount.MountPoint == mergeList[i].MountPoint {
				if *mount.Fstype == mergeList[i].FSType {
					logger.Info("merge already exists - mount not needed", zap.Any("merge", mergeList[i]))
					mountNeeded = false
					break
				}
				logger.Error("not a mergerfs mount point", zap.Any("mount", mount))
			}
		}

		// mount if not mounted yet
		if mountNeeded {
			logger.Info("merge not found - mount needed", zap.Any("merge", mergeList[i]))
			if err := s.SetMerge(&mergeList[i]); err != nil {
				logger.Error("failed to create merge", zap.Error(err))
			}
			continue
		}

		currentSourceList, err := mergerfs.GetSource(mergeList[i].MountPoint)
		if err != nil {
			logger.Error("failed to get current source list", zap.Error(err), zap.Any("merge", mergeList[i]))
			continue
		}

		expectSourceList := []string{*mergeList[i].SourceBasePath}
		for _, volume := range mergeList[i].SourceVolumes {
			expectSourceList = append(expectSourceList, volume.MountPoint)
		}

		if !utils.CompareStringSlices(currentSourceList, expectSourceList) {

			logger.Info("merge source list not match - update needed",
				zap.String("currentSourceList", strings.Join(currentSourceList, ",")),
				zap.String("expectSourceList", strings.Join(expectSourceList, ",")))

			if err := s.SetMerge(&mergeList[i]); err != nil {
				logger.Error("failed to set merge sources", zap.Any("merge", mergeList[i]), zap.Error(err))
			}
		}
	}
}
