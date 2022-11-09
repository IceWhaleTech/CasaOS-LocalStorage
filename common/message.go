package common

import (
	"fmt"

	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen/message_bus"
	"github.com/pilebones/go-udev/netlink"
)

var (
	// devtype -> action -> event
	EventTypes map[string]map[string]message_bus.EventType

	EventPropertyNamePath string
)

func init() {
	EventPropertyNamePath = fmt.Sprintf("%s:%s", ServiceName, "path")

	for _, devtype := range []string{"disk", "partition"} {
		for _, action := range []string{"add", "remove"} {
			if EventTypes == nil {
				EventTypes = make(map[string]map[string]message_bus.EventType)
			}

			if EventTypes[devtype] == nil {
				EventTypes[devtype] = make(map[string]message_bus.EventType)
			}

			EventTypes[devtype][action] = message_bus.EventType{
				SourceID: utils.Ptr(ServiceName),
				Name:     utils.Ptr(fmt.Sprintf("%s:%s:%s", ServiceName, devtype, action)),
				PropertyTypeList: &[]message_bus.PropertyType{
					{Name: utils.Ptr(EventPropertyNamePath)},
				},
			}
		}
	}
}

func EventAdapter(e netlink.UEvent) *message_bus.Event {
	eventType, ok := EventTypes[e.Env["DEVTYPE"]][string(e.Action)]
	if !ok {
		return nil
	}

	return &message_bus.Event{
		SourceID: eventType.SourceID,
		Name:     eventType.Name,
		Properties: &[]message_bus.Property{
			{
				Name:  utils.Ptr(EventPropertyNamePath),
				Value: utils.Ptr(e.Env["DEVNAME"]),
			},
		},
	}
}
