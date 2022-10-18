package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	interfaces "github.com/IceWhaleTech/CasaOS-Common"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/version"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/common"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/sqlite"
	"gopkg.in/ini.v1"
)

type migrationTool1 struct{}

const (
	defaultDBPath = "/var/lib/casaos"
	tableName     = "o_disk"
)

func (u *migrationTool1) IsMigrationNeeded() (bool, error) {
	status, err := version.GetGlobalMigrationStatus(localStorageNameShort)
	if err == nil {
		_status = status
		if status.LastMigratedVersion != "" {
			_logger.Info("Last migrated version: %s", status.LastMigratedVersion)
			if r, err := version.Compare(status.LastMigratedVersion, common.Version); err == nil {
				return r < 0, nil
			}
		}
	}

	_, err = os.Stat(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		_logger.Info("`%s` not found, migration is not needed.", version.LegacyCasaOSConfigFilePath)
		return false, nil
	}

	var majorVersion, minorVersion, patchVersion int

	majorVersion, minorVersion, patchVersion, err = version.DetectVersion()
	if err != nil {
		_logger.Info("version not detected - trying to detect if it is a legacy version (v0.3.4 or earlier)...")
		majorVersion, minorVersion, patchVersion, err = version.DetectLegacyVersion()
		if err != nil {
			if err == version.ErrLegacyVersionNotFound {
				_logger.Info("legacy version not detected, migration is not needed.")
				return false, nil
			}

			_logger.Error("failed to detect legacy version: %s", err)
			return false, err
		}
	}

	if majorVersion != 0 {
		return false, nil
	}

	if minorVersion < 2 {
		return false, nil
	}

	if minorVersion == 2 && patchVersion < 5 {
		return false, nil
	}

	if minorVersion == 3 && patchVersion > 6 {
		return false, nil
	}

	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		_logger.Error("failed to load config file %s - %s", version.LegacyCasaOSConfigFilePath, err.Error())
		return false, err
	}

	dbFile, err := getDBfile(legacyConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			_logger.Info("database file not found from %s, migration is not needed.", version.LegacyCasaOSConfigFilePath)
			return false, nil
		}

		_logger.Error("failed to get database file from %s - %s", version.LegacyCasaOSConfigFilePath, err.Error())
		return false, err
	}

	legacyDB, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		_logger.Error("failed to open database file %s - %s", dbFile, err.Error())
		return false, err
	}

	defer legacyDB.Close()

	tableExists, err := isTableExist(legacyDB, tableName)
	if err != nil {
		_logger.Error("failed to check if table %s exists - %s", tableName, err.Error())
		return false, err
	}

	if !tableExists {
		_logger.Info("table %s does not exist, migration is not needed.", tableName)
		return false, nil
	}

	_logger.Info("Migration is needed for a CasaOS version between 0.2.5 and 0.3.6...")
	return true, nil
}

func (u *migrationTool1) PreMigrate() error {
	if _, err := os.Stat(localStorageConfigDirPath); os.IsNotExist(err) {
		_logger.Info("Creating %s since it doesn't exists...", localStorageConfigDirPath)
		if err := os.Mkdir(localStorageConfigDirPath, 0o755); err != nil {
			return err
		}
	}

	if _, err := os.Stat(localStorageConfigFilePath); os.IsNotExist(err) {
		_logger.Info("Creating %s since it doesn't exist...", localStorageConfigFilePath)

		f, err := os.Create(localStorageConfigFilePath)
		if err != nil {
			return err
		}

		defer f.Close()

		if _, err := f.WriteString(_localStorageConfigFileSample); err != nil {
			return err
		}
	}

	extension := "." + time.Now().Format("20060102") + ".bak"

	_logger.Info("Creating a backup %s if it doesn't exist...", version.LegacyCasaOSConfigFilePath+extension)
	if err := file.CopySingleFile(version.LegacyCasaOSConfigFilePath, version.LegacyCasaOSConfigFilePath+extension, "skip"); err != nil {
		return err
	}

	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		return err
	}

	dbFile, err := getDBfile(legacyConfigFile)
	if err != nil {
		return err
	}

	_logger.Info("Creating a backup %s if it doesn't exist...", dbFile+extension)
	if err := file.CopySingleFile(dbFile, dbFile+extension, "skip"); err != nil {
		return err
	}

	return nil
}

func (u *migrationTool1) Migrate() error {
	_logger.Info("Loading legacy %s...", version.LegacyCasaOSConfigFilePath)
	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		return err
	}

	migrateConfigurationFile1(legacyConfigFile)

	return migrateDisk1(legacyConfigFile)
}

func (u *migrationTool1) PostMigrate() error {
	defer func() {
		if err := _status.Done(common.Version); err != nil {
			_logger.Error("Failed to update migration status")
			panic(err)
		}
	}()

	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		return err
	}

	if err := postMigrateConfigurationFile1(legacyConfigFile); err != nil {
		return err
	}

	return postMigrateDisk1(legacyConfigFile)
}

func NewMigrationToolFor036AndOlder() interfaces.MigrationTool {
	return &migrationTool1{}
}

func migrateConfigurationFile1(legacyConfigFile *ini.File) {
	_logger.Info("Updating %s with settings from legacy configuration...", config.LocalStorageConfigFilePath)
	config.InitSetup(config.LocalStorageConfigFilePath)

	// LogPath
	if logPath, err := legacyConfigFile.Section("app").GetKey("LogPath"); err == nil {
		_logger.Info("[app] LogPath = %s", logPath.Value())
		config.AppInfo.LogPath = logPath.Value()
	}

	// LogFileExt
	if logFileExt, err := legacyConfigFile.Section("app").GetKey("LogFileExt"); err == nil {
		_logger.Info("[app] LogFileExt = %s", logFileExt.Value())
		config.AppInfo.LogFileExt = logFileExt.Value()
	}

	// DBPath
	if dbPath, err := legacyConfigFile.Section("app").GetKey("DBPath"); err == nil {
		_logger.Info("[app] DBPath = %s", dbPath.Value())
		config.AppInfo.DBPath = dbPath.Value() + "/db"
	}

	_logger.Info("Saving %s...", config.LocalStorageConfigFilePath)
	config.SaveSetup(config.LocalStorageConfigFilePath)
}

func postMigrateConfigurationFile1(legacyConfigFile *ini.File) error {
	_logger.Info("Deleting legacy `USBAutoMount` in %s...", version.LegacyCasaOSConfigFilePath)
	legacyConfigFile.Section("app").DeleteKey("USBAutoMount")

	if err := legacyConfigFile.SaveTo(version.LegacyCasaOSConfigFilePath); err != nil {
		return err
	}

	return nil
}

func migrateDisk1(legacyConfigFile *ini.File) error {
	_logger.Info("Migrating disk information from legacy database to local storage database...")

	dbFile, err := getDBfile(legacyConfigFile)
	if err != nil {
		return err
	}

	legacyDB, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return err
	}

	defer legacyDB.Close()

	newDB := sqlite.GetGlobalDB(config.AppInfo.DBPath)
	diskService := service.NewDiskService(newDB)

	tableExists, err := isTableExist(legacyDB, tableName)
	if err != nil {
		return err
	}

	if !tableExists {
		return nil
	}

	rows, err := legacyDB.Query("SELECT id, uuid, mount_point, created_at FROM " + tableName)
	if err != nil {
		return err
	}

	defer rows.Close()

	volumesInNewDB, err := diskService.GetSerialAllFromDB()
	if err != nil {
		return err
	}

	uuidMapInNewDB := make(map[string]string)
	for _, v := range volumesInNewDB {
		uuidMapInNewDB[v.UUID] = v.MountPoint
	}

	for rows.Next() {
		v := &model2.Volume{}

		if err := rows.Scan(&v.ID, &v.UUID, &v.MountPoint, &v.CreatedAt); err != nil {
			return err
		}

		_logger.Info("volume: %+v", v)

		if v.UUID == "" {
			_logger.Info("UUID is empty, skipping...")
			continue
		}

		if _, ok := uuidMapInNewDB[v.UUID]; ok {
			_logger.Info("volume %s (mount point: %s) already exists in local storage database, skipping...", v.UUID, v.MountPoint)
			continue
		}

		// TODO - update o_disk mount point base path from /DATA to /mnt

		_logger.Info("creating volume %s (mount point: %s) in local storage database...", v.UUID, v.MountPoint)
		if err := diskService.SaveMountPointToDB(*v); err != nil {
			return err
		}
	}

	return nil
}

func postMigrateDisk1(legacyConfigFile *ini.File) error {
	dbFile, err := getDBfile(legacyConfigFile)
	if err != nil {
		return err
	}

	legacyDB, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return err
	}

	defer legacyDB.Close()

	tableExists, err := isTableExist(legacyDB, tableName)
	if err != nil {
		return err
	}

	if !tableExists {
		return nil
	}

	_logger.Info("Dropping `%s` table in legacy database...", tableName)

	if _, err := legacyDB.Exec("DROP TABLE " + tableName); err != nil {
		_logger.Error("Failed to drop `%s` table in legacy database: %s", tableName, err)
	}

	return nil
}

func isTableExist(legacyDB *sql.DB, tableName string) (bool, error) {
	rows, err := legacyDB.Query("SELECT name FROM sqlite_master WHERE type='table' AND name = ?", tableName)
	if err != nil {
		return false, err
	}

	defer rows.Close()

	return rows.Next(), nil
}

func getDBfile(legacyConfigFile *ini.File) (string, error) {
	if legacyConfigFile == nil {
		return "", errors.New("legacy configuration file is nil")
	}

	dbPath := legacyConfigFile.Section("app").Key("DBPath").String()

	dbFile := filepath.Join(dbPath, "db", "casaOS.db")

	if _, err := os.Stat(dbFile); err != nil {
		dbFile = filepath.Join(defaultDBPath, "db", "casaOS.db")

		if _, err := os.Stat(dbFile); err != nil {
			return "", err
		}
	}

	return dbFile, nil
}
