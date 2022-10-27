package main

import (
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"syscall"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	"github.com/pilebones/go-udev/netlink"
	"go.uber.org/zap"
)

func sendDiskBySocket() {
	blkList := service.MyService.Disk().LSBLK(true)

	status := model.DiskStatus{}
	healthy := true

	var systemDisk *model.LSBLKModel

	for _, currentDisk := range blkList {

		if systemDisk == nil {
			// go 5 level deep to look for system block device by mount point being "/"
			systemDisk = service.WalkDisk(currentDisk, 5, func(blk model.LSBLKModel) bool { return blk.MountPoint == "/" })

			if systemDisk != nil {
				s, _ := strconv.ParseUint(systemDisk.FSSize, 10, 64)
				a, _ := strconv.ParseUint(systemDisk.FSAvail, 10, 64)
				u, _ := strconv.ParseUint(systemDisk.FSUsed, 10, 64)
				status.Size += s
				status.Avail += a
				status.Used += u

				continue
			}
		}

		if !service.IsDiskSupported(currentDisk) {
			continue
		}

		temp := service.MyService.Disk().SmartCTL(currentDisk.Path)
		if reflect.DeepEqual(temp, model.SmartctlA{}) {
			healthy = true
		} else {
			healthy = temp.SmartStatus.Passed
		}

		for _, v := range currentDisk.Children {
			s, _ := strconv.ParseUint(v.FSSize, 10, 64)
			a, _ := strconv.ParseUint(v.FSAvail, 10, 64)
			u, _ := strconv.ParseUint(v.FSUsed, 10, 64)
			status.Size += s
			status.Avail += a
			status.Used += u
		}
	}

	status.Health = healthy

	message := make(map[string]interface{})
	message["sys_disk"] = status

	if err := service.MyService.Notify().SendSystemStatusNotify(message); err != nil {
		logger.Error("failed to send notify", zap.Any("message", message), zap.Error(err))
	}
}

func sendUSBBySocket() {
	message := map[string]interface{}{
		"sys_usb": service.MyService.Disk().GetUSBDriveStatusList(),
	}

	if err := service.MyService.Notify().SendSystemStatusNotify(message); err != nil {
		logger.Error("failed to send notify", zap.Any("message", message), zap.Error(err))
	}
}

func monitorUSB() {
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
			if uevent.Env["DEVTYPE"] == "partition" && uevent.Env["ID_BUS"] == "usb" {
				time.Sleep(1 * time.Second)
				sendUSBBySocket()
				continue
			}
		case err := <-errors:
			logger.Error("udev err", zap.Error(err))
		}
	}
}

func sendStorageStats() {
	sendDiskBySocket()
	sendUSBBySocket()
}
