package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"axent.pl/resq/model"
	"gorm.io/gorm"
)

type IncidentService struct {
	db *gorm.DB
}

func NewIncidentService(db *gorm.DB) *IncidentService {
	return &IncidentService{db: db}
}

func (s *IncidentService) CreateIncident(ctx context.Context, incident model.Incident) (model.Incident, error) {
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.
			Session(&gorm.Session{FullSaveAssociations: true}).
			Create(&incident).
			Error
	}); err != nil {
		return model.Incident{}, err
	}

	return s.GetIncident(ctx, incident.ID)
}

func (s *IncidentService) GetIncident(ctx context.Context, id uint) (model.Incident, error) {
	var incident model.Incident
	err := s.withIncidentPreloads(s.db.WithContext(ctx)).
		First(&incident, id).
		Error
	if err != nil {
		return model.Incident{}, err
	}

	return incident, nil
}

func (s *IncidentService) ListIncidents(ctx context.Context, filter model.IncidentFilter, paging model.PagingQuery) ([]model.Incident, model.PagingResult, error) {
	paging = normalizePaging(paging)

	query := applyIncidentFilter(s.db.WithContext(ctx).Model(&model.Incident{}), filter)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, model.PagingResult{}, err
	}

	var incidents []model.Incident
	listQuery := applyIncidentFilter(s.db.WithContext(ctx).Model(&model.Incident{}), filter)
	err := s.withIncidentPreloads(listQuery).
		Order(incidentOrderClause(paging)).
		Limit(paging.Size).
		Offset((paging.Number - 1) * paging.Size).
		Find(&incidents).
		Error
	if err != nil {
		return nil, model.PagingResult{}, err
	}

	return incidents, pageResult(paging, total), nil
}

func (s *IncidentService) UpdatePatient(ctx context.Context, patient model.Patient) (model.Patient, error) {
	if patient.ID == 0 && patient.IncidentID == 0 {
		return model.Patient{}, errors.New("patient id or incident id is required")
	}

	var existing model.Patient
	query := s.db.WithContext(ctx)
	if patient.ID != 0 {
		query = query.First(&existing, patient.ID)
	} else {
		query = query.Where("incident_id = ?", patient.IncidentID).First(&existing)
	}
	if err := query.Error; err != nil {
		return model.Patient{}, err
	}

	existing.Name = patient.Name
	existing.Age = patient.Age
	existing.PESEL = patient.PESEL
	existing.Sex = patient.Sex
	existing.Notes = patient.Notes

	if err := s.db.WithContext(ctx).Save(&existing).Error; err != nil {
		return model.Patient{}, err
	}

	return existing, nil
}

func (s *IncidentService) withIncidentPreloads(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Patient").
		Preload("Reports", func(db *gorm.DB) *gorm.DB {
			return db.Order("performed_at ASC, id ASC")
		}).
		Preload("Reports.VitalMeasurements", func(db *gorm.DB) *gorm.DB {
			return db.Order("measured_at ASC, id ASC")
		})
}

func applyIncidentFilter(query *gorm.DB, filter model.IncidentFilter) *gorm.DB {
	if filter.EventID != nil {
		query = query.Where("event_id = ?", *filter.EventID)
	}
	if filter.Status != nil && strings.TrimSpace(*filter.Status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(*filter.Status))
	}
	if filter.StartedFrom != nil {
		query = query.Where("started_at >= ?", *filter.StartedFrom)
	}
	if filter.StartedTo != nil {
		query = query.Where("started_at <= ?", *filter.StartedTo)
	}
	if filter.Search != nil && strings.TrimSpace(*filter.Search) != "" {
		search := "%" + strings.TrimSpace(*filter.Search) + "%"
		query = query.Where(
			"number LIKE ? OR location LIKE ? OR description LIKE ?",
			search,
			search,
			search,
		)
	}

	return query
}

func incidentOrderClause(paging model.PagingQuery) string {
	sortBy := map[string]string{
		"id":         "id",
		"number":     "number",
		"started_at": "started_at",
		"status":     "status",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}[paging.SortBy]
	if sortBy == "" {
		sortBy = "started_at"
	}

	return fmt.Sprintf("%s %s", sortBy, strings.ToUpper(paging.Order))
}
