package v2

import (
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
)


func (s *LocalStorageService) GetMergeAll() []model2.Merge {
	var m []model2.Merge
	s._db.Find(&m)
	return m
}

func (s *LocalStorageService) CheckMergeMount() {
	logger.Info("Checking merge mount...")

	// mergeList := s.GetMergeAll()

	// for _, merge := range mergeList {
	// 	// check if serial disk is mounted
		
	// }
}