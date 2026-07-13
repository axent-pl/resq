package dto

import (
	"errors"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestUnmarshalDTOFromFormDecodesNestedCreateIncidentRequest(t *testing.T) {
	form := url.Values{
		"event_id":                              {"7"},
		"number":                                {"INC-42"},
		"started_at":                            {"2026-06-20T10:15:00Z"},
		"location":                              {"Sector A"},
		"lat":                                   {"53.4285"},
		"lng":                                   {"14.5528"},
		"description":                           {"patient fainted"},
		"status":                                {"open"},
		"patient.name":                          {"Jan Kowalski"},
		"patient.age":                           {"34"},
		"patient.pesel":                         {"12345678901"},
		"patient.sex":                           {"male"},
		"patient.notes":                         {"no notes"},
		"report.author_id":                      {"3"},
		"report.type":                           {"initial_assessment"},
		"report.performed_at":                   {"2026-06-20T10:16:00Z"},
		"report.symptoms":                       {"dizziness"},
		"report.vital_measurements.measured_at": {"2026-06-20T10:17:00Z"},
		"report.vital_measurements.hr":          {"82"},
		"report.vital_measurements.spo2":        {"98"},
		"report.vital_measurements.temperature": {"36.7"},
		"report.vital_measurements.avpu":        {"alert"},
		"report.vital_measurements.gcs":         {"15"},
		"report.vital_measurements.respiratory_rate": {"16"},
		"report.vital_measurements.notes":            {"stable"},
	}

	req := httptest.NewRequest("POST", "/incidents", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	d, err := UnmarshalDTO[IncidentCreateRequestDTO](req)
	if err != nil {
		t.Fatalf("unmarshal form: %v", err)
	}

	if *d.EventID != 7 || d.Number != "INC-42" || d.Patient.Name != "Jan Kowalski" {
		t.Fatalf("unexpected dto: %#v", d)
	}
	if d.Patient.Age == nil || *d.Patient.Age != 34 {
		t.Fatalf("expected patient age pointer to be decoded, got %#v", d.Patient.Age)
	}
	if d.Report.VitalMeasurements.SpO2 == nil || *d.Report.VitalMeasurements.SpO2 != 98 {
		t.Fatalf("expected first spo2 to be decoded, got %#v", d.Report.VitalMeasurements.SpO2)
	}
	if d.Report.VitalMeasurements.RespiratoryRate == nil || *d.Report.VitalMeasurements.RespiratoryRate != 16 {
		t.Fatalf("expected second respiratory rate to be decoded, got %#v", d.Report.VitalMeasurements.RespiratoryRate)
	}
}

func TestUnmarshalDTOReturnsValidationErrorForInvalidFormValues(t *testing.T) {
	form := url.Values{
		"event_id":                              {"abc"},
		"number":                                {"INC-42"},
		"occured_at":                            {"not-a-date"},
		"location":                              {"Sector A"},
		"status":                                {"open"},
		"patient.name":                          {"Jan Kowalski"},
		"patient.age":                           {"old"},
		"patient.sex":                           {"male"},
		"report.vital_measurements.temperature": {"warm"},
	}

	req := httptest.NewRequest("POST", "/incidents", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	d, err := UnmarshalDTO[IncidentCreateRequestDTO](req)
	if err == nil {
		t.Fatal("expected validation error")
	}

	var validationErr ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	expectedFields := []string{
		"event_id",
		"occured_at",
		"patient.age",
		"report.vital_measurements.temperature",
	}
	for _, field := range expectedFields {
		if len(validationErr.Errors[field]) == 0 {
			t.Fatalf("expected unmarshal error for %q, got %#v", field, validationErr.Errors)
		}
	}
	if d.Number != "INC-42" || d.Status != "open" || d.Patient.Name != "Jan Kowalski" {
		t.Fatalf("expected valid fields to remain decoded, got %#v", d)
	}
}
