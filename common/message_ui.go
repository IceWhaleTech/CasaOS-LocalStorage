package common

import (
	"fmt"

	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen/message_bus"
)

const (
	UIPropertyNameType     = "casaos-ui:type"
	UIPropertyNameTitle    = "casaos-ui:title"
	UIPropertyNameIcon1    = "casaos-ui:icon-1"
	UIPropertyNameIcon2    = "casaos-ui:icon-2"
	UIPropertyNameIcon3    = "casaos-ui:icon-3"
	UIPropertyNameMessage1 = "casaos-ui:message-1"
	UIPropertyNameMessage2 = "casaos-ui:message-2"
	UIPropertyNameMessage3 = "casaos-ui:message-3"

	UITypeNotificationStyle1 = "notification-style-1"
	UITypeNotificationStyle2 = "notification-style-2"
	UITypeNotificationStyle3 = "notification-style-3"
)

// property types for rendering event as notification in CasaOS-UI
var UIPropertyTypes = []message_bus.PropertyType{
	{Name: UIPropertyNameType},     // e.g. notification-style-2
	{Name: UIPropertyNameTitle},    // e.g. "New disk found"
	{Name: UIPropertyNameIcon1},    // e.g. "casaos-icon-disk"
	{Name: UIPropertyNameMessage1}, // e.g. "A new disk, SanDisk Cruzer, is added."
}

// include property types for CasaOS UI in eventType
func AddUIPropertyTypes(eventType message_bus.EventType) message_bus.EventType {
	eventType.PropertyTypeList = append(eventType.PropertyTypeList, UIPropertyTypes...)
	return eventType
}

func EventAdapterWithUIProperties(event *message_bus.Event) *message_bus.Event {
	if event.SourceID != ServiceName {
		return event
	}

	switch event.Name {
	case fmt.Sprintf("%s:%s:%s", ServiceName, "disk", "added"):
		propertyMap := PropertiesToMap(event.Properties)

		vendor := propertyMap[fmt.Sprintf("%s:%s", ServiceName, "vendor")]
		model := propertyMap[fmt.Sprintf("%s:%s", ServiceName, "model")]

		propertyMap[UIPropertyNameType] = UITypeNotificationStyle2
		propertyMap[UIPropertyNameTitle] = "New disk found"
		propertyMap[UIPropertyNameIcon1] = "casaos-icon-disk"
		propertyMap[UIPropertyNameMessage1] = fmt.Sprintf("A new disk, %s %s, is added.", vendor, model)

		event.Properties = MapToProperties(propertyMap)
	}

	return event
}
