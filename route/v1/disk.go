package v1

import (
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/jwt"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	model1 "github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/utils/encryption"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service"
	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/disk"
	"go.uber.org/zap"
)

const messagePathStorageStatus = "storage_status"

var diskMap = make(map[string]string)

type StorageMessage struct {
	Type   string `json:"type"`   // sata,usb
	Action string `json:"action"` // remove add
	Path   string `json:"path"`
	Volume string `json:"volume"`
	Size   uint64 `json:"size"`
}

// @Summary disk list
// @Produce  application/json
// @Accept application/json
// @Tags disk
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /disk/list [get]
func GetDiskList(c *gin.Context) {
	path := c.Query("path")
	if len(path) > 0 {
		m := service.MyService.Disk().GetDiskInfo(path)
		c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: m})
		return
	}
	t := c.DefaultQuery("type", "")
	list := service.MyService.Disk().LSBLK(false)
	if t == "usb" {
		data := []model1.USBDriveStatus{}
		for _, v := range list {
			if v.Tran == "usb" {
				temp := model1.USBDriveStatus{}
				temp.Model = v.Model
				temp.Name = v.Name
				temp.Size = v.Size
				for _, child := range v.Children {
					if len(child.MountPoint) > 0 {
						avail, _ := strconv.ParseUint(child.FSAvail, 10, 64)
						temp.Avail += avail
					}
				}
				data = append(data, temp)
			}
		}
		c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
		return
	}

	dbList := service.MyService.Disk().GetSerialAll()
	part := make(map[string]int64, len(dbList))
	for _, v := range dbList {
		part[v.MountPoint] = v.CreatedAt
	}
	findSystem := 0

	disks := []model1.Drive{}
	storage := []model1.Storage{}
	avail := []model1.Drive{}
	for i := 0; i < len(list); i++ {
		disk := model1.Drive{}
		if list[i].Rota {
			disk.DiskType = "HDD"
		} else {
			disk.DiskType = "SSD"
		}
		disk.Serial = list[i].Serial
		disk.Name = list[i].Name
		disk.Size = list[i].Size
		disk.Path = list[i].Path
		disk.Model = list[i].Model
		disk.ChildrenNumber = len(list[i].Children)
		if len(list[i].Children) > 0 && findSystem == 0 {
			for j := 0; j < len(list[i].Children); j++ {
				if len(list[i].Children[j].Children) > 0 {
					for _, v := range list[i].Children[j].Children {
						if v.MountPoint == "/" {
							stor := model1.Storage{}
							stor.MountPoint = v.MountPoint
							stor.Size = v.FSSize
							stor.Avail = v.FSAvail
							stor.Path = v.Path
							stor.Type = v.FsType
							stor.DriveName = "System"
							disk.Model = "System"
							if strings.Contains(v.SubSystems, "mmc") {
								disk.DiskType = "MMC"
							} else if strings.Contains(v.SubSystems, "usb") {
								disk.DiskType = "USB"
							}
							disk.Health = "true"

							disks = append(disks, disk)
							storage = append(storage, stor)
							findSystem = 1
							break
						}
					}
				} else {
					if list[i].Children[j].MountPoint == "/" {
						stor := model1.Storage{}
						stor.MountPoint = list[i].Children[j].MountPoint
						stor.Size = list[i].Children[j].FSSize
						stor.Avail = list[i].Children[j].FSAvail
						stor.Path = list[i].Children[j].Path
						stor.Type = list[i].Children[j].FsType
						stor.DriveName = "System"
						disk.Model = "System"
						if strings.Contains(list[i].Children[j].SubSystems, "mmc") {
							disk.DiskType = "MMC"
						} else if strings.Contains(list[i].Children[j].SubSystems, "usb") {
							disk.DiskType = "USB"
						}
						disk.Health = "true"

						disks = append(disks, disk)
						storage = append(storage, stor)
						findSystem = 1
						break
					}
				}
			}
		}
		if findSystem == 1 {
			findSystem++
			continue
		}

		if !isDiskSupported(&list[i]) {
			continue
		}

		temp := service.MyService.Disk().SmartCTL(list[i].Path)
		if reflect.DeepEqual(temp, model1.SmartctlA{}) {
			temp.SmartStatus.Passed = true
		}

		isAvail := true
		for _, v := range list[i].Children {
			if v.MountPoint != "" {
				stor := model1.Storage{}
				stor.MountPoint = v.MountPoint
				stor.Size = v.FSSize
				stor.Avail = v.FSAvail
				stor.Path = v.Path
				stor.Type = v.FsType
				stor.DriveName = list[i].Name
				storage = append(storage, stor)
				isAvail = false
			}
		}

		if isAvail {
			disk.NeedFormat = false
			avail = append(avail, disk)
		}

		disk.Temperature = temp.Temperature.Current
		disk.Health = strconv.FormatBool(temp.SmartStatus.Passed)

		disks = append(disks, disk)
	}

	data := map[string]interface{}{
		"disks":   disks,
		"storage": storage,
		"avail":   avail,
	}

	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

// @Summary disk list
// @Produce  application/json
// @Accept application/json
// @Tags disk
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /disk/list [get]
func GetDisksUSBList(c *gin.Context) {
	list := service.MyService.Disk().LSBLK(false)
	data := []model1.USBDriveStatus{}
	for _, v := range list {
		if v.Tran == "usb" {
			temp := model1.USBDriveStatus{}
			temp.Model = v.Model
			temp.Name = v.Label
			if temp.Name == "" {
				temp.Name = v.Name
			}
			temp.Size = v.Size
			children := []model1.USBChildren{}
			for _, child := range v.Children {
				if len(child.MountPoint) > 0 {
					tempChildren := model1.USBChildren{}
					tempChildren.MountPoint = child.MountPoint
					tempChildren.Size, _ = strconv.ParseUint(child.FSSize, 10, 64)
					tempChildren.Avail, _ = strconv.ParseUint(child.FSAvail, 10, 64)
					tempChildren.Name = child.Label
					if len(tempChildren.Name) == 0 {
						tempChildren.Name = filepath.Base(child.MountPoint)
					}
					avail, _ := strconv.ParseUint(child.FSAvail, 10, 64)
					children = append(children, tempChildren)
					temp.Avail += avail
				}
			}

			temp.Children = children
			data = append(data, temp)
		}
	}
	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

func DeleteDisksUmount(c *gin.Context) {
	js := make(map[string]string)
	if err := c.ShouldBind(&js); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err.Error()})
		return
	}

	// requires password from user to confirm the action
	if claims, err := jwt.ParseToken(c.GetHeader("Authorization"), false); err != nil || encryption.GetMD5ByStr(js["password"]) != claims.Password {
		c.JSON(http.StatusUnauthorized, model.Result{Success: common_err.PWD_INVALID, Message: common_err.GetMsg(common_err.PWD_INVALID)})
		return
	}

	path := js["path"]

	if len(path) == 0 {
		c.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
		return
	}

	if _, ok := diskMap[path]; ok {
		c.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.DISK_BUSYING, Message: common_err.GetMsg(common_err.DISK_BUSYING)})
		return
	}

	diskInfo := service.MyService.Disk().GetDiskInfo(path)
	for _, v := range diskInfo.Children {
		if output, err := service.MyService.Disk().UmountPointAndRemoveDir(v.Path); err != nil {
			c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.REMOVE_MOUNT_POINT_ERROR, Message: output})
			return
		}

		// delete data
		service.MyService.Disk().DeleteMountPoint(v.Path, v.MountPoint)

		if err := service.MyService.Shares().DeleteShare(v.MountPoint); err != nil {
			logger.Error("error when deleting share by mount point", zap.Error(err), zap.String("mount point", v.MountPoint))
		}
	}

	service.MyService.Disk().RemoveLSBLKCache()

	// send notify to client
	go func() {
		message := map[string]interface{}{
			"data": StorageMessage{
				Action: "REMOVED",
				Path:   path,
				Volume: "",
				Size:   0,
				Type:   "",
			},
		}

		if err := service.MyService.Notify().SendNotify(messagePathStorageStatus, message); err != nil {
			logger.Error("error when sending notification", zap.Error(err), zap.String("message path", messagePathStorageStatus), zap.Any("message", message))
		}
	}()

	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: path})
}

func DeleteDiskUSB(c *gin.Context) {
	js := make(map[string]string)
	if err := c.ShouldBind(&js); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err.Error()})
		return
	}
	mountPoint := js["mount_point"]
	if file.CheckNotExist(mountPoint) {
		c.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.DIR_NOT_EXISTS, Message: common_err.GetMsg(common_err.DIR_NOT_EXISTS)})
		return
	}
	service.MyService.Disk().UmountUSB(mountPoint)
	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: mountPoint})
}

// @Summary get disk list
// @Produce  application/json
// @Accept application/json
// @Tags disk
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /disk/lists [get]
func GetPlugInDisks(c *gin.Context) {
	list := service.MyService.Disk().LSBLK(true)
	var result []*disk.UsageStat
	for _, item := range list {
		result = append(result, service.MyService.Disk().GetDiskInfoByPath(item.Path))
	}
	c.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: result})
}

// @Summary disk detail
// @Produce  application/json
// @Accept application/json
// @Tags disk
// @Security ApiKeyAuth
// @Param  path query string true "for example /dev/sda"
// @Success 200 {string} string "ok"
// @Router /disk/info [get]
func GetDiskInfo(c *gin.Context) {
	path := c.Query("path")
	if len(path) == 0 {
		c.JSON(http.StatusOK, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	m := service.MyService.Disk().GetDiskInfo(path)
	c.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: m})
}

// @Summary 获取支持的格式
// @Produce  application/json
// @Accept application/json
// @Tags disk
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /disk/type [get]
func FormatDiskType(c *gin.Context) {
	strArr := [4]string{"fat32", "ntfs", "ext4", "exfat"}
	c.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: strArr})
}

// @Summary 删除分区
// @Produce  application/json
// @Accept multipart/form-data
// @Tags disk
// @Security ApiKeyAuth
// @Param  path formData string true "磁盘路径 例如/dev/sda1"
// @Success 200 {string} string "ok"
// @Router /disk/delpart [delete]
func RemovePartition(c *gin.Context) {
	js := make(map[string]string)
	if err := c.ShouldBind(&js); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err.Error()})
		return
	}
	path := js["path"]

	if len(path) == 0 {
		c.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}
	p := path[:len(path)-1]
	n := path[len(path)-1:]
	service.MyService.Disk().DelPartition(p, n)
	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

// @Summary  add storage
// @Produce  application/json
// @Accept multipart/form-data
// @Tags disk
// @Security ApiKeyAuth
// @Param  path formData string true "disk path  e.g. /dev/sda"
// @Param  serial formData string true "serial"
// @Param  name formData string true "name"
// @Param  format formData bool true "need format(true)"
// @Success 200 {string} string "ok"
// @Router /disk/storage [post]
func PostDiskAddPartition(c *gin.Context) {
	js := make(map[string]interface{})
	if err := c.ShouldBind(&js); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err.Error()})
		return
	}
	path := js["path"].(string)

	name := js["name"].(string)
	if len(name) == 0 {
		name = "Storage"
	}

	format := js["format"].(bool)

	if len(path) == 0 {
		c.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
		return
	}
	if _, ok := diskMap[path]; ok {
		c.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.DISK_BUSYING, Message: common_err.GetMsg(common_err.DISK_BUSYING)})
		return
	}

	diskMap[path] = "busying"

	defer delete(diskMap, path)

	currentDisk := service.MyService.Disk().GetDiskInfo(path)
	if format {

		output, err := service.MyService.Disk().AddPartition(path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: output})
			return
		}
	}

	currentDisk = service.MyService.Disk().GetDiskInfo(path)

	for i := 0; i < len(currentDisk.Children); i++ {
		childrenName := currentDisk.Children[i].Label
		if len(childrenName) == 0 {
			childrenName = name + "_" + strconv.Itoa(i+1)
		}
		mountPoint := "/mnt/" + childrenName

		logger.Info("checking if mount point exist", zap.String("mount point", mountPoint))
		if empty, err := file.IsDirEmpty(mountPoint); err != nil {
			message := err.Error()
			logger.Error("error when trying to check if mount point is empty", zap.Error(err), zap.String("mount point", mountPoint))
			c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.NAME_NOT_AVAILABLE, Message: message})
			return
		} else if !empty {
			message := "mount point is not empty"
			logger.Error(message, zap.String("mount point", mountPoint))
			c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.NAME_NOT_AVAILABLE, Message: message})
			return
		}

		if err := service.MyService.Disk().MountDisk(currentDisk.Children[i].Path, mountPoint); err != nil {
			c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

		m := model2.Volume{}
		m.MountPoint = mountPoint
		m.Path = currentDisk.Children[i].Path
		m.UUID = currentDisk.Children[i].UUID
		m.State = 0
		m.CreatedAt = time.Now().Unix()
		service.MyService.Disk().SaveMountPoint(m)
		// mount dir
	}

	service.MyService.Disk().RemoveLSBLKCache()

	// send notify to client
	go func() {
		message := map[string]interface{}{
			"data": StorageMessage{
				Action: "ADDED",
				Path:   currentDisk.Children[0].Path,
				Volume: "/mnt/",
				Size:   currentDisk.Children[0].Size,
				Type:   currentDisk.Children[0].Tran,
			},
		}

		if err := service.MyService.Notify().SendNotify(messagePathStorageStatus, message); err != nil {
			logger.Error("error when sending notification", zap.Error(err), zap.String("message path", messagePathStorageStatus), zap.Any("message", message))
		}
	}()

	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

// @Param  pwd formData string true "user password"
// @Param  volume formData string true "mount point"
// @Success 200 {string} string "ok"
// @Router /disk/format [post]
func PostDiskFormat(c *gin.Context) {
	js := make(map[string]string)
	if err := c.ShouldBind(&js); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err.Error()})
		return
	}

	// requires password from user to confirm the action
	if claims, err := jwt.ParseToken(c.GetHeader("Authorization"), false); err != nil || encryption.GetMD5ByStr(js["password"]) != claims.Password {
		c.JSON(http.StatusUnauthorized, model.Result{Success: common_err.PWD_INVALID, Message: common_err.GetMsg(common_err.PWD_INVALID)})
		return
	}

	path := js["path"]
	t := "ext4"
	volume := js["volume"]

	if len(path) == 0 || len(t) == 0 {
		c.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
		return
	}
	if _, ok := diskMap[path]; ok {
		c.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.DISK_BUSYING, Message: common_err.GetMsg(common_err.DISK_BUSYING)})
		return
	}
	diskMap[path] = "busying"
	service.MyService.Disk().UmountPointAndRemoveDir(path)

	_, err := service.MyService.Disk().FormatDisk(path, t)
	if err != nil {
		delete(diskMap, path)
		c.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.FORMAT_ERROR, Message: common_err.GetMsg(common_err.FORMAT_ERROR)})
	}

	service.MyService.Disk().MountDisk(path, volume)
	service.MyService.Disk().RemoveLSBLKCache()
	delete(diskMap, path)
	c.JSON(common_err.SUCCESS, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

// @Summary remove mount point
// @Produce  application/json
// @Accept multipart/form-data
// @Tags disk
// @Security ApiKeyAuth
// @Param  path formData string true "e.g. /dev/sda1"
// @Param  mount_point formData string true "e.g. /mnt/volume1"
// @Param  pwd formData string true "user password"
// @Success 200 {string} string "ok"
// @Router /disk/umount [post]
func PostDiskUmount(c *gin.Context) {
	js := make(map[string]string)
	if err := c.ShouldBind(&js); err != nil {
		c.JSON(http.StatusBadRequest, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err.Error()})
		return
	}

	// requires password from user to confirm the action
	if claims, err := jwt.ParseToken(c.GetHeader("Authorization"), false); err != nil || encryption.GetMD5ByStr(js["password"]) != claims.Password {
		c.JSON(http.StatusUnauthorized, model.Result{Success: common_err.PWD_INVALID, Message: common_err.GetMsg(common_err.PWD_INVALID)})
		return
	}

	path := js["path"]
	mountPoint := js["volume"]

	if len(path) == 0 || len(mountPoint) == 0 {
		c.JSON(common_err.CLIENT_ERROR, model.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
		return
	}

	if _, ok := diskMap[path]; ok {
		c.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.DISK_BUSYING, Message: common_err.GetMsg(common_err.DISK_BUSYING)})
		return
	}

	if output, err := service.MyService.Disk().UmountPointAndRemoveDir(path); err != nil {
		c.JSON(http.StatusInternalServerError, model.Result{Success: common_err.REMOVE_MOUNT_POINT_ERROR, Message: output})
		return
	}

	// delete data
	service.MyService.Disk().DeleteMountPoint(path, mountPoint)
	service.MyService.Disk().RemoveLSBLKCache()

	// send notify to client
	go func() {
		message := map[string]interface{}{
			"data": StorageMessage{
				Action: "REMOVED",
				Path:   path,
				Volume: mountPoint,
				Size:   0,
				Type:   "",
			},
		}

		if err := service.MyService.Notify().SendNotify(messagePathStorageStatus, message); err != nil {
			logger.Error("error when sending notification", zap.Error(err), zap.String("message path", messagePathStorageStatus), zap.Any("message", message))
		}
	}()

	c.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

// @Summary confirm delete disk
// @Produce  application/json
// @Accept application/json
// @Tags disk
// @Security ApiKeyAuth
// @Param  id path string true "id"
// @Success 200 {string} string "ok"
// @Router /disk/remove/{id} [delete]
func DeleteDisk(c *gin.Context) {
	id := c.Param("id")
	service.MyService.Disk().DeleteMount(id)
	c.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

// @Summary check mount point
// @Produce  application/json
// @Accept application/json
// @Tags disk
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /disk/init [get]
func GetDiskCheck(c *gin.Context) {
	dbList := service.MyService.Disk().GetSerialAll()
	list := service.MyService.Disk().LSBLK(true)

	mapList := make(map[string]string)

	for _, v := range list {
		mapList[v.Serial] = "1"
	}

	for _, v := range dbList {
		if _, ok := mapList[v.UUID]; !ok {
			// disk undefind
			c.JSON(common_err.SERVICE_ERROR, model.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: "disk undefind"})
			return
		}
	}

	c.JSON(http.StatusOK, model.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

func isDiskSupported(d *model1.LSBLKModel) bool {
	return d.Tran == "sata" ||
		d.Tran == "nvme" ||
		d.Tran == "spi" ||
		d.Tran == "sas" ||
		strings.Contains(d.SubSystems, "virtio") ||
		strings.Contains(d.SubSystems, "block:scsi:vmbus:acpi") || // Microsoft Hyper-V
		(d.Tran == "ata" && d.Type == "disk")
}
