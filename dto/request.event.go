package dto

import (
	"time"

	"axent.pl/resq/model"
)

type EventsListQueryDTO struct {
	Search      *string    `form:"search"`
	StartedFrom *time.Time `form:"started_from"`
	StartedTo   *time.Time `form:"started_to"`
	Page        int        `form:"page"`
	Size        int        `form:"size"`
	SortBy      string     `form:"sort_by"`
	Order       string     `form:"order"`
}

func (d *EventsListQueryDTO) MapToModel() (model.EventFilter, model.PagingQuery, error) {
	filter := model.EventFilter{
		StartedFrom: d.StartedFrom,
		StartedTo:   d.StartedTo,
		Search:      d.Search,
	}
	paging := model.PagingQuery{
		Number: d.Page,
		Size:   d.Size,
		SortBy: d.SortBy,
		Order:  d.Order,
	}
	return filter, paging, nil
}

type EventCreateRequestDTO struct {
	Name        string     `json:"name" form:"name" validate:"required"`
	StartTime   *time.Time `json:"start_time" form:"start_time" validate:"required"`
	EndTime     *time.Time `json:"end_time" form:"end_time" validate:"required"`
	Location    string     `json:"location" form:"location" validate:"required"`
	Description string     `json:"description" form:"description"`
}

func (d *EventCreateRequestDTO) MapToModel() model.Event {
	return model.Event{
		Name:        d.Name,
		StartTime:   *d.StartTime,
		EndTime:     *d.EndTime,
		Location:    d.Location,
		Description: d.Description,
	}
}
