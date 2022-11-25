package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen/message_bus"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/common"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/config"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/fstab"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/mount"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/partition"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/utils/command"
	"github.com/moby/sys/mountinfo"

	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DiskService interface {
	AddPartition(path string) error
	DeletePartition(path string) error
	CheckSerialDiskMount()
	FormatDisk(path string) error
	GetDiskInfo(path string) model.LSBLKModel
	GetPersistentTypeByUUID(uuid string) string
	GetUSBDriveStatusList() []model.USBDriveStatus
	LSBLK(isUseCache bool) []model.LSBLKModel
	MountDisk(path, volume string) (string, error)
	RemoveLSBLKCache()
	SmartCTL(path string) model.SmartctlA
	UmountPointAndRemoveDir(path string) error
	UmountUSB(path string) error

	UpdateMountPointInDB(m model2.Volume) error
	DeleteMountPointFromDB(path, mountPoint string) error
	GetSerialAllFromDB() ([]model2.Volume, error)
	SaveMountPointToDB(m model2.Volume) error
	InitCheck()
}

type diskService struct {
	db *gorm.DB
}

const (
	PersistentTypeNone   = "none"
	PersistentTypeFStab  = "fstab"
	PersistentTypeCasaOS = "casaos"
)

var (
	ErrVolumeWithEmptyUUID = errors.New("cannot save volume with empty uuid")
	json2                  = jsoniter.ConfigCompatibleWithStandardLibrary
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
		if err := Cache.Add(key, m, time.Minute*10); err != nil {
			logger.Error("failed to add cache", zap.Error(err), zap.String("key", key))
		}
		return m
	}

	err := json2.Unmarshal(buf, &m)
	if err != nil {
		logger.Error("failed to unmarshal json", zap.Error(err), zap.String("json", string(buf)))
	}
	if !reflect.DeepEqual(m, model.SmartctlA{}) {
		if err := Cache.Add(key, m, time.Hour*24); err != nil {
			logger.Error("failed to add cache", zap.Error(err), zap.String("key", key))
		}
	}
	return m
}

// 格式化硬盘
func (d *diskService) FormatDisk(path string) error {
	// wait for partition path to be ready
	count := 5
	for count > 0 {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				time.Sleep(1 * time.Second)
				count--
				continue
			}
			logger.Error("error when checking partition path", zap.Error(err), zap.String("path", path))
			return err
		}
		break
	}

	logger.Info("formatting partition...", zap.String("path", path))
	if err := partition.FormatPartition(path); err != nil {
		logger.Error("failed to format partition", zap.Error(err), zap.String("path", path))
		return err
	}

	return nil
}

// 移除挂载点,删除目录
func (d *diskService) UmountPointAndRemoveDir(path string) error {
	logger.Info("trying to get all partitions of device...", zap.String("path", path))
	partitions, err := partition.GetPartitions(path)
	if err != nil {
		logger.Error("error when getting all partitions of device", zap.Error(err), zap.String("path", path))
		return err
	}

	for _, p := range partitions {
		if p.LSBLKProperties["MOUNTPOINT"] != "" {
			logger.Info("trying to umount partition...", zap.String("path", p.LSBLKProperties["PATH"]), zap.String("mount point", p.LSBLKProperties["MOUNTPOINT"]))
			if err := mount.UmountByMountPoint(p.LSBLKProperties["MOUNTPOINT"]); err != nil {
				logger.Error("error when umounting partition", zap.Error(err), zap.String("path", p.LSBLKProperties["PATH"]), zap.String("mount point", p.LSBLKProperties["MOUNTPOINT"]))
				return err
			}

			logger.Info("trying to remove mount point directory...", zap.String("path", p.LSBLKProperties["PATH"]), zap.String("mount point", p.LSBLKProperties["MOUNTPOINT"]))
			if err := file.RMDir(p.LSBLKProperties["MOUNTPOINT"]); err != nil {
				logger.Error("error when removing mount point directory", zap.Error(err), zap.String("path", p.LSBLKProperties["PATH"]), zap.String("mount point", p.LSBLKProperties["MOUNTPOINT"]))
				return err
			}
		}
	}

	return nil
}

// part
func (d *diskService) AddPartition(path string) error {
	logger.Info("creating partition table...", zap.String("path", path))
	if err := partition.CreatePartitionTable(path); err != nil {
		logger.Error("failed to create partition table", zap.Error(err), zap.String("path", path))
		return err
	}

	logger.Info("creating partition...", zap.String("path", path))
	partitions, err := partition.AddPartition(path)
	if err != nil {
		logger.Error("failed to create partition", zap.Error(err), zap.String("path", path))
		return err
	}

	for _, p := range partitions {
		partitionPath := p.LSBLKProperties["PATH"]

		// wait for partition path to be ready
		count := 5
		for count > 0 {
			if _, err := os.Stat(partitionPath); err != nil {
				if os.IsNotExist(err) {
					time.Sleep(1 * time.Second)
					count--
					continue
				}
				logger.Error("error when checking partition path", zap.Error(err), zap.String("path", partitionPath))
				return err
			}
			break
		}

		logger.Info("formatting partition...", zap.String("path", partitionPath))
		if err := partition.FormatPartition(partitionPath); err != nil {
			logger.Error("failed to format partition", zap.Error(err), zap.String("path", partitionPath))
			return err
		}
	}

	return nil
}

func (d *diskService) DeletePartition(path string) error {
	// check if path exists
	if !file.Exists(path) {
		return errors.New("device " + path + " does not exists")
	}

	logger.Info("trying to get all partitions of device...", zap.String("path", path))
	partitions, err := partition.GetPartitions(path)
	if err != nil {
		logger.Error("error when getting all partitions of device", zap.Error(err), zap.String("path", path))
		return err
	}

	for _, p := range partitions {

		n, err := strconv.Atoi(p.PARTXProperties["NR"])
		if err != nil {
			logger.Error("error when converting partition number to int", zap.Error(err), zap.String("path", path), zap.String("partition number", p.PARTXProperties["NR"]))
			return err
		}

		logger.Info("trying to delete partition...", zap.String("path", p.LSBLKProperties["PATH"]))
		if err := partition.DeletePartition(path, n); err != nil {
			logger.Error("error when deleting partition", zap.Error(err), zap.String("path", p.LSBLKProperties["PATH"]))
			return err
		}
	}

	return nil
}

// get disk details
func (d *diskService) LSBLK(isUseCache bool) []model.LSBLKModel {
	key := "system_lsblk"

	if isUseCache {
		if result, ok := Cache.Get(key); ok {
			if res, ok := result.([]model.LSBLKModel); ok {
				return res
			}
		}
	}

	str := command.ExecLSBLK()
	if str == nil {
		logger.Error("Failed to exec shell - lsblk exec error")
		return nil
	}

	blkList, err := ParseBlockDevices(str)
	if err != nil {
		logger.Error("Failed to parse block devices from output of lsblk", zap.Error(err))
	}

	var fsused uint64

	health := true

	result := make([]model.LSBLKModel, 0)

	for _, blk := range blkList {

		if blk.Type == "loop" || blk.RO {
			continue
		}

		fsused = 0

		var blkChildren []model.LSBLKModel
		for _, child := range blk.Children {
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
				f, _ := strconv.ParseUint(child.FSUsed.String(), 10, 64)
				fsused += f
			} else {
				health = false
			}
			blkChildren = append(blkChildren, child)
		}

		if health {
			blk.Health = "OK"
		}

		blk.FSUsed = json.Number(fmt.Sprintf("%d", fsused))
		blk.Children = blkChildren
		if fsused > 0 {
			blk.UsedPercent, err = strconv.ParseFloat(fmt.Sprintf("%.4f", float64(fsused)/float64(blk.Size)), 64)
			if err != nil {
				logger.Error("Failed to parse float", zap.Error(err))
			}
		}
		result = append(result, blk)
		health = true
	}

	if len(result) > 0 {
		Cache.Set(key, result, time.Second*100)
	}

	return result
}

func (d *diskService) GetDiskInfo(path string) model.LSBLKModel {
	logger.Info("trying to get disk info...", zap.String("path", path))

	str := command.ExecLSBLKByPath(path)
	if str == nil {
		logger.Error("Failed to exec shell - lsblk exec error")
		return model.LSBLKModel{}
	}

	blkList, err := ParseBlockDevices(str)
	if err != nil {
		logger.Error("Failed to parse block devices from output of lsblk", zap.Error(err))
		return model.LSBLKModel{}
	}

	blk := model.LSBLKModel{}
	if len(blkList) > 0 {
		blk = blkList[0]
	}
	return blk
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

	if out, err := command.OnlyExec("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;do_mount " + path + " " + mountPoint); err != nil {
		logger.Error("error when mounting", zap.Error(err), zap.String("path", path), zap.String("mount point", mountPoint), zap.String("output", string(out)))
		return out, err
	}

	return "", partition.ProbePartition(path)
}

func (d *diskService) SaveMountPointToDB(m model2.Volume) error {
	if m.UUID == "" {
		return ErrVolumeWithEmptyUUID
	}

	var existing model2.Volume

	result := d.db.Where(&model2.Volume{UUID: m.UUID}).Limit(1).Find(&existing)

	if result.Error != nil {
		logger.Error("error when querying volume by UUID", zap.Error(result.Error), zap.Any("uuid", m.UUID))
		return result.Error
	}

	if result.RowsAffected > 0 {
		m.ID = existing.ID
	}

	if result := d.db.Save(&m); result.Error != nil {
		logger.Error("error when saving volume to db", zap.Error(result.Error), zap.Any("volume", m))
		return result.Error
	}

	return nil
}

func (d *diskService) UpdateMountPointInDB(m model2.Volume) error {
	result := d.db.Model(&model2.Volume{}).Where(&model2.Volume{UUID: m.UUID}).Update("mount_point", m.MountPoint)
	if result.Error != nil {
		logger.Error("error when updating mount point in db by UUID", zap.Error(result.Error), zap.String("uuid", m.UUID), zap.String("mount point", m.MountPoint))
		return result.Error
	}

	logger.Info(strconv.Itoa(int(result.RowsAffected))+" volume(s) with mount point updated in db by UUID", zap.String("uuid", m.UUID), zap.String("mount point", m.MountPoint))

	return nil
}

func (d *diskService) DeleteMountPointFromDB(path, mountPoint string) error {
	partitions, err := partition.GetPartitions(path)
	if err != nil {
		logger.Error("error when getting partitions by path", zap.Error(err), zap.String("path", path))
		return err
	}

	if len(partitions) != 1 {
		logger.Error("there should be only 1 partition returned", zap.Any("partitions", partitions))
	}

	var existingVolumes []model2.Volume

	result := d.db.Where(&model2.Volume{UUID: partitions[0].PARTXProperties["UUID"], MountPoint: mountPoint}).Limit(1).Find(&existingVolumes)

	if result.Error != nil {
		logger.Error("error when finding the volume by path and mount point", zap.Error(result.Error), zap.String("path", path), zap.String("mount point", mountPoint))
	}

	if result.RowsAffected == 0 {
		logger.Info("no volume found by path and mount point", zap.String("path", path), zap.String("mount point", mountPoint))
		return nil
	}

	if result := d.db.Delete(&existingVolumes); result.Error != nil {
		logger.Error("error when deleting volume", zap.Error(result.Error), zap.Any("volume", existingVolumes))
		return result.Error
	}

	return nil
}

func (d *diskService) GetSerialAllFromDB() ([]model2.Volume, error) {
	var volumes []model2.Volume

	result := d.db.Find(&volumes)
	if result.Error != nil {
		logger.Error("error when querying all volumes from db", zap.Error(result.Error))
		return nil, result.Error
	}

	return volumes, nil
}

func (d *diskService) GetPersistentTypeByUUID(uuid string) string {
	// check if path is in database
	var m model2.Volume

	if result := d.db.Where(&model2.Volume{UUID: uuid}).Limit(1).Find(&m); result.Error != nil {
		logger.Error("error when finding the volume by uuid in database", zap.Error(result.Error), zap.String("uuid", uuid))
	} else if result.RowsAffected > 0 {
		return PersistentTypeCasaOS
	}

	// check if it is in fstab
	if entry, err := fstab.Get().GetEntryBySource(uuid); err != nil {
		logger.Error("error when finding the volume by uuid in fstab", zap.Error(err), zap.String("uuid", uuid))
	} else if entry != nil {
		return PersistentTypeFStab
	}

	// return none if not found
	return PersistentTypeNone
}

func (d *diskService) CheckSerialDiskMount() {
	logger.Info("Checking serial disk mount...")

	// check mount point
	dbList, err := d.GetSerialAllFromDB()
	if err != nil {
		logger.Error("error when getting all volumes from db", zap.Error(err))
		return
	}

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
				i := 1
				for {
					mountPoint = m + "-" + strconv.Itoa(i)
					if file.CheckNotExist(mountPoint) {
						break
					}
					i++
				}
				logger.Info("mount point already exists, using new mount point", zap.String("path", blkChild.Path), zap.String("mount point", mountPoint))
			}

			if output, err := d.MountDisk(blkChild.Path, mountPoint); err != nil {
				logger.Error(output, zap.Error(err), zap.String("path", blkChild.Path), zap.String("volume", mountPoint))
			}

			// obtain the actual mount path (just in case)
			partitions, err := partition.GetPartitions(blkChild.Path)
			if err != nil {
				logger.Error("error when getting partitions by path", zap.Error(err), zap.String("path", blkChild.Path))
				continue
			}

			mountPoint = partitions[0].LSBLKProperties["MOUNTPOINT"]

			if mountPoint != m {
				v := model2.Volume{
					UUID:       blkChild.UUID,
					MountPoint: mountPoint,
				}
				if err := d.UpdateMountPointInDB(v); err != nil {
					logger.Error("error when updating mount point in db", zap.Error(err), zap.Any("volume", v))
				}
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
				avail, _ := strconv.ParseUint(child.FSAvail.String(), 10, 64)
				status.Avail += avail
			}
		}
		if isMount {
			statusList = append(statusList, status)
		}
	}
	return statusList
}

func (d *diskService) InitCheck() {
	diskMap := make(map[string]model.LSBLKModel)
	diskMapNew := make(map[string]model.LSBLKModel)
	diskTempFilePath := filepath.Join(config.AppInfo.DBPath, "disk.temp")
	if file.Exists(diskTempFilePath) {
		tempData := file.ReadFullFile(diskTempFilePath)
		err := json.Unmarshal(tempData, &diskMap)
		if err != nil {
			os.Remove(diskTempFilePath)
		}
	}

	diskList := MyService.Disk().LSBLK(false)
	for _, v := range diskList {
		if v.Tran == "sata" {
			if _, ok := diskMap[v.Serial]; !ok {
				properties := make(map[string]string)
				properties["tran"] = v.Tran
				properties["size"] = strconv.FormatUint(v.Size, 10)
				properties["used"] = string(v.FSUsed)
				properties["model"] = v.Model
				properties["path"] = v.Path
				properties["children:num"] = strconv.Itoa(len(v.Children))

				for i := 0; i < len(v.Children); i++ {
					properties["children:"+strconv.Itoa(i)+":fstype"] = v.Children[i].FsType
					properties["children:"+strconv.Itoa(i)+":path"] = v.Children[i].Path
					properties["children:"+strconv.Itoa(i)+":size"] = string(v.Children[i].FSSize)
					properties["children:"+strconv.Itoa(i)+":used"] = string(v.Children[i].FSUsed)
				}

				eventModel := message_bus.Event{
					SourceID:   "local-storage",
					Name:       "local-storage:disk:removed",
					Properties: properties,
				}

				// add UI properties to applicable events so that CasaOS UI can render it
				event := common.EventAdapterWithUIProperties(&eventModel)
				time.Sleep(time.Second * 5)
				response, err := MyService.MessageBus().PublishEventWithResponse(context.Background(), event.SourceID, event.Name, event.Properties)
				if err != nil {
					logger.Error("failed to publish event to message bus", zap.Error(err), zap.Any("event", event))
				}

				if response.StatusCode() != http.StatusOK {
					logger.Error("failed to publish event to message bus", zap.String("status", response.Status()), zap.Any("response", response))
				}

			}
			diskMapNew[v.Serial] = v
		}
	}
	data, err := json.Marshal(diskMapNew)
	if err != nil {
		return
	}
	file.WriteToPath(data, config.AppInfo.DBPath, "disk.temp")

}

func NewDiskService(db *gorm.DB) DiskService {
	return &diskService{db: db}
}

func IsDiskSupported(d model.LSBLKModel) bool {
	return d.Tran == "sata" ||
		d.Tran == "nvme" ||
		d.Tran == "spi" ||
		d.Tran == "sas" ||
		strings.Contains(d.SubSystems, "virtio") ||
		strings.Contains(d.SubSystems, "block:scsi:vmbus:acpi") || // Microsoft Hyper-V
		(d.Tran == "ata" && d.Type == "disk")
}

func WalkDisk(rootBlk model.LSBLKModel, depth uint, shouldStopAt func(blk model.LSBLKModel) bool) *model.LSBLKModel {
	if shouldStopAt(rootBlk) {
		return &rootBlk
	}

	if depth == 0 {
		return nil
	}

	for _, blkChild := range rootBlk.Children {
		if blk := WalkDisk(blkChild, depth-1, shouldStopAt); blk != nil {
			return blk
		}
	}

	return nil
}

func ParseBlockDevices(str []byte) ([]model.LSBLKModel, error) {
	var blkList []model.LSBLKModel
	if err := json2.Unmarshal([]byte(jsoniter.Get(str, "blockdevices").ToString()), &blkList); err != nil {
		return nil, err
	}

	return blkList, nil
}
