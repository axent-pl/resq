package service

import (
	"context"
	"fmt"
	"strings"

	"axent.pl/resq/model"
	"gorm.io/gorm"
)

type EventService struct {
	db *gorm.DB
}

func NewEventService(db *gorm.DB) *EventService {
	return &EventService{db: db}
}

func (s *EventService) GetEvent(ctx context.Context, id uint) (model.Event, error) {
	var event model.Event
	err := s.db.WithContext(ctx).
		First(&event, id).
		Error
	if err != nil {
		return model.Event{}, err
	}

	return event, nil
}

func (s *EventService) CreateEvent(ctx context.Context, event model.Event) (model.Event, error) {
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.
			Session(&gorm.Session{FullSaveAssociations: false}).
			Create(&event).
			Error
	}); err != nil {
		return model.Event{}, err
	}

	return s.GetEvent(ctx, event.ID)
}

func (s *EventService) ListEvents(ctx context.Context, filter model.EventFilter, paging model.PagingQuery) ([]model.Event, model.PagingResult, error) {
	paging = normalizePaging(paging)

	query := applyEventFilter(s.db.WithContext(ctx).Model(&model.Event{}), filter)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, model.PagingResult{}, err
	}

	var events []model.Event
	listQuery := applyEventFilter(s.db.WithContext(ctx).Model(&model.Event{}), filter)
	err := listQuery.
		Order(eventOrderClause(paging)).
		Limit(paging.Size).
		Offset((paging.Number - 1) * paging.Size).
		Find(&events).
		Error
	if err != nil {
		return nil, model.PagingResult{}, err
	}

	return events, pageResult(paging, total), nil
}

func (s *EventService) EventOptions(ctx context.Context) map[string]string {
	var events []struct {
		ID   string
		Name string
	}
	if err := s.db.WithContext(ctx).
		Model(&model.Event{}).
		Select("id", "name").
		Find(&events).
		Error; err != nil {
		return map[string]string{}
	}

	options := make(map[string]string, len(events))
	for _, event := range events {
		options[event.ID] = event.Name
	}

	return options
}

func applyEventFilter(query *gorm.DB, filter model.EventFilter) *gorm.DB {
	if filter.StartedFrom != nil {
		query = query.Where("start_time >= ?", *filter.StartedFrom)
	}
	if filter.StartedTo != nil {
		query = query.Where("start_time <= ?", *filter.StartedTo)
	}
	if filter.Search != nil && strings.TrimSpace(*filter.Search) != "" {
		search := "%" + strings.TrimSpace(*filter.Search) + "%"
		query = query.Where(
			"name LIKE ? OR location LIKE ? OR description LIKE ?",
			search,
			search,
			search,
		)
	}

	return query
}

func eventOrderClause(paging model.PagingQuery) string {
	sortBy := map[string]string{
		"id":         "id",
		"name":       "name",
		"start_time": "start_time",
		"end_time":   "end_time",
		"location":   "location",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}[paging.SortBy]
	if sortBy == "" {
		sortBy = "start_time"
	}

	return fmt.Sprintf("%s %s", sortBy, strings.ToUpper(paging.Order))
}
