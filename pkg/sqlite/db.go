package sqlite

import (
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
)

var (
	_gdb *gorm.DB

	Hooks map[string][]func(interface{})
)

func init() {
	for _, v := range []string{
		"before_create", "after_create",
		"before_save", "after_save",
		"before_update", "after_update",
		"before_delete", "after_delete",
		"after_find",
	} {
		Hooks[v] = make([]func(interface{}), 0)
	}
}

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

	if err := db.AutoMigrate(&model.Merge{}, &model.SerialDisk{}); err != nil {
		panic(err)
	}

	if err := db.Callback().Create().Before("gorm:delete").Register("before_delete", func(d *gorm.DB) {
		if d == nil || d.Statement == nil || d.Statement.Schema == nil || d.Statement.SkipHooks || !d.Statement.Schema.BeforeDelete {
			return
		}

		for _, f := range Hooks["before_delete"] {
			f(d.Statement.Model)
		}
	}); err != nil {
		panic(err)
	}

	_gdb = db

	return _gdb
}
