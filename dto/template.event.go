package dto

import "axent.pl/resq/model"

type EventListTemplateDTO struct {
	*BaseTemplateDTO
	Events        []model.Event
	Filter        model.EventFilter
	Paging        model.PagingResult
	PrevPageQuery string
	NextPageQuery string
}

type EventFormTemplateDTO struct {
	*BaseTemplateDTO
	Form   EventCreateRequestDTO
	Errors map[string][]string
}

type EventReadTemplateDTO struct {
	*BaseTemplateDTO
	Event model.Event
}
