package sqlite

import (
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
)

var _gdb *gorm.DB

func GetDB(dbPath string) *gorm.DB {
	if _gdb != nil {
		return _gdb
	}

	if err := file.IsNotExistMkDir(dbPath); err != nil {
		panic(err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath+"/local-storage.db"), &gorm.Config{})

	c, _ := db.DB()
	c.SetMaxIdleConns(10)
	c.SetMaxOpenConns(100)
	c.SetConnMaxIdleTime(time.Second * 1000)
	if err != nil {
		panic(err)
	}

	if err := db.AutoMigrate(&model.SerialDisk{}); err != nil {
		logger.Error("check or create db error", zap.Error(err))
	}

	_gdb = db

	return _gdb
}
