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
	"github.com/IceWhaleTech/CasaOS-LocalStorage/pkg/utils/command"

	model2 "github.com/IceWhaleTech/CasaOS-LocalStorage/service/model"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DiskService interface {
	AddPartition(path string) string
	CheckSerialDiskMount()
	DeleteMount(id string)
	DeleteMountPoint(path, mountPoint string)
	DelPartition(path, num string) ([]string, error)
	FormatDisk(path, format string) ([]string, error)
	GetDiskInfo(path string) model.LSBLKModel
	GetDiskInfoByPath(path string) *disk.UsageStat
	GetPlugInDisk() ([]string, error)
	GetSerialAll() []model2.Mount
	LSBLK(isUseCache bool) []model.LSBLKModel
	MountDisk(path, volume string) error
	RemoveLSBLKCache()
	SaveMountPoint(m model2.Mount)
	SmartCTL(path string) model.SmartctlA
	UmountPointAndRemoveDir(path string) ([]string, error)
	UmountUSB(path string) error
	UpdateMountPoint(m model2.Mount)
}
type diskService struct {
	db *gorm.DB
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
	str := command.ExecSmartCTLByPath(path)
	if str == nil {
		logger.Error("failed to  exec shell - smartctl exec error")
		Cache.Add(key, m, time.Minute*10)
		return m
	}

	err := json2.Unmarshal([]byte(str), &m)
	if err != nil {
		logger.Error("Failed to unmarshal json", zap.Error(err))
	}
	if !reflect.DeepEqual(m, model.SmartctlA{}) {
		Cache.Add(key, m, time.Hour*24)
	}
	return m
}

// 通过脚本获取外挂磁盘
func (d *diskService) GetPlugInDisk() ([]string, error) {
	return command.ExecResultStrArray("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;GetPlugInDisk")
}

// 格式化硬盘
func (d *diskService) FormatDisk(path, format string) ([]string, error) {
	return command.ExecResultStrArray("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;FormatDisk " + path + " " + format)
}

// 移除挂载点,删除目录
func (d *diskService) UmountPointAndRemoveDir(path string) ([]string, error) {
	return command.ExecResultStrArray("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;UMountPointAndRemoveDir " + path)
}

// 删除分区
func (d *diskService) DelPartition(path, num string) ([]string, error) {
	return command.ExecResultStrArray("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;DelPartition " + path + " " + num)
}

// part
func (d *diskService) AddPartition(path string) string {
	command.ExecResultStrArray("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;AddPartition " + path)
	return ""
}

func (d *diskService) AddAllPartition(path string) {
}

// 获取硬盘详情
func (d *diskService) GetDiskInfoByPath(path string) *disk.UsageStat {
	diskInfo, err := disk.Usage(path + "1")
	if err != nil {
		fmt.Println(err)
	}
	diskInfo.UsedPercent, _ = strconv.ParseFloat(fmt.Sprintf("%.1f", diskInfo.UsedPercent), 64)
	diskInfo.InodesUsedPercent, _ = strconv.ParseFloat(fmt.Sprintf("%.1f", diskInfo.InodesUsedPercent), 64)
	return diskInfo
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

func (d *diskService) MountDisk(path, volume string) error {
	r, err := command.ExecResultStr("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;do_mount " + path + " " + volume)
	if err != nil {
		return err
	}

	fmt.Println(r)
	return nil
}

func (d *diskService) SaveMountPoint(m model2.Mount) {
	var existing model2.Mount

	d.db.Where(&model2.Mount{UUID: m.UUID}).First(&existing)

	m.ID = existing.ID

	d.db.Save(&m)
}

func (d *diskService) UpdateMountPoint(m model2.Mount) {
	d.db.Model(&model2.Mount{}).Where("uui = ?", m.UUID).Update("mount_point", m.MountPoint)
}

func (d *diskService) DeleteMount(id string) {
	d.db.Delete(&model2.Mount{}).Where("id = ?", id)
}

func (d *diskService) DeleteMountPoint(path, mountPoint string) {
	d.db.Where("path = ? AND mount_point = ?", path, mountPoint).Delete(&model2.Mount{})

	command.OnlyExec("source " + config.AppInfo.ShellPath + "/local-storage-helper.sh ;do_umount " + path)
}

func (d *diskService) GetSerialAll() []model2.Mount {
	var m []model2.Mount
	d.db.Find(&m)
	return m
}

func (d *diskService) CheckSerialDiskMount() {
	logger.Info("Checking serial disk mount...")

	// check mount point
	dbList := d.GetSerialAll()

	list := d.LSBLK(true)
	mountPoint := make(map[string]string, len(dbList))
	// remount
	for _, v := range dbList {
		mountPoint[v.UUID] = v.MountPoint
	}
	for _, v := range list {
		output, err := command.ExecEnabledSMART(v.Path)
		if err != nil {
			if output != nil {
				logger.Error("failed to enable S.M.A.R.T: "+string(output), zap.Error(err), zap.String("path", v.Path))
			} else {
				logger.Error("failed to enable S.M.A.R.T", zap.Error(err), zap.String("path", v.Path))
			}
		}

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
					if err := d.MountDisk(h.Path, volume); err != nil {
						logger.Error("Failed to mount disk", zap.Error(err), zap.String("path", h.Path), zap.String("volume", volume))
					}

					if volume != m {
						ms := model2.Mount{}
						ms.UUID = v.UUID
						ms.MountPoint = volume
						d.UpdateMountPoint(ms)
					}
				}
			}
		}
	}
	d.RemoveLSBLKCache()
}

func NewDiskService(db *gorm.DB) DiskService {
	return &diskService{db: db}
}
