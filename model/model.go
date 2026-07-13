package model

import (
	"time"

	"axent.pl/resq/i18n"
	"gorm.io/gorm"
)

type Event struct {
	gorm.Model

	Name        string
	StartTime   time.Time
	EndTime     time.Time
	Location    string
	Description string
}

type EventParticipant struct {
	gorm.Model

	EventID uint
	Event   Event
	UserID  uint
	User    User

	Status string
	Role   string
	Notes  string
}

type User struct {
	gorm.Model

	Username    string `gorm:"uniqueIndex;not null"`
	DisplayName string
	Email       string
	Phone       string
	Role        string
}

type Incident struct {
	gorm.Model

	EventID *uint
	Event   Event
	Patient Patient
	Reports []Report

	Number      string    `json:"number"` // np. numer wewnętrzny incydentu
	OccuredAt   time.Time `json:"started_at"`
	Location    string    `json:"location"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // open, in_progress, handed_over, closed
}

func (m Incident) StatusOptions() map[string]string {
	return map[string]string{
		"open":        "Open",
		"in_progress": "In progress",
		"handed_over": "Handed over",
		"closed":      "Closed",
	}
}

type Patient struct {
	gorm.Model

	IncidentID uint `gorm:"uniqueIndex"`

	Name  string `json:"name"`
	Age   *int   `json:"age"`
	PESEL string `json:"pesel"`
	Sex   string `json:"sex"` // male, female, unknown, other
	Notes string `json:"notes"`
}

func (m Patient) SexOptions() map[string]string {
	return map[string]string{
		i18n.Mark("unknown"): i18n.Mark("Unknown"),
		i18n.Mark("male"):    i18n.Mark("Male"),
		i18n.Mark("female"):  i18n.Mark("Female"),
		i18n.Mark("other"):   i18n.Mark("Other"),
	}
}

type Report struct {
	gorm.Model

	IncidentID        uint
	AuthorID          uint
	Author            User
	VitalMeasurements []VitalMeasurement `gorm:"foreignKey:ReportID"`

	Type          string    `json:"type"` // initial_assessment, followup_assessment, intervention, handoff, note
	PerformedAt   time.Time `json:"performed_at"`
	DeviceLat     *float64  `json:"device_lat"`
	DeviceLng     *float64  `json:"device_lng"`
	Symptoms      string    `json:"symptoms"`
	Allergies     string    `json:"allergies"`
	Medications   string    `json:"medications"`
	PastMedical   string    `json:"past_medical"`
	LastIntake    string    `json:"last_intake"`
	Events        string    `json:"events"`
	Notes         string    `json:"notes"`
	Interventions string    `json:"interventions"`
	Handoff       string    `json:"handoff"`
}

func (m Report) TypeOptions() map[string]string {
	return map[string]string{
		i18n.Mark("initial_assessment"):  i18n.Mark("Initial assessment"),
		i18n.Mark("followup_assessment"): i18n.Mark("Follow-up assessment"),
		i18n.Mark("intervention"):        i18n.Mark("Intervention"),
		i18n.Mark("handoff"):             i18n.Mark("Handoff"),
		i18n.Mark("note"):                i18n.Mark("Note"),
	}
}

type VitalMeasurement struct {
	gorm.Model

	ReportID uint

	MeasuredAt      time.Time `json:"measured_at"`
	HR              *int      `json:"hr"`
	SpO2            *int      `json:"spo2"`
	BPSys           *int      `json:"bp_sys"`
	BPDia           *int      `json:"bp_dia"`
	Glucose         *int      `json:"glucose"`
	RespiratoryRate *int      `json:"respiratory_rate"`
	Temperature     *float64  `json:"temperature"`
	AVPU            string    `json:"avpu"` // alert, voice, pain, unresponsive
	GCS             *int      `json:"gcs"`
	Notes           string    `json:"notes"`
}

func (m VitalMeasurement) AVPUOptions() map[string]string {
	return map[string]string{
		i18n.Mark("alert"):        i18n.Mark("Alert"),
		i18n.Mark("voice"):        i18n.Mark("Voice"),
		i18n.Mark("pain"):         i18n.Mark("Pain"),
		i18n.Mark("unresponsive"): i18n.Mark("Unresponsive"),
	}
}
