package sqlite

import (
	"context"
	"path/filepath"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
)

type ContextKey string

const (
	contextKeyGlobalDB = ContextKey("gdb")

	// GORM's lifecyle
	HookBeforeCreate = "before_create"
	HookAfterCreate  = "after_create"
	HookBeforeSave   = "before_save"
	HookAfterSave    = "after_save"
	HookBeforeUpdate = "before_update"
	HookAfterUpdate  = "after_update"
	HookBeforeDelete = "before_delete"
	HookAfterDelete  = "after_delete"
	HookAfterFind    = "after_find"
)

var (
	_gdb *gorm.DB

	// A map of hook to callbacks per GORM’s lifecycle.
	// This allows additional logic to happen when a row is created, updated, deleted, or queried from the database, without changing original model structure or logic.
	// Otherwise we will have to add migration logic from version to version.
	Hooks map[string][]func(*gorm.DB, interface{})
)

func init() {
	Hooks = make(map[string][]func(*gorm.DB, interface{}))

	for _, v := range []string{
		HookBeforeCreate, HookAfterCreate,
		HookBeforeSave, HookAfterSave,
		HookBeforeUpdate, HookAfterUpdate,
		HookBeforeDelete, HookAfterDelete,
		HookAfterFind,
	} {
		Hooks[v] = make([]func(*gorm.DB, interface{}), 0)
	}
}

// Initialize each hook to call each registered callback function when the hook is triggered
func initializeHooks(db *gorm.DB) error {
	if err := db.Callback().Create().Before("gorm:create").Register(HookBeforeCreate, hookFunc(HookBeforeCreate)); err != nil {
		return err
	}

	if err := db.Callback().Create().After("gorm:create").Register(HookAfterCreate, hookFunc(HookAfterCreate)); err != nil {
		return err
	}

	if err := db.Callback().Update().Before("gorm:save").Register(HookBeforeSave, hookFunc(HookBeforeSave)); err != nil {
		return err
	}

	if err := db.Callback().Update().After("gorm:save").Register(HookAfterSave, hookFunc(HookAfterSave)); err != nil {
		return err
	}

	if err := db.Callback().Update().Before("gorm:update").Register(HookBeforeUpdate, hookFunc(HookBeforeUpdate)); err != nil {
		return err
	}

	if err := db.Callback().Update().After("gorm:update").Register(HookAfterUpdate, hookFunc(HookAfterUpdate)); err != nil {
		return err
	}

	if err := db.Callback().Delete().Before("gorm:delete").Register(HookBeforeDelete, hookFunc(HookBeforeDelete)); err != nil {
		return err
	}

	if err := db.Callback().Delete().After("gorm:delete").Register(HookAfterDelete, hookFunc(HookAfterDelete)); err != nil {
		return err
	}

	if err := db.Callback().Query().After("gorm:find").Register(HookAfterFind, hookFunc(HookAfterFind)); err != nil {
		return err
	}

	return nil
}

// Make sure each hook calls each registered callback function per hook name
func hookFunc(name string) func(d *gorm.DB) {
	return func(d *gorm.DB) {
		if d == nil || d.Statement == nil || d.Statement.Schema == nil || d.Statement.SkipHooks {
			return
		}

		gdb := d.Statement.Context.Value(contextKeyGlobalDB)
		if gdb, ok := gdb.(*gorm.DB); ok {
			for _, f := range Hooks[name] {
				f(gdb, d.Statement.Model)
			}
		}
	}
}

func GetDBByFile(dbFile string) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(dbFile), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	c, _ := db.DB()
	c.SetMaxIdleConns(10)
	c.SetMaxOpenConns(100)
	c.SetConnMaxIdleTime(time.Second * 1000)

	if err := db.AutoMigrate(&model.Merge{}, &model.Volume{}); err != nil {
		panic(err)
	}

	if err := initializeHooks(db); err != nil {
		panic(err)
	}

	ctx := context.WithValue(context.Background(), contextKeyGlobalDB, db)

	return db.WithContext(ctx)
}

func GetGlobalDB(dbPath string) *gorm.DB {
	if _gdb != nil {
		return _gdb
	}

	if err := file.IsNotExistMkDir(dbPath); err != nil {
		panic(err)
	}

	_gdb = GetDBByFile(filepath.Join(dbPath, "local-storage.db"))

	return _gdb
}
