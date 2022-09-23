package main

import (
	"os"
	"strings"

	interfaces "github.com/IceWhaleTech/CasaOS-Common"
	"github.com/IceWhaleTech/CasaOS-Common/utils/version"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/common"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
)

type migrationTool1 struct{}

func (u *migrationTool1) IsMigrationNeeded() (bool, error) {
	if status, err := version.GetGlobalMigrationStatus(localStorageNameShort); err == nil {
		_status = status
		if status.LastMigratedVersion != "" {
			_logger.Info("Last migrated version: %s", status.LastMigratedVersion)
			if r, err := version.Compare(status.LastMigratedVersion, common.Version); err == nil {
				return r < 0, nil
			}
		}
	}

	return false, nil
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

	return nil
}

func (u *migrationTool1) Migrate() error {
	checkToken2_11()

	// TODO - update o_disk mount point base path from /DATA to /mnt
	return nil
}

func (u *migrationTool1) PostMigrate() error {
	defer func() {
		if err := _status.Done(common.Version); err != nil {
			_logger.Error("Failed to update migration status")
			panic(err)
		}
	}()

	return nil
}

func NewMigrationToolDummy() interfaces.MigrationTool {
	return &migrationTool1{}
}

func checkToken2_11() {
	deviceTree, err := service.MyService.USB().GetDeviceTree()
	if err != nil {
		panic(err)
	}

	if service.MyService.USB().GetSysInfo().KernelArch == "aarch64" && strings.ToLower(config.ServerInfo.USBAutoMount) != "true" && strings.Contains(deviceTree, "Raspberry Pi") {
		service.MyService.USB().UpdateUSBAutoMount("False")
		service.MyService.USB().ExecUSBAutoMountShell("False")
	}
}
