package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/common"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/utils/command"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/route"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
)

func init() {
	versionFlag := flag.Bool("v", false, "version")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("v%s\n", common.Version)
		os.Exit(0)
	}

	checkSerialDiskMount()
}

func main() {
	go route.MonitoryUSB()
}

func checkToken2_11() {
	if service.MyService.USB().GetSysInfo().KernelArch == "aarch64" && config.ServerInfo.USBAutoMount != "True" && strings.Contains(service.MyService.USB().GetDeviceTree(), "Raspberry Pi") {
		service.MyService.USB().UpdateUSBAutoMount("False")
		service.MyService.USB().ExecUSBAutoMountShell("False")
	}
}

func checkSerialDiskMount() {
	// check mount point
	dbList := service.MyService.Disk().GetSerialAll()

	list := service.MyService.Disk().LSBLK(true)
	mountPoint := make(map[string]string, len(dbList))
	// remount
	for _, v := range dbList {
		mountPoint[v.UUID] = v.MountPoint
	}
	for _, v := range list {
		command.ExecEnabledSMART(v.Path)
		if v.Children != nil {
			for _, h := range v.Children {
				// if len(h.MountPoint) == 0 && len(v.Children) == 1 && h.FsType == "ext4" {
				if m, ok := mountPoint[h.UUID]; ok {
					// mount point check
					volume := m
					if !file.CheckNotExist(m) {
						for i := 0; file.CheckNotExist(volume); i++ {
							volume = m + strconv.Itoa(i+1)
						}
					}
					service.MyService.Disk().MountDisk(h.Path, volume)
					if volume != m {
						ms := model.SerialDisk{}
						ms.UUID = v.UUID
						ms.MountPoint = volume
						service.MyService.Disk().UpdateMountPoint(ms)
					}

				}
				//}
			}
		}
	}
	service.MyService.Disk().RemoveLSBLKCache()
	command.OnlyExec("source " + config.AppInfo.ShellPath + "/helper.sh ;AutoRemoveUnuseDir")
}
