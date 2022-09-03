package route

import (
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	"github.com/pilebones/go-udev/netlink"
	"go.uber.org/zap"
)

func SendDiskBySocket() {
	list := service.MyService.Disk().LSBLK(true)

	summary := model.Summary{}
	healthy := true
	findSystem := 0

	for i := 0; i < len(list); i++ {
		if len(list[i].Children) > 0 && findSystem == 0 {
			for j := 0; j < len(list[i].Children); j++ {
				if len(list[i].Children[j].Children) > 0 {
					for _, v := range list[i].Children[j].Children {
						if v.MountPoint == "/" {
							s, _ := strconv.ParseUint(v.FSSize, 10, 64)
							a, _ := strconv.ParseUint(v.FSAvail, 10, 64)
							u, _ := strconv.ParseUint(v.FSUsed, 10, 64)
							summary.Size += s
							summary.Avail += a
							summary.Used += u
							findSystem = 1
							break
						}
					}
				} else {
					if list[i].Children[j].MountPoint == "/" {
						s, _ := strconv.ParseUint(list[i].Children[j].FSSize, 10, 64)
						a, _ := strconv.ParseUint(list[i].Children[j].FSAvail, 10, 64)
						u, _ := strconv.ParseUint(list[i].Children[j].FSUsed, 10, 64)
						summary.Size += s
						summary.Avail += a
						summary.Used += u
						findSystem = 1
						break
					}
				}
			}
		}
		if findSystem == 1 {
			findSystem += 1
			continue
		}
		if list[i].Tran == "sata" || list[i].Tran == "nvme" || list[i].Tran == "spi" || list[i].Tran == "sas" || strings.Contains(list[i].SubSystems, "virtio") || (list[i].Tran == "ata" && list[i].Type == "disk") {
			temp := service.MyService.Disk().SmartCTL(list[i].Path)
			if reflect.DeepEqual(temp, model.SmartctlA{}) {
				healthy = true
			} else {
				healthy = temp.SmartStatus.Passed
			}

			// list[i].Temperature = temp.Temperature.Current

			if len(list[i].Children) > 0 {
				for _, v := range list[i].Children {
					s, _ := strconv.ParseUint(v.FSSize, 10, 64)
					a, _ := strconv.ParseUint(v.FSAvail, 10, 64)
					u, _ := strconv.ParseUint(v.FSUsed, 10, 64)
					summary.Size += s
					summary.Avail += a
					summary.Used += u
				}
			}

		}
	}

	summary.Health = healthy

	// TODO - @tiger - implement proxy to notify service

	// service.MyService.Notify().SendDiskInfoBySocket(summary)
}

func SendUSBBySocket() {
	usbList := service.MyService.Disk().LSBLK(false)
	usb := []model.DriveUSB{}
	for _, v := range usbList {
		if v.Tran == "usb" {
			isMount := false
			temp := model.DriveUSB{}
			temp.Model = v.Model
			temp.Name = v.Name
			temp.Size = v.Size
			for _, child := range v.Children {
				if len(child.MountPoint) > 0 {
					isMount = true
					avail, _ := strconv.ParseUint(child.FSAvail, 10, 64)
					temp.Avail += avail

				}
			}
			if isMount {
				usb = append(usb, temp)
			}
		}
	}

	// TODO - @tiger - implement proxy to notify service

	// service.MyService.Notify().SendUSBInfoBySocket(usb)
}

func MonitoryUSB() {
	var matcher netlink.Matcher

	conn := new(netlink.UEventConn)
	if err := conn.Connect(netlink.UdevEvent); err != nil {
		logger.Error("udev err", zap.Any("Unable to connect to Netlink Kobject UEvent socket", err))
	}
	defer conn.Close()

	queue := make(chan netlink.UEvent)
	errors := make(chan error)
	quit := conn.Monitor(queue, errors, matcher)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-signals
		close(quit)
		os.Exit(0)
	}()

	for {
		select {
		case uevent := <-queue:
			if uevent.Env["DEVTYPE"] == "disk" {
				time.Sleep(time.Microsecond * 500)
				SendUSBBySocket()
				continue
			}
		case err := <-errors:
			logger.Error("udev err", zap.Any("err", err))
		}
	}
}
