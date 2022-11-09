package common

import (
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen/message_bus"
)

var (
	SourceID = "local-storage"

	EventTypes = []message_bus.EventType{
		{
			SourceID: &SourceID,
			Name:     utils.Ptr("local-storage:disk:added"),
			PropertyTypeList: utils.Ptr(
				[]message_bus.PropertyType{
					{Name: utils.Ptr("path")},
				},
			),
		},

		// TODO - more events
	}
)
