package main

import (
	"strings"

	interfaces "github.com/IceWhaleTech/CasaOS-Common"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
)

type migrationTool1 struct{}

func (u *migrationTool1) IsMigrationNeeded() (bool, error) {
	return false, nil
}

func (u *migrationTool1) PreMigrate() error {
	return nil
}

func (u *migrationTool1) Migrate() error {
	checkToken2_11()
	return nil
}

func (u *migrationTool1) PostMigrate() error {
	return nil
}

func NewMigrationToolDummy() interfaces.MigrationTool {
	return &migrationTool1{}
}

func checkToken2_11() {
	if service.MyService.USB().GetSysInfo().KernelArch == "aarch64" && config.ServerInfo.USBAutoMount != "True" && strings.Contains(service.MyService.USB().GetDeviceTree(), "Raspberry Pi") {
		service.MyService.USB().UpdateUSBAutoMount("False")
		service.MyService.USB().ExecUSBAutoMountShell("False")
	}
}
