package common

import (
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen/message_bus"
)

const (
	UITypeNotificationStyle1 = "notification-style-1"
	UITypeNotificationStyle2 = "notification-style-2"
	UITypeNotificationStyle3 = "notification-style-3"
)

func EventAdapterWithUIProperties(event *message_bus.Event) *message_bus.Event {
	if event.SourceID != ServiceName {
		return event
	}

	propertyMap := make(map[string]string)

	for k, v := range event.Properties {
		propertyMap[k] = v
	}
	event.Properties = propertyMap

	return event
}
