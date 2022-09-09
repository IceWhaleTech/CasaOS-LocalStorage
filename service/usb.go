package service

import (
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	command2 "github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/utils/command"
	"github.com/shirou/gopsutil/host"
)

type USBService interface {
	UpdateUSBAutoMount(state string)
	ExecUSBAutoMountShell(state string)

	GetSysInfo() host.InfoStat
	GetDeviceTree() (string, error)
}

type usbService struct{}

func (s *usbService) UpdateUSBAutoMount(state string) {
	config.ServerInfo.USBAutoMount = state
	config.Cfg.Section("server").Key("USBAutoMount").SetValue(state)
	config.Cfg.SaveTo(config.ConfigFilePath)
}

func (s *usbService) ExecUSBAutoMountShell(state string) {
	if state == "False" {
		command2.OnlyExec("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;USB_Stop_Auto")
	} else {
		command2.OnlyExec("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;USB_Start_Auto")
	}
}

func (c *usbService) GetSysInfo() host.InfoStat {
	info, _ := host.Info()
	return *info
}

func (c *usbService) GetDeviceTree() (string, error) {
	return command2.ExecResultStr("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;GetDeviceTree")
}

func NewUSBService() USBService {
	return &usbService{}
}
