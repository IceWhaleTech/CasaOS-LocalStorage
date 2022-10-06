package v2

import (
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/sqlite"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
)

func init() {
	// register the callback function to be called after a serial disk is deleted from database each time
	sqlite.Hooks[sqlite.HookAfterDelete] = append(sqlite.Hooks[sqlite.HookAfterDelete], hookAfterDeleteVolume)
}

func (s *LocalStorageService) GetMergeAllFromDB(mountPoint *string) ([]model2.Merge, error) {
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

func (s *LocalStorageService) GetFirstMergeFromDB(mountPoint string) (*model2.Merge, error) {
	var merge model2.Merge

	if result := s._db.Preload(model2.MergeSourceVolumes).Where(&model2.Merge{MountPoint: mountPoint}).Limit(1).Find(&merge); result.Error != nil {
		return nil, result.Error
	} else if result.RowsAffected == 0 {
		return nil, nil
	}

	return &merge, nil
}

func (s *LocalStorageService) UpdateMergeSourcesInDB(existingMergeInDB *model2.Merge) error {
	if existingMergeInDB == nil {
		return nil
	}

	if err := s._db.Model(existingMergeInDB).Update(model.MergeSourceBasePath, existingMergeInDB.SourceBasePath).Error; err != nil {
		return err
	}

	// start association mode
	if err := s._db.Model(existingMergeInDB).Association(model2.MergeSourceVolumes).Error; err != nil {
		return err
	}

	if err := s._db.Model(existingMergeInDB).Association(model2.MergeSourceVolumes).Replace(existingMergeInDB.SourceVolumes); err != nil {
		return err
	}

	return nil
}

func (s *LocalStorageService) CreateMergeInDB(merge *model2.Merge) error {
	if result := s._db.Create(merge); result.Error != nil {
		return result.Error
	}
	return nil
}
