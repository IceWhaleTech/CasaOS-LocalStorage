package service

import (
	json2 "encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/fstab"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/utils/command"
	"github.com/moby/sys/mountinfo"

	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DiskService interface {
	AddPartition(path string) (string, error)
	CheckSerialDiskMount()
	DeleteMountPoint(path, mountPoint string)
	FormatDisk(path, format string) ([]string, error)
	GetDiskInfo(path string) model.LSBLKModel
	GetPersistentType(path string) string
	GetSerialAll() []model2.Volume
	GetUSBDriveStatusList() []model.USBDriveStatus
	LSBLK(isUseCache bool) []model.LSBLKModel
	MountDisk(path, volume string) (string, error)
	RemoveLSBLKCache()
	SaveMountPoint(m model2.Volume)
	SmartCTL(path string) model.SmartctlA
	UmountPointAndRemoveDir(path string) (string, error)
	UmountUSB(path string) error
	UpdateMountPoint(m model2.Volume)
}

type diskService struct {
	db *gorm.DB
}

const (
	PersistentTypeNone   = "none"
	PersistentTypeFStab  = "fstab"
	PersistentTypeCasaOS = "casaos"
)

func (d *diskService) RemoveLSBLKCache() {
	key := "system_lsblk"
	Cache.Delete(key)
}

func (d *diskService) UmountUSB(path string) error {
	_, err := command.ExecResultStr("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;UDEVILUmount " + path)
	if err != nil {
		return err
	}

	return nil
}

func (d *diskService) SmartCTL(path string) model.SmartctlA {
	key := "system_smart_" + path
	if result, ok := Cache.Get(key); ok {

		res, ok := result.(model.SmartctlA)
		if ok {
			return res
		}
	}
	var m model.SmartctlA
	buf := command.ExecSmartCTLByPath(path)
	if buf == nil {
		logger.Error("failed to exec shell - smartctl exec error")
		Cache.Add(key, m, time.Minute*10)
		return m
	}

	err := json2.Unmarshal(buf, &m)
	if err != nil {
		logger.Error("failed to unmarshal json", zap.Error(err), zap.String("json", string(buf)))
	}
	if !reflect.DeepEqual(m, model.SmartctlA{}) {
		Cache.Add(key, m, time.Hour*24)
	}
	return m
}

// 格式化硬盘
func (d *diskService) FormatDisk(path, format string) ([]string, error) {
	return command.ExecResultStrArray("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;FormatDisk " + path + " " + format)
}

// 移除挂载点,删除目录
func (d *diskService) UmountPointAndRemoveDir(path string) (string, error) {
	return command.OnlyExec("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;UMountPointAndRemoveDir " + path)
}

// part
func (d *diskService) AddPartition(path string) (string, error) {
	return command.OnlyExec("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;AddPartition " + path)
}

// get disk details
func (d *diskService) LSBLK(isUseCache bool) []model.LSBLKModel {
	key := "system_lsblk"
	var n []model.LSBLKModel

	if result, ok := Cache.Get(key); ok && isUseCache {

		res, ok := result.([]model.LSBLKModel)
		if ok {
			return res
		}
	}

	str := command.ExecLSBLK()
	if str == nil {
		logger.Error("Failed to exec shell - lsblk exec error")
		return nil
	}
	var m []model.LSBLKModel
	err := json2.Unmarshal([]byte(gjson.Get(string(str), "blockdevices").String()), &m)
	if err != nil {
		logger.Error("Failed to unmarshal json", zap.Error(err))
	}

	var c []model.LSBLKModel

	var fsused uint64

	health := true
	for _, i := range m {
		if i.Type != "loop" && !i.RO {
			fsused = 0
			for _, child := range i.Children {
				if child.RM {
					output, err := command.ExecResultStr("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;GetDiskHealthState " + child.Path)
					if err != nil {
						logger.Error("Failed to exec shell", zap.Error(err))
						return nil
					}

					child.Health = strings.TrimSpace(output)
					if strings.ToLower(strings.TrimSpace(child.State)) != "ok" {
						health = false
					}
					f, _ := strconv.ParseUint(child.FSUsed, 10, 64)
					fsused += f
				} else {
					health = false
				}
				c = append(c, child)
			}
			// i.Format = strings.TrimSpace(command.ExecResultStr("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;GetDiskType " + i.Path))
			if health {
				i.Health = "OK"
			}
			i.FSUsed = strconv.FormatUint(fsused, 10)
			i.Children = c
			if fsused > 0 {
				i.UsedPercent, err = strconv.ParseFloat(fmt.Sprintf("%.4f", float64(fsused)/float64(i.Size)), 64)
				if err != nil {
					logger.Error("Failed to parse float", zap.Error(err))
				}
			}
			n = append(n, i)
			health = true
			c = []model.LSBLKModel{}
			fsused = 0
		}
	}
	if len(n) > 0 {
		Cache.Add(key, n, time.Second*100)
	}
	return n
}

func (d *diskService) GetDiskInfo(path string) model.LSBLKModel {
	str := command.ExecLSBLKByPath(path)
	if str == nil {
		logger.Error("Failed to exec shell - lsblk exec error")
		return model.LSBLKModel{}
	}

	var ml []model.LSBLKModel
	err := json2.Unmarshal([]byte(gjson.Get(string(str), "blockdevices").String()), &ml)
	if err != nil {
		logger.Error("Failed to unmarshal json", zap.Error(err))
		return model.LSBLKModel{}
	}

	m := model.LSBLKModel{}
	if len(ml) > 0 {
		m = ml[0]
	}
	return m
}

func (d *diskService) MountDisk(path, mountPoint string) (string, error) {
	logger.Info("trying to mount...", zap.String("path", path), zap.String("mountPoint", mountPoint))

	// check if path is already mounted at mountPoint
	if mountInfoList, err := mountinfo.GetMounts(func(i *mountinfo.Info) (skip bool, stop bool) {
		if i.Source == path && i.Mountpoint == mountPoint {
			return false, true
		}
		return true, false
	}); err != nil {
		logger.Error("error when trying to get mount info", zap.Error(err))
		return "", err
	} else if len(mountInfoList) > 0 {
		logger.Info("already mounted", zap.String("path", path), zap.String("mount point", mountPoint))
		return "", nil
	}

	if err := file.IsNotExistMkDir(mountPoint); err != nil {
		logger.Error("error when checking if mount point already exists, or when creating the mount point if it does not exists", zap.Error(err), zap.String("mount point", mountPoint))
		return "", err
	}

	return command.OnlyExec("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;do_mount " + path + " " + mountPoint)
}

func (d *diskService) SaveMountPoint(m model2.Volume) {
	var existing model2.Volume

	d.db.Where(&model2.Volume{UUID: m.UUID}).First(&existing)

	m.ID = existing.ID

	d.db.Save(&m)
}

func (d *diskService) UpdateMountPoint(m model2.Volume) {
	d.db.Model(&model2.Volume{}).Where("uuid = ?", m.UUID).Update("mount_point", m.MountPoint)
}

func (d *diskService) DeleteMountPoint(path, mountPoint string) {
	if mountInfoList, err := mountinfo.GetMounts(func(i *mountinfo.Info) (skip bool, stop bool) {
		if i.Source == path && i.Mountpoint == mountPoint {
			return false, true
		}
		return true, false
	}); err != nil {
		logger.Error("failed to checking for existing mount", zap.Error(err), zap.String("path", path), zap.String("mount point", mountPoint))
		return
	} else if len(mountInfoList) == 0 {
		logger.Info("already umounted", zap.String("path", path), zap.String("mount point", mountPoint))
	} else {
		output, err := command.OnlyExec("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;do_umount " + path)
		if err != nil {
			logger.Error(output, zap.Error(err), zap.String("path", path), zap.String("mount point", mountPoint))
			return
		}

		logger.Info(output, zap.String("path", path), zap.String("mount point", mountPoint))
	}

	var existingVolumes []model2.Volume

	if result := d.db.Where(&model2.Volume{Path: path, MountPoint: mountPoint}).Find(&existingVolumes); result.Error != nil {
		logger.Error("error when finding the volume", zap.Error(result.Error), zap.String("path", path), zap.String("mount point", mountPoint))
	} else if result.RowsAffected <= 0 {
		logger.Info("no volume found", zap.String("path", path), zap.String("mount point", mountPoint))
	} else {
		logger.Info("deleting volume from database", zap.String("path", path), zap.String("mount point", mountPoint))
		d.db.Delete(&existingVolumes)
	}
}

func (d *diskService) GetSerialAll() []model2.Volume {
	var m []model2.Volume
	d.db.Find(&m)
	return m
}

func (d *diskService) GetPersistentType(path string) string {
	// check if path is in database
	var m model2.Volume

	if result := d.db.Where(&model2.Volume{Path: path}).Limit(1).Find(&m); result.Error != nil {
		logger.Error("error when finding the volume by path in database", zap.Error(result.Error), zap.String("path", path))
	} else if result.RowsAffected > 0 {
		return PersistentTypeCasaOS
	}

	// check if it is in fstab
	if entry, err := fstab.Get().GetEntryBySource(path); err != nil {
		logger.Error("error when finding the volume by path in fstab", zap.Error(err), zap.String("path", path))
	} else if entry != nil {
		return PersistentTypeFStab
	}

	// return none if not found
	return PersistentTypeNone
}

func (d *diskService) CheckSerialDiskMount() {
	logger.Info("Checking serial disk mount...")

	// check mount point
	dbList := d.GetSerialAll()

	list := d.LSBLK(true)
	mountPointMap := make(map[string]string, len(dbList))

	defer d.RemoveLSBLKCache()

	// remount
	for _, v := range dbList {
		logger.Info("previously persisted mount point", zap.Any("volume", v))
		mountPointMap[v.UUID] = v.MountPoint
	}

	for _, currentDisk := range list {
		output, err := command.ExecEnabledSMART(currentDisk.Path)
		if err != nil {
			if output != nil {
				logger.Error("failed to enable S.M.A.R.T: "+string(output), zap.Error(err), zap.String("path", currentDisk.Path))
			} else {
				logger.Error("failed to enable S.M.A.R.T", zap.Error(err), zap.String("path", currentDisk.Path))
			}
		}

		for _, blkChild := range currentDisk.Children {
			m, ok := mountPointMap[blkChild.UUID]
			if !ok {
				continue
			}

			logger.Info("trying to re-mount...", zap.String("path", blkChild.Path), zap.String("mount point", m))

			// mount point check
			mountPoint := m
			if !file.CheckNotExist(m) {
				for i := 0; file.CheckNotExist(mountPoint); i++ {
					mountPoint = m + strconv.Itoa(i+1)
				}
				logger.Info("mount point already exists, using new mount point", zap.String("path", blkChild.Path), zap.String("mount point", mountPoint))
			}

			if output, err := d.MountDisk(blkChild.Path, mountPoint); err != nil {
				logger.Error(output, zap.Error(err), zap.String("path", blkChild.Path), zap.String("volume", mountPoint))
			}

			if mountPoint != m {
				ms := model2.Volume{
					UUID:       currentDisk.UUID,
					MountPoint: mountPoint,
				}
				d.UpdateMountPoint(ms)
			}
		}
	}
}

func (d *diskService) GetUSBDriveStatusList() []model.USBDriveStatus {
	blockList := d.LSBLK(false)
	statusList := []model.USBDriveStatus{}
	for _, v := range blockList {
		if v.Tran != "usb" {
			continue
		}

		isMount := false
		status := model.USBDriveStatus{Model: v.Model, Name: v.Name, Size: v.Size}
		for _, child := range v.Children {
			if len(child.MountPoint) > 0 {
				isMount = true
				avail, _ := strconv.ParseUint(child.FSAvail, 10, 64)
				status.Avail += avail
			}
		}
		if isMount {
			statusList = append(statusList, status)
		}
	}
	return statusList
}

func NewDiskService(db *gorm.DB) DiskService {
	return &diskService{db: db}
}
