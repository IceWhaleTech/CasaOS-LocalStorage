package v2

import (
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/sqlite"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	"gorm.io/gorm"
)

func init() {
	sqlite.Hooks[sqlite.HookAfterDelete] = append(sqlite.Hooks[sqlite.HookAfterDelete], hookAfterDeleteSerialDisk)
}

func hookAfterDeleteSerialDisk(db *gorm.DB, model interface{}) {
	if d, ok := model.(*model2.SerialDisk); ok {
		gdb := db.Statement.Context.Value(sqlite.ContextKeyGlobalDB)
		if gdb, ok := gdb.(*gorm.DB); ok {

			var merges []model2.Merge

			if err := gdb.Model(&model2.Merge{}).Preload("SerialDisks").Find(&merges).Error; err != nil {
				panic(err)
			}

			for i := range merges {
				updatedSerialDisks := make([]*model2.SerialDisk, 0)
				for _, serialDisk := range merges[i].SerialDisks {
					if serialDisk.ID != d.ID {
						updatedSerialDisks = append(updatedSerialDisks, serialDisk)
					}
				}

				if err := gdb.Model(&merges[i]).Association("SerialDisks").Error; err != nil {
					panic(err)
				}

				if err := gdb.Model(&merges[i]).Association("SerialDisks").Replace(updatedSerialDisks); err != nil {
					panic(err)
				}
			}
		}

	}
}

func (s *LocalStorageService) GetMergeAll() []model2.Merge {
	var merges []model2.Merge
	s._db.Find(&merges)
	return merges
}

func (s *LocalStorageService) CreateMerge(mountPoint string) error {
	merge := model2.Merge{
		MountPoint: mountPoint,
	}

	return s._db.Save(merge).Error
}

func (s *LocalStorageService) CheckMergeMount() {
	logger.Info("Checking merge mount...")

	// mergeList := s.GetMergeAll()

	// for _, merge := range mergeList {
	// 	// check if serial disk is mounted

	// }
}
