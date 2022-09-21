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
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrMergeMountPointAlreadyExists = errors.New("merge mount point already exists")
	ErrMergeMountPointDoesNotExist  = errors.New("merge mount point does not exist")
)

func init() {
	// register the callback function to be called after a serial disk is deleted from database each time
	sqlite.Hooks[sqlite.HookAfterDelete] = append(sqlite.Hooks[sqlite.HookAfterDelete], hookAfterDeleteVolume)
}

// Make sure the serial disk is removed from the merge list when it is deleted from database, to keep the database consistent.
func hookAfterDeleteVolume(db *gorm.DB, model interface{}) {
	if targetVolume, ok := model.(*model2.Volume); ok {
		gdb := db.Statement.Context.Value(sqlite.ContextKeyGlobalDB)
		if gdb, ok := gdb.(*gorm.DB); ok {

			var merges []model2.Merge

			if err := gdb.Model(&model2.Merge{}).Preload(model2.MergeSourceVolumes).Find(&merges).Error; err != nil {
				panic(err)
			}

			for i := range merges {
				updatedVolumes := make([]*model2.Volume, 0)
				for _, sourceVolume := range merges[i].SourceVolumes {
					if sourceVolume.ID != targetVolume.ID {
						updatedVolumes = append(updatedVolumes, sourceVolume)
					}
				}

				if err := gdb.Model(&merges[i]).Association(model2.MergeSourceVolumes).Error; err != nil {
					panic(err)
				}

				if err := gdb.Model(&merges[i]).Association(model2.MergeSourceVolumes).Replace(updatedVolumes); err != nil {
					panic(err)
				}
			}
		}

	}
}

func (s *LocalStorageService) GetMergeAll() ([]model2.Merge, error) {
	var merges []model2.Merge
	if err := s._db.Preload(model2.MergeSourceVolumes).Find(&merges).Error; err != nil {
		return nil, err
	}
	return merges, nil
}

func (s *LocalStorageService) CreateMerge(merge *model2.Merge) error {
	// check if a existingMerge of mouthPoint already exists in database
	var existingMerge model2.Merge
	if result := s._db.Where("mount_point = ?", merge.MountPoint).Limit(1).Find(&existingMerge); result.Error != nil {
		return result.Error
	} else if result.RowsAffected > 0 {
		return ErrMergeMountPointAlreadyExists
	}

	// create source path if it does not exists
	if err := file.IsNotExistMkDir(*merge.SourceBasePath); err != nil {
		return err
	}

	source := *merge.SourceBasePath
	for _, volume := range merge.SourceVolumes {
		source = source + ":" + volume.MountPoint
	}

	// create a new merge mount
	_, err := s.Mount(codegen.Mount{
		MountPoint: merge.MountPoint,
		Fstype:     &merge.FSType,
		Source:     &source,
	})
	if err != nil {
		return err
	}

	// then persist to database
	if err := s._db.Create(&merge).Error; err != nil {
		return err
	}

	return nil
}

func (s *LocalStorageService) UpdateMerge(mountPoint string, volumes []*model2.Volume) error {
	// check if a merge of mount point already exists in database
	var merge model2.Merge
	if result := s._db.Model(&model2.Merge{}).Preload(model2.MergeSourceVolumes).First(
		&merge,
		&model2.Merge{MountPoint: mountPoint},
	); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrMergeMountPointDoesNotExist
	}

	// check if the mount point exists
	if _, err := os.Stat(mountPoint); err != nil {
		return err
	}

	// update the merge mount point
	sources := []string{*merge.SourceBasePath}
	for _, volume := range volumes {
		sources = append(sources, volume.MountPoint)
	}

	// check if the mount point is a mergerfs mount
	if _, err := mergerfs.ListValues(mountPoint); err != nil {
		// try to mount it if it is not a mergerfs mount
		source := strings.Join(sources, ":")
		if _, err := s.Mount(codegen.Mount{
			MountPoint: merge.MountPoint,
			Fstype:     &merge.FSType,
			Source:     &source,
		}); err != nil {
			return err
		}
	} else {
		// otherwise, update the mergerfs sources
		if err := mergerfs.SetSource(mountPoint, sources); err != nil {
			return err
		}
	}

	// then persist to database
	if err := s._db.Model(&merge).Association(model2.MergeSourceVolumes).Error; err != nil {
		return err
	}

	if err := s._db.Model(&merge).Association(model2.MergeSourceVolumes).Replace(volumes); err != nil {
		return err
	}

	return nil
}

func (s *LocalStorageService) CheckMergeMount() {
	logger.Info("Checking merge mount...")

	mergeList, err := s.GetMergeAll()
	if err != nil {
		logger.Error("Failed to get merge list from database", zap.Error(err))
		return
	}

	mounts, err := s.GetMounts(codegen.GetMountsParams{})
	if err != nil {
		logger.Error("Failed to get mount list from system", zap.Error(err))
		return
	}

	for i := range mergeList {
		mountNeeded := true
		for _, mount := range mounts {
			if mount.MountPoint == mergeList[i].MountPoint {
				if *mount.Fstype == mergeList[i].FSType {
					logger.Info("Merge already exists - mount not needed", zap.Any("merge", mergeList[i]))
					mountNeeded = false
					break
				}
				logger.Error("Not a mergerfs mount point", zap.Any("mount", mount))
			}
		}

		if mountNeeded {
			logger.Info("Merge not found - mount needed", zap.Any("merge", mergeList[i]))
			if err := s.UpdateMerge(mergeList[i].MountPoint, mergeList[i].SourceVolumes); err != nil {
				logger.Error("Failed to create merge", zap.Error(err))
			}
			continue
		}

		currentSourceList, err := mergerfs.GetSource(mergeList[i].MountPoint)
		if err != nil {
			logger.Error("Failed to get current source list", zap.Error(err), zap.Any("merge", mergeList[i]))
			continue
		}

		expectSourceList := []string{*mergeList[i].SourceBasePath}
		for _, volume := range mergeList[i].SourceVolumes {
			expectSourceList = append(expectSourceList, volume.MountPoint)
		}

		if !utils.CompareStringSlices(currentSourceList, expectSourceList) {

			logger.Info("Merge source list not match - update needed",
				zap.String("currentSourceList", strings.Join(currentSourceList, ",")),
				zap.String("expectSourceList", strings.Join(expectSourceList, ",")))

			if err := s.UpdateMerge(mergeList[i].MountPoint, mergeList[i].SourceVolumes); err != nil {
				logger.Error("Failed to set merge sources", zap.Any("merge", mergeList[i]), zap.Error(err))
			}
		}
	}
}
