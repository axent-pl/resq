package dto

import (
	"errors"
	"testing"
	"time"
)

func TestValidateDTOReturnsNestedValidationErrors(t *testing.T) {
	req := IncidentCreateRequestDTO{
		Status: "invalid",
		Patient: PatientCreateRequestDTO{
			Sex: "invalid",
		},
		Report: ReportCreateRequestDTO{
			Type: "invalid",
			VitalMeasurements: VitalMeasurementCreateRequestDTO{
				SpO2: intPtr(101),
				GCS:  intPtr(2),
			},
		},
	}

	err := ValidateDTO(req)
	if err == nil {
		t.Fatal("expected validation error")
	}

	var validationErr ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	expectedFields := []string{
		"number",
		"started_at",
		"location",
		"status",
		"patient.name",
		"patient.sex",
		"report.author_id",
		"report.type",
		"report.performed_at",
		"report.vital_measurements.measured_at",
		"report.vital_measurements.spo2",
		"report.vital_measurements.gcs",
	}

	for _, field := range expectedFields {
		if len(validationErr.Errors[field]) == 0 {
			t.Fatalf("expected validation error for %q, got %#v", field, validationErr.Errors)
		}
	}
}

func TestValidateDTOAllowsValidCreateIncidentRequest(t *testing.T) {
	req := IncidentCreateRequestDTO{
		EventID:   uintPointer(1),
		Number:    "INC-1",
		OccuredAt: time.Now(),
		Location:  "Sector A",
		Status:    "open",
		Patient: PatientCreateRequestDTO{
			Name: "Jan Kowalski",
			Age:  intPtr(32),
			Sex:  "male",
		},
		Report: ReportCreateRequestDTO{
			AuthorID:    1,
			Type:        "initial_assessment",
			PerformedAt: time.Now(),
			VitalMeasurements: VitalMeasurementCreateRequestDTO{
				MeasuredAt: time.Now(),
				SpO2:       intPtr(98),
				GCS:        intPtr(15),
				AVPU:       "alert",
			},
		},
	}

	if err := ValidateDTO(req); err != nil {
		t.Fatalf("expected request to be valid, got %v", err)
	}
}

func TestValidateDTOUsesCustomFunctionRule(t *testing.T) {
	type request struct {
		Code string `validate:"fn=uppercase"`
	}

	funcs := map[string]ValidationFunc{
		"uppercase": func(value any) bool {
			code, ok := value.(string)
			return ok && code == "ABC"
		},
	}

	if err := ValidateDTO(request{Code: "ABC"}, funcs); err != nil {
		t.Fatalf("expected request to be valid, got %v", err)
	}

	err := ValidateDTO(request{Code: "abc"}, funcs)
	if err == nil {
		t.Fatal("expected validation error")
	}

	var validationErr ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if len(validationErr.Errors["Code"]) == 0 {
		t.Fatalf("expected validation error for Code, got %#v", validationErr.Errors)
	}
}

func intPtr(value int) *int { return &value }

func uintPointer(value uint) *uint { return &value }
