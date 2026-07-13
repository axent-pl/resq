package service

import (
	"context"
	"testing"
	"time"

	"axent.pl/resq/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIncidentServiceCreateIncidentPersistsAggregate(t *testing.T) {
	db := testDB(t)
	service := NewIncidentService(db)

	incident := testIncident("INC-1", "open", time.Now())
	created, err := service.CreateIncident(context.Background(), incident)
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}

	if created.ID == 0 {
		t.Fatal("expected incident id to be set")
	}
	if created.Patient.ID == 0 || created.Patient.IncidentID != created.ID {
		t.Fatalf("expected patient to be persisted for incident, got %#v", created.Patient)
	}
	if len(created.Reports) != 1 {
		t.Fatalf("expected one report, got %d", len(created.Reports))
	}
	if created.Reports[0].IncidentID != created.ID {
		t.Fatalf("expected report incident id %d, got %d", created.ID, created.Reports[0].IncidentID)
	}
	if len(created.Reports[0].VitalMeasurements) != 1 {
		t.Fatalf("expected one vital measurement, got %d", len(created.Reports[0].VitalMeasurements))
	}
	if created.Reports[0].VitalMeasurements[0].ReportID != created.Reports[0].ID {
		t.Fatalf("expected vital report id %d, got %d", created.Reports[0].ID, created.Reports[0].VitalMeasurements[0].ReportID)
	}
}

func TestReportServiceCreateReportPersistsVitals(t *testing.T) {
	db := testDB(t)
	incidentService := NewIncidentService(db)
	reportService := NewReportService(db)

	incident, err := incidentService.CreateIncident(context.Background(), testIncident("INC-1", "open", time.Now()))
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}

	report := model.Report{
		IncidentID:  incident.ID,
		AuthorID:    1,
		Type:        "followup_assessment",
		PerformedAt: time.Now().Add(time.Minute),
		Notes:       "follow-up",
		VitalMeasurements: []model.VitalMeasurement{
			{MeasuredAt: time.Now().Add(time.Minute), SpO2: intPointer(97)},
		},
	}

	created, err := reportService.CreateReport(context.Background(), report)
	if err != nil {
		t.Fatalf("create report: %v", err)
	}

	if created.ID == 0 {
		t.Fatal("expected report id to be set")
	}
	if len(created.VitalMeasurements) != 1 {
		t.Fatalf("expected one vital measurement, got %d", len(created.VitalMeasurements))
	}

	reports, err := reportService.ListIncidentReports(context.Background(), incident.ID)
	if err != nil {
		t.Fatalf("list incident reports: %v", err)
	}
	if len(reports) != 2 {
		t.Fatalf("expected initial and follow-up reports, got %d", len(reports))
	}
}

func TestIncidentServiceUpdatePatient(t *testing.T) {
	db := testDB(t)
	service := NewIncidentService(db)

	incident, err := service.CreateIncident(context.Background(), testIncident("INC-1", "open", time.Now()))
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}

	updated, err := service.UpdatePatient(context.Background(), model.Patient{
		IncidentID: incident.ID,
		Name:       "Updated Patient",
		Age:        intPointer(45),
		PESEL:      "12345678901",
		Sex:        "female",
		Notes:      "updated notes",
	})
	if err != nil {
		t.Fatalf("update patient: %v", err)
	}

	if updated.Name != "Updated Patient" || updated.Age == nil || *updated.Age != 45 {
		t.Fatalf("unexpected updated patient: %#v", updated)
	}
}

func TestIncidentServiceListIncidentsFiltersAndPages(t *testing.T) {
	db := testDB(t)
	service := NewIncidentService(db)
	startedAt := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)

	_, err := service.CreateIncident(context.Background(), testIncident("INC-1", "open", startedAt))
	if err != nil {
		t.Fatalf("create first incident: %v", err)
	}
	_, err = service.CreateIncident(context.Background(), testIncident("INC-2", "closed", startedAt.Add(time.Hour)))
	if err != nil {
		t.Fatalf("create second incident: %v", err)
	}

	status := "open"
	incidents, page, err := service.ListIncidents(context.Background(), model.IncidentFilter{
		Status: &status,
	}, model.PagingQuery{
		Number: 1,
		Size:   10,
		SortBy: "started_at",
		Order:  "asc",
	})
	if err != nil {
		t.Fatalf("list incidents: %v", err)
	}

	if page.TotalItems != 1 || page.TotalPages != 1 {
		t.Fatalf("unexpected page result: %#v", page)
	}
	if len(incidents) != 1 || incidents[0].Number != "INC-1" {
		t.Fatalf("unexpected incidents: %#v", incidents)
	}
	if incidents[0].Patient.ID == 0 || len(incidents[0].Reports) != 1 {
		t.Fatalf("expected patient and reports to be preloaded: %#v", incidents[0])
	}
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&model.Event{}, &model.EventParticipant{}, &model.User{}, &model.Incident{}, &model.Patient{}, &model.Report{}, &model.VitalMeasurement{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	return db
}

func testIncident(number string, status string, startedAt time.Time) model.Incident {
	return model.Incident{
		EventID:     uintPointer(1),
		Number:      number,
		OccuredAt:   startedAt,
		Location:    "Sector A",
		Description: "test incident",
		Status:      status,
		Patient: model.Patient{
			Name: "Patient",
			Age:  intPointer(32),
			Sex:  "unknown",
		},
		Reports: []model.Report{
			{
				AuthorID:    1,
				Type:        "initial_assessment",
				PerformedAt: startedAt,
				Notes:       "initial",
				VitalMeasurements: []model.VitalMeasurement{
					{MeasuredAt: startedAt, SpO2: intPointer(98), GCS: intPointer(15), AVPU: "alert"},
				},
			},
		},
	}
}

func intPointer(value int) *int {
	return &value
}

func uintPointer(value uint) *uint { return &value }
