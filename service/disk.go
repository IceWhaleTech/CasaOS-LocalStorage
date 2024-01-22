package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/IceWhaleTech/CasaOS-Common/utils/constants"
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
	v2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2"
	"github.com/moby/sys/mountinfo"

	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/service/v2/fs"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DiskService interface {
	EnsureDefaultMergePoint() bool
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
	UmountPointAndRemoveDir(m model.LSBLKModel) error
	UmountUSB(path string) error

	UpdateMountPointInDB(m model2.Volume) error
	DeleteMountPointFromDB(path, mountPoint string) error
	GetSerialAllFromDB() ([]model2.Volume, error)
	SaveMountPointToDB(m model2.Volume) error
	InitCheck()
	GetSystemDf() (model.DFDiskSpace, error)
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

func (d *diskService) EnsureDefaultMergePoint() bool {
	mountPoint := common.DefaultMountPoint
	sourceBasePath := constants.DefaultFilePath

	existingMerges, err := MyService.LocalStorage().GetMergeAllFromDB(&mountPoint)
	if err != nil {
		panic(err)
	}

	// check if /DATA is already a merge point
	if len(existingMerges) > 0 {
		if len(existingMerges) > 1 {
			logger.Error("more than one merge point with the same mount point found", zap.String("mount point", mountPoint))
		}
		config.ServerInfo.EnableMergerFS = "true"
		return true
	}

	merge := &model2.Merge{
		FSType:         fs.MergerFSFullName,
		MountPoint:     mountPoint,
		SourceBasePath: &sourceBasePath,
	}
	if err := MyService.LocalStorage().CreateMerge(merge); err != nil {
		if errors.Is(err, v2.ErrMergeMountPointAlreadyExists) {
			logger.Info(err.Error(), zap.String("mount point", mountPoint))
		} else if errors.Is(err, v2.ErrMountPointIsNotEmpty) {
			logger.Error("Mount point "+mountPoint+" is not empty", zap.String("mount point", mountPoint))
			return false
		} else {
			panic(err)
		}
	}

	// mounts, err := MyService.LocalStorage().GetMounts(codegen.GetMountsParams{})
	// if err != nil {
	// 	logger.Error("failed to get mount list from system", zap.Error(err))
	// 	return false
	// }
	// isExist := false
	// for _, v := range mounts {
	// 	if v.MountPoint == mountPoint {
	// 		config.ServerInfo.EnableMergerFS = "true"
	// 		isExist = true
	// 		merge.SourceBasePath = v.Source
	// 		break
	// 	}
	// }

	// if !isExist {
	// 	if err := MyService.LocalStorage().CreateMerge(merge); err != nil {
	// 		if errors.Is(err, v2.ErrMergeMountPointAlreadyExists) {
	// 			logger.Info(err.Error(), zap.String("mount point", mountPoint))
	// 		} else if errors.Is(err, v2.ErrMountPointIsNotEmpty) {
	// 			logger.Error("Mount point "+mountPoint+" is not empty", zap.String("mount point", mountPoint))
	// 			return false
	// 		} else {
	// 			panic(err)
	// 		}
	// 	}
	// }

	if err := MyService.LocalStorage().CreateMergeInDB(merge); err != nil {
		panic(err)
	}
	config.ServerInfo.EnableMergerFS = "true"
	return true
}
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
		if err := Cache.Add(key, m, time.Minute*10); err != nil {
			//logger.Error("failed to add cache", zap.Error(err), zap.String("key", key))
		}
		return m
	}

	err := json2.Unmarshal(buf, &m)
	if err != nil {
		//logger.Error("failed to unmarshal json", zap.Error(err), zap.String("json", string(buf)))
	}
	if !reflect.DeepEqual(m, model.SmartctlA{}) {
		if err := Cache.Add(key, m, time.Hour*24); err != nil {
			//logger.Error("failed to add cache", zap.Error(err), zap.String("key", key))
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
func (d *diskService) UmountPointAndRemoveDir(m model.LSBLKModel) error {
	if len(m.MountPoint) > 0 {
		if err := mount.UmountByMountPoint(m.MountPoint); err != nil {
			logger.Error("error when umounting partition", zap.Error(err), zap.String("path", m.Path), zap.String("mount point", m.MountPoint))
			return err
		}
		if err := file.RMDir(m.MountPoint); err != nil {
			logger.Error("error when removing mount point directory", zap.Error(err), zap.String("path", m.Path), zap.String("mount point", m.MountPoint))
			return err
		}
	}
	for _, p := range m.Children {
		if len(p.MountPoint) > 0 {

			if err := mount.UmountByMountPoint(p.MountPoint); err != nil {
				logger.Error("error when umounting partition", zap.Error(err), zap.String("path", p.Path), zap.String("mount point", p.MountPoint))
				return err
			}
			if err := file.RMDir(p.MountPoint); err != nil {
				logger.Error("error when removing mount point directory", zap.Error(err), zap.String("path", p.Path), zap.String("mount point", p.MountPoint))
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

	result := make([]model.LSBLKModel, 0)

	for _, blk := range blkList {

		if blk.Type == "loop" || blk.RO {
			continue
		}

		fsused = 0

		var blkChildren []model.LSBLKModel
		smart := MyService.Disk().SmartCTL(blk.Path)
		for _, child := range blk.Children {
			if child.RM {

				// if strings.ToLower(strings.TrimSpace(child.State)) != "ok" {
				// 	health = false
				// }
				f, _ := strconv.ParseUint(child.FSUsed.String(), 10, 64)
				fsused += f
			}
			blkChildren = append(blkChildren, child)
		}
		if smart.SmartStatus.Passed {
			blk.Health = "OK"
		} else {
			for _, v := range smart.Smartctl.Messages {
				if strings.Contains(v.String, "STANDBY") {
					blk.Health = "OK"
					break
				}
			}
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
	}

	if len(result) > 0 {
		Cache.Set(key, result, time.Second*100)
	}

	return result
}

func (d *diskService) GetDiskInfo(path string) model.LSBLKModel {

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

	// return "", partition.ProbePartition(path)
	return "", nil
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
	f := model2.Volume{MountPoint: mountPoint}
	if len(partitions) > 0 {
		f.UUID = partitions[0].LSBLKProperties[`UUID`]
		logger.Info("trying to delete volume by path and mount point", zap.String("path", path), zap.String("mount point", mountPoint), zap.Any("uuid", partitions[0].LSBLKProperties[`UUID`]), zap.Any("partitons", partitions))
	}

	result := d.db.Where(&f).Limit(1).Find(&existingVolumes)
	logger.Info("result", zap.Any("result", result))
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
			if blkChild.MountPoint == m {
				continue
			}
			logger.Info("trying to re-mount...", zap.String("path", blkChild.Path), zap.String("mount point", m))
			// mount point check
			mountPoint := m
			mount.UmountByMountPoint(m)
			dir, _ := ioutil.ReadDir(m)
			if len(dir) > 0 {
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
		if !isMount && len(v.MountPoint) > 0 {
			isMount = true
			avail, _ := strconv.ParseUint(v.FSAvail.String(), 10, 64)
			status.Avail += avail
		}

		if isMount {
			statusList = append(statusList, status)
		}
	}
	return statusList
}

func (d *diskService) InitCheck() {
	time.Sleep(time.Second * 5)
	var fileName string = "local-storage.json"
	diskMap := make(map[string]model.LSBLKModel)
	diskMapNew := make(map[string]model.LSBLKModel)
	diskTempFilePath := filepath.Join(config.AppInfo.DBPath, fileName)
	if file.Exists(diskTempFilePath) {
		tempData := file.ReadFullFile(diskTempFilePath)
		err := json.Unmarshal(tempData, &diskMap)
		if err != nil {
			os.Remove(diskTempFilePath)
		}
	}

	diskList := MyService.Disk().LSBLK(false)
	for _, v := range diskList {
		if IsDiskSupported(v) {
			if _, ok := diskMap[v.Serial]; !ok {
				properties := common.AdditionalProperties(v)
				eventModel := message_bus.Event{
					SourceID:   "local-storage",
					Name:       "local-storage:disk:added",
					Properties: properties,
				}
				// add UI properties to applicable events so that CasaOS UI can render it
				event := common.EventAdapterWithUIProperties(&eventModel)

				bk := false
				for _, k := range v.Children {
					if k.MountPoint == "/" {
						bk = true
						break
					}
					for _, s := range k.Children {
						if s.MountPoint == "/" {
							bk = true
							break
						}
					}
					if bk {
						break
					}
				}
				if bk {
					continue
				}

				logger.Info("disk added", zap.Any("eventModel", eventModel))

				response, err := MyService.MessageBus().PublishEventWithResponse(context.Background(), event.SourceID, event.Name, event.Properties)
				if err != nil {
					logger.Error("failed to publish event to message bus", zap.Error(err), zap.Any("event", event))
					continue
				}

				if response.StatusCode() != http.StatusOK {
					logger.Error("failed to publish event to message bus", zap.String("status", response.Status()), zap.Any("response", response))
				}

			}
			diskMapNew[v.Serial] = v
		}
	}
	for k, v := range diskMap {
		if _, ok := diskMapNew[k]; !ok {
			logger.Info("disk removed", zap.Any("disk", v))
			properties := common.AdditionalProperties(v)
			eventModel := message_bus.Event{
				SourceID:   "local-storage",
				Name:       "local-storage:disk:removed",
				Properties: properties,
			}
			event := common.EventAdapterWithUIProperties(&eventModel)
			logger.Info("InitCheck disk removed", zap.Any("eventModel", eventModel))
			response, err := MyService.MessageBus().PublishEventWithResponse(context.Background(), event.SourceID, event.Name, event.Properties)
			if err != nil {
				logger.Error("failed to publish event to message bus", zap.Error(err), zap.Any("event", event))
			}

			if response.StatusCode() != http.StatusOK {
				logger.Error("failed to publish event to message bus", zap.String("status", response.Status()), zap.Any("response", response))
			}
		}
	}
	data, err := json.Marshal(diskMapNew)
	if err != nil {
		return
	}
	file.WriteToPath(data, config.AppInfo.DBPath, fileName)

}

func (d *diskService) GetSystemDf() (model.DFDiskSpace, error) {
	out, err := exec.Command("df", "-kPT").Output()
	if err != nil {
		log.Fatal(err)
	}

	outputStr := string(out)
	// 按行分割字符串
	lines := strings.Split(outputStr, "\n")
	// 忽略第一行（标题行）
	lines = lines[1:]
	// 遍历每一行，解析文件信息
	for _, line := range lines {
		// 分割行，获取各个字段
		fields := strings.Fields(line)
		// 如果行为空，则跳过
		if len(fields) == 0 {
			continue
		}
		if len(fields) == 7 && fields[6] == "/" {
			m := model.DFDiskSpace{
				FileSystem: fields[0],
				Type:       fields[1],

				UsePercent: fields[5],
				MountedOn:  fields[6],
			}
			b, _ := strconv.ParseInt(fields[2], 10, 64)
			u, _ := strconv.ParseInt(fields[3], 10, 64)
			a, _ := strconv.ParseInt(fields[4], 10, 64)
			m.Blocks = strconv.FormatInt(b*1024, 10)
			m.Used = strconv.FormatInt(u*1024, 10)
			m.Available = strconv.FormatInt(a*1024, 10)
			return m, nil
		} else {
			continue
		}
	}
	return model.DFDiskSpace{}, errors.New("not found")
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
		strings.Contains(d.SubSystems, "block:mmc:mmc_host:pci") ||
		strings.Contains(d.SubSystems, "block:mmc:mmc_host:platform") ||
		strings.Contains(d.SubSystems, "block:scsi:pci") || d.Tran == "usb"
}
func IsFormatSupported(d model.LSBLKModel) bool {
	if d.FsType == "vfat" || d.FsType == "ext4" || d.FsType == "ext3" || d.FsType == "ext2" || d.FsType == "exfat" || d.FsType == "ntfs-3g" || d.FsType == "iso9660" {
		return true
	}
	return false
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
