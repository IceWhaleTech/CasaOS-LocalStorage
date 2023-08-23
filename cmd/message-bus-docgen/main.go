package main

import (
	"github.com/IceWhaleTech/CasaOS-Common/external"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/codegen/message_bus"
	"github.com/IceWhaleTech/CasaOS-LocalStorage/common"
	"github.com/samber/lo"
)

func main() {
	eventTypes := lo.Flatten(
		lo.MapToSlice(
			common.EventTypes,
			func(key string, eventTypeMap map[string]message_bus.EventType) []external.EventType {
				return lo.MapToSlice(
					eventTypeMap,
					func(key string, eventType message_bus.EventType) external.EventType {
						return external.EventType{
							Name:     eventType.Name,
							SourceID: eventType.SourceID,
							PropertyTypeList: lo.Map(
								eventType.PropertyTypeList, func(item message_bus.PropertyType, index int) external.PropertyType {
									return external.PropertyType{
										Name:        item.Name,
										Description: item.Description,
										Example:     item.Example,
									}
								},
							),
						}
					},
				)
			},
		),
	)

	external.PrintEventTypesAsMarkdown(common.ServiceName, common.Version, eventTypes)
}
