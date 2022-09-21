package v2

import (
	"errors"
	"os"

	"github.com/IceWhaleTech/CasaOS-Common/utils/constants"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/mergerfs"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/sqlite"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/fs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrMergeMountPointAlreadyExists = errors.New("merge mount point already exists")
	ErrMergeMountPointDoesNotExist  = errors.New("merge mount point does not exist")
)

func init() {
	// register the callback function to be called after a serial disk is deleted from database each time
	sqlite.Hooks[sqlite.HookAfterDelete] = append(sqlite.Hooks[sqlite.HookAfterDelete], hookAfterDeleteMountPoint)
}

// Make sure the serial disk is removed from the merge list when it is deleted from database, to keep the database consistent.
func hookAfterDeleteMountPoint(db *gorm.DB, model interface{}) {
	if d, ok := model.(*model2.Mount); ok {
		gdb := db.Statement.Context.Value(sqlite.ContextKeyGlobalDB)
		if gdb, ok := gdb.(*gorm.DB); ok {

			var merges []model2.Merge

			if err := gdb.Model(&model2.Merge{}).Preload("SourceMounts").Find(&merges).Error; err != nil {
				panic(err)
			}

			for i := range merges {
				updatedMounts := make([]*model2.Mount, 0)
				for _, serialDisk := range merges[i].SourceMounts {
					if serialDisk.ID != d.ID {
						updatedMounts = append(updatedMounts, serialDisk)
					}
				}

				if err := gdb.Model(&merges[i]).Association("SourceMounts").Error; err != nil {
					panic(err)
				}

				if err := gdb.Model(&merges[i]).Association("SourceMounts").Replace(updatedMounts); err != nil {
					panic(err)
				}
			}
		}

	}
}

func (s *LocalStorageService) GetMergeAll() ([]model2.Merge, error) {
	var merges []model2.Merge
	if err := s._db.Preload("SourceMounts").Find(&merges).Error; err != nil {
		return nil, err
	}
	return merges, nil
}

func (s *LocalStorageService) CreateMerge(mountPoint string) error {
	// check if a merge of mouthPoint already exists in database
	var merge model2.Merge
	if result := s._db.Where("mount_point = ?", mountPoint).First(&merge); result.Error != nil {
		return result.Error
	} else if result.RowsAffected > 0 {
		return ErrMergeMountPointAlreadyExists
	}

	// if not, create a new merge mount
	fstype := fs.MergerFS
	source := constants.DefaultFilePath
	mount, err := s.Mount(codegen.Mount{
		MountPoint: mountPoint,
		Fstype:     &fstype,
		Source:     &source,
	})
	if err != nil {
		return err
	}

	// then persist to database
	merge = model2.Merge{
		MountPoint: mount.MountPoint,
	}

	if err := s._db.Create(&merge).Error; err != nil {
		return err
	}

	return nil
}

func (s *LocalStorageService) UpdateMerge(mountPoint string, mounts []*model2.Mount) error {
	// check if a merge of mount point already exists in database
	var merge model2.Merge
	if result := s._db.Model(&model2.Merge{}).Preload("SourceMounts").First(
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

	// check if the mount point is a mergerfs mount
	if _, err := mergerfs.ListValues(mountPoint); err != nil {
		return err
	}

	// update the merge mount point
	sources := make([]string, 0)
	for _, mount := range mounts {
		sources = append(sources, mount.MountPoint)
	}

	if err := mergerfs.SetSource(mountPoint, sources); err != nil {
		return err
	}

	// then persist to database
	if err := s._db.Model(&merge).Association("SourceMounts").Error; err != nil {
		return err
	}

	if err := s._db.Model(&merge).Association("SourceMounts").Replace(mounts); err != nil {
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

	fstype := fs.MergerFS
	source := constants.DefaultFilePath

	codegenMounts, err := s.GetMounts(codegen.GetMountsParams{})
	if err != nil {
		logger.Error("Failed to get mount list from system", zap.Error(err))
	}

	for _, merge := range mergeList {
		mountNeeded := true
		for _, codegenMount := range codegenMounts {
			if codegenMount.MountPoint == merge.MountPoint {
				if *codegenMount.Fstype == fstype {
					mountNeeded = false
					break
				}
				logger.Error("Not a mergerfs mount point", zap.Any("mount", codegenMount))
			}
		}

		if mountNeeded {
			if _, err := s.Mount(codegen.Mount{
				MountPoint: merge.MountPoint,
				Fstype:     &fstype,
				Source:     &source,
			}); err != nil {
				logger.Error("Failed to mount merge", zap.Any("merge", merge), zap.Error(err))
			}
		}

		if err := s.UpdateMerge(merge.MountPoint, merge.SourceMounts); err != nil {
			logger.Error("Failed to set merge sources", zap.Any("merge", merge), zap.Error(err))
		}
	}
}
