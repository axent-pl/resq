package dto

import (
	"time"

	"axent.pl/resq/model"
)

type IncidentsListQueryDTO struct {
	EventID     *uint      `form:"event_id"`
	Status      *string    `form:"status"`
	StartedFrom *time.Time `form:"started_from"`
	StartedTo   *time.Time `form:"started_to"`
	Search      *string    `form:"search"`
	Page        int        `form:"page"`
	Size        int        `form:"size"`
	SortBy      string     `form:"sort_by"`
	Order       string     `form:"order"`
}

func (d *IncidentsListQueryDTO) MapToModel() (model.IncidentFilter, model.PagingQuery, error) {
	filter := model.IncidentFilter{
		EventID:     d.EventID,
		Status:      d.Status,
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

type IncidentCreateRequestDTO struct {
	EventID *uint `json:"event_id" form:"event_id"`

	Number      string    `json:"number" form:"number" validate:"required"`
	OccuredAt   time.Time `json:"occured_at" form:"occured_at" validate:"required"`
	Location    string    `json:"location" form:"location" validate:"required"`
	Description string    `json:"description" form:"description"`
	Status      string    `json:"status" form:"status" validate:"required,oneof=open in_progress handed_over closed"`

	Patient PatientCreateRequestDTO `json:"patient" form:"patient"`
	Report  ReportCreateRequestDTO  `json:"report" form:"report"`
}

func (d *IncidentCreateRequestDTO) MapToModel() model.Incident {
	return model.Incident{
		EventID:     d.EventID,
		Number:      d.Number,
		OccuredAt:   d.OccuredAt,
		Location:    d.Location,
		Description: d.Description,
		Status:      d.Status,
		Patient:     d.Patient.MapToModel(),
		Reports:     []model.Report{d.Report.MapToModel()},
	}
}

type PatientCreateRequestDTO struct {
	Name  string `json:"name" form:"name" validate:"required"`
	Age   *int   `json:"age,omitempty" form:"age" validate:"min=0,max=130"`
	PESEL string `json:"pesel" form:"pesel"`
	Sex   string `json:"sex" form:"sex" validate:"required,oneof=male female unknown other"`
	Notes string `json:"notes" form:"notes"`
}

func (d *PatientCreateRequestDTO) MapToModel() model.Patient {
	return model.Patient{
		Name:  d.Name,
		Age:   d.Age,
		PESEL: d.PESEL,
		Sex:   d.Sex,
		Notes: d.Notes,
	}
}

type ReportCreateRequestDTO struct {
	AuthorID          uint                             `json:"author_id" form:"author_id"`
	Type              string                           `json:"type" form:"type"`
	PerformedAt       time.Time                        `json:"performed_at" form:"performed_at"`
	DeviceLat         *float64                         `json:"device_lat,omitempty" form:"device_lat"`
	DeviceLng         *float64                         `json:"device_lng,omitempty" form:"device_lng"`
	Symptoms          string                           `json:"symptoms" form:"symptoms"`
	Allergies         string                           `json:"allergies" form:"allergies"`
	Medications       string                           `json:"medications" form:"medications"`
	PastMedical       string                           `json:"past_medical" form:"past_medical"`
	LastIntake        string                           `json:"last_intake" form:"last_intake"`
	Events            string                           `json:"events" form:"events"`
	Notes             string                           `json:"notes" form:"notes"`
	Interventions     string                           `json:"interventions" form:"interventions"`
	Handoff           string                           `json:"handoff" form:"handoff"`
	VitalMeasurements VitalMeasurementCreateRequestDTO `json:"vital_measurements" form:"vital_measurements"`
}

func (d *ReportCreateRequestDTO) MapToModel() model.Report {
	now := time.Now()
	return model.Report{
		AuthorID:          d.AuthorID,
		Type:              d.Type,
		PerformedAt:       now,
		DeviceLat:         d.DeviceLat,
		DeviceLng:         d.DeviceLng,
		Symptoms:          d.Symptoms,
		Allergies:         d.Allergies,
		Medications:       d.Medications,
		PastMedical:       d.PastMedical,
		LastIntake:        d.LastIntake,
		Events:            d.Events,
		Notes:             d.Notes,
		Interventions:     d.Interventions,
		Handoff:           d.Handoff,
		VitalMeasurements: []model.VitalMeasurement{d.VitalMeasurements.MapToModel()},
	}
}

type VitalMeasurementCreateRequestDTO struct {
	MeasuredAt      time.Time `json:"measured_at" form:"measured_at"`
	HR              *int      `json:"hr,omitempty" form:"hr" validate:"min=0,max=300"`
	SpO2            *int      `json:"spo2,omitempty" form:"spo2" validate:"min=0,max=100"`
	BPSys           *int      `json:"bp_sys,omitempty" form:"bp_sys" validate:"min=0,max=300"`
	BPDia           *int      `json:"bp_dia,omitempty" form:"bp_dia" validate:"min=0,max=200"`
	Glucose         *int      `json:"glucose,omitempty" form:"glucose" validate:"min=0,max=1000"`
	RespiratoryRate *int      `json:"respiratory_rate,omitempty" form:"respiratory_rate" validate:"min=0,max=100"`
	Temperature     *float64  `json:"temperature,omitempty" form:"temperature" validate:"min=25,max=45"`
	AVPU            string    `json:"avpu" form:"avpu" validate:"oneof=alert voice pain unresponsive"`
	GCS             *int      `json:"gcs,omitempty" form:"gcs" validate:"min=3,max=15"`
	Notes           string    `json:"notes" form:"notes"`
}

func (d *VitalMeasurementCreateRequestDTO) MapToModel() model.VitalMeasurement {
	now := time.Now()
	return model.VitalMeasurement{
		MeasuredAt:      now,
		HR:              d.HR,
		SpO2:            d.SpO2,
		BPSys:           d.BPSys,
		BPDia:           d.BPDia,
		Glucose:         d.Glucose,
		RespiratoryRate: d.RespiratoryRate,
		Temperature:     d.Temperature,
		AVPU:            d.AVPU,
		GCS:             d.GCS,
		Notes:           d.Notes,
	}
}
