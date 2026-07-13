package service

import (
	"context"
	"errors"

	"axent.pl/resq/model"
	"gorm.io/gorm"
)

type ReportService struct {
	db *gorm.DB
}

func NewReportService(db *gorm.DB) *ReportService {
	return &ReportService{db: db}
}

func (s *ReportService) CreateReport(ctx context.Context, report model.Report) (model.Report, error) {
	if report.IncidentID == 0 {
		return model.Report{}, errors.New("incident id is required")
	}

	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.
			Session(&gorm.Session{FullSaveAssociations: true}).
			Omit("Incident", "Author").
			Create(&report).
			Error
	}); err != nil {
		return model.Report{}, err
	}

	return s.GetReport(ctx, report.ID)
}

func (s *ReportService) GetReport(ctx context.Context, id uint) (model.Report, error) {
	var report model.Report
	err := s.db.WithContext(ctx).
		Preload("VitalMeasurements").
		First(&report, id).
		Error
	if err != nil {
		return model.Report{}, err
	}

	return report, nil
}

func (s *ReportService) ListIncidentReports(ctx context.Context, incidentID uint) ([]model.Report, error) {
	var reports []model.Report
	err := s.db.WithContext(ctx).
		Preload("VitalMeasurements").
		Where("incident_id = ?", incidentID).
		Order("performed_at ASC, id ASC").
		Find(&reports).
		Error
	if err != nil {
		return nil, err
	}

	return reports, nil
}
