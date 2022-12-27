#!/bin/bash

UDEVILUmount(){
  $sudo_cmd udevil umount -f $1
}

#获磁盘的插入路径
#param 路径 /dev/sda
GetPlugInDisk() {
  fdisk -l | grep 'Disk' | grep 'sd' | awk -F , '{print substr($1,11,3)}'
}

#格式化fat32磁盘
#param 需要格式化的目录 /dev/sda1
#param 格式
FormatDisk() {
  if [ "$2" == "fat32" ]; then
    mkfs.vfat -F 32 $1
  elif [ "$2" == "ntfs" ]; then
    mkfs.ntfs $1
  elif [ "$2" == "ext4" ]; then
    mkfs.ext4 -m 1 -F $1
  elif [ "$2" == "exfat" ]; then
    mkfs.exfat $1
  else
    mkfs.ext4 -m 1 -F $1
  fi
}

#移除挂载点,删除已挂在的文件夹
UMountPointAndRemoveDir() {
  set -e
  DEVICE=$1
  MOUNT_POINT=$(mount | grep ${DEVICE} | awk '{ print $3 }')
  if [[ -z ${MOUNT_POINT} ]]; then
    echo "Warning: ${DEVICE} is not mounted"
  else
    umount -lf ${DEVICE}
    /bin/rmdir "${MOUNT_POINT}"
  fi
}

#添加分区只有一个分区
#param 路径   /dev/sdb
#param 要挂载的目录
AddPartition() {
  set -e

  parted -s $1 mklabel gpt

  parted -s $1 mkpart primary ext4 0 100%
  P=`lsblk -r $1 | sort | grep part | head -n 1 | awk '{print $1}'`
  mkfs.ext4 -m 1 -F /dev/${P}

  partprobe $1
}

#磁盘类型
GetDiskType() {
  fdisk $1 -l | grep Disklabel | awk -F: '{print $2}'
}

# $1=sda1
# $2=volume{1}
do_mount() {
  set -e

  DEVBASE=$1
  DEVICE="${DEVBASE}"
  # See if this drive is already mounted, and if so where
  MOUNT_POINT=$(lsblk -o mountpoint -nr "${DEVICE}" | head -n 1)

  if [ -n "${MOUNT_POINT}" ]; then
    echo "${DEVICE} is already mounted at ${MOUNT_POINT}"
    exit 1
  fi

  # Get info for this drive: $ID_FS_LABEL and $ID_FS_TYPE
  DRIVE_INFO=$(blkid -o udev "${DEVICE}" | grep -i -e "ID_FS_LABEL" -e "ID_FS_TYPE") || {
    echo "${DEVICE} does not have a filesystem or it might be corrupted. Please consider format it."
    exit 1
  }

  eval "${DRIVE_INFO}"

  LABEL=$2
  if grep -q " ${LABEL} " /etc/mtab; then
    # Already in use, make a unique one
    LABEL+="-${DEVBASE}"
  fi
  DEV_LABEL="${LABEL}"

  # Use the device name in case the drive doesn't have label
  if [ -z "${DEV_LABEL}" ]; then
    DEV_LABEL="${DEVBASE}"
  fi

  MOUNT_POINT="${DEV_LABEL}"

  echo "Mount point: ${MOUNT_POINT}"

  mkdir -p "${MOUNT_POINT}"

  case ${ID_FS_TYPE} in
  vfat)
    mount -t vfat -o rw,relatime,users,gid=100,umask=000,shortname=mixed,utf8=1,flush "${DEVICE}" "${MOUNT_POINT}"
    ;;
  ext[2-4])
    mount -o noatime "${DEVICE}" "${MOUNT_POINT}"
    ;;
  exfat)
    mount -t exfat "${DEVICE}" "${MOUNT_POINT}"
    ;;
  ntfs)
    ntfs-3g "${DEVICE}" "${MOUNT_POINT}"
    ;;
  iso9660)
    mount -t iso9660 "${DEVICE}" "${MOUNT_POINT}"
    ;;
  *)
    echo "Unsupported filesystem type: ${ID_FS_TYPE}"
    /bin/rmdir "${MOUNT_POINT}"
    exit 1
    ;;
  esac
}

# $1=sda1
do_umount() {
  DEVBASE=$1
  DEVICE="${DEVBASE}"
  MOUNT_POINT=$(mount | grep ${DEVICE} | awk '{ print $3 }')

  if [[ -z ${MOUNT_POINT} ]]; then
    echo "Warning: ${DEVICE} is not mounted"
  else
    /bin/kill -9 $(lsof ${MOUNT_POINT})
    umount -l ${DEVICE}
    echo "Unmounted ${DEVICE} from ${MOUNT_POINT}"
    if [ "`ls -A ${MOUNT_POINT}`" = "" ]; then
      /bin/rm -fr "${MOUNT_POINT}"
    fi
    
    sed -i.bak "\@${MOUNT_POINT}@d" /var/log/usb-mount.track
  fi

}

USB_Start_Auto() {
  ((EUID)) && sudo_cmd="sudo"
  $sudo_cmd systemctl enable devmon@devmon
  $sudo_cmd systemctl start devmon@devmon
}

USB_Stop_Auto() {
  ((EUID)) && sudo_cmd="sudo"
  $sudo_cmd systemctl stop devmon@devmon
  $sudo_cmd systemctl disable devmon@devmon
  $sudo_cmd udevil clean
}

GetDeviceTree(){  
  cat /proc/device-tree/model
}