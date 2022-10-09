/*@Author: LinkLeong link@icewhale.org
 *@Date: 2021-12-07 17:14:41
 *@LastEditors: LinkLeong
 *@LastEditTime: 2022-08-17 18:46:43
 *@FilePath: /CasaOS/service/model/o_disk.go
 *@Description:
 *@Website: https://www.casaos.io
 *Copyright (c) 2022 by icewhale, All Rights Reserved.
 */
package model

type Volume struct {
	ID         uint   `gorm:"column:id;primary_key" json:"id"`
	UUID       string `json:"uuid"`
	State      int    `json:"state"`
	MountPoint string `json:"mount_point"`
	CreatedAt  int64  `json:"created_at"`
}

func (p *Volume) TableName() string {
	return "o_disk" // legacy table name - :(
}
