package service

import (
	"os"

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

func (s *usbService) GetSysInfo() host.InfoStat {
	info, _ := host.Info()
	return *info
}

func (s *usbService) GetDeviceTree() (string, error) {
	deviceTreeFilePath := "/proc/device-tree/model"

	if _, err := os.Stat(deviceTreeFilePath); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	// read string from deviceTreeFilePath
	deviceTree, err := os.ReadFile(deviceTreeFilePath)
	if err != nil {
		return "", err
	}

	return string(deviceTree), nil
}

func NewUSBService() USBService {
	return &usbService{}
}
