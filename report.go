package main

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/axent-pl/resq/storage"
	"github.com/google/uuid"
)

type Report struct {
	Id      string `json:"Id"`
	Version int    `json:"Version"`
	Author  string `json:"Author"` // username (e.g. "sub" claim of the user who created the report)
	TeamID  string `form:"team_id" json:"TeamId"`
	// Device location
	DeviceLat string `form:"device_lat"`
	DeviceLng string `form:"device_lng"`
	// Incident
	IncidentTime     time.Time `form:"incident_time" json:"IncidentTime"`
	IncidentLocation string    `form:"incident_location" json:"Location"`
	// Patient
	PatientName  string `form:"patient_name"`
	PatientAge   int    `form:"patient_age"`
	PatientPESEL string `form:"patient_pesel"`
	PatientSex   string `form:"patient_sex"`
	// SAMPLE
	Symptoms    string `form:"symptoms"`
	Allergies   string `form:"allergies"`
	Medications string `form:"medications"`
	Past        string `form:"past"`
	LastIntake  string `form:"last"`
	Events      string `form:"events"`
	// Vitals
	HR      int `form:"vitals_hr"`
	SpO2    int `form:"vitals_spo2"`
	BPSys   int `form:"vitals_bp_sys"`
	BPDia   int `form:"vitals_bp_dia"`
	Glucose int `form:"vitals_glucose"`
	// Additional
	Notes         string `form:"notes"`
	Interventions string `form:"interventions"`
	Handoff       string `form:"handoff"`
}

func (r *Report) IsFromDate(dateStr string) bool {
	if dateStr == "" {
		return true
	}

	// Try parsing the date in standard formats (you can adjust as needed)
	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		// If parsing fails, ignore the condition
		return true
	}

	incident := r.IncidentTime
	y1, m1, d1 := incident.Date()
	y2, m2, d2 := parsed.Date()

	return y1 == y2 && m1 == m2 && d1 == d2
}

type ReportFilter func(Report) bool

type UpdateMode string

const (
	UpdateModeCreateNewVersion UpdateMode = "create_new_version"
	UpdateModeOverwriteVersion UpdateMode = "overwrite_version"
)

type ReportService struct {
	store *storage.Storage[string, Report]
}

func NewReportService(path string) (*ReportService, error) {
	store, err := storage.NewStorage[string, Report](path)
	if err != nil {
		return nil, err
	}
	r := &ReportService{
		store: store,
	}
	return r, nil
}

func (s *ReportService) reportKey(id string, version int) string {
	return fmt.Sprintf("%s:%d", id, version)
}

func (s *ReportService) listVersions(id string) ([]Report, error) {
	reports, err := s.store.FindBy(func(r Report) bool {
		return r.Id == id
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Version < reports[j].Version
	})
	return reports, nil
}

func (s *ReportService) latestVersionNumber(id string) (int, error) {
	versions, err := s.listVersions(id)
	if err != nil {
		return 0, err
	}
	if len(versions) == 0 {
		return 0, nil
	}
	return versions[len(versions)-1].Version, nil
}

func (s *ReportService) List() ([]Report, error) {
	return s.store.List()
}

func (s *ReportService) FindBy(filter func(Report) bool) ([]Report, error) {
	return s.store.FindBy(filter)
}

func (s *ReportService) Create(report Report) (Report, error) {
	if report.Id == "" {
		report.Id = uuid.NewString()
	}
	if report.Version <= 0 {
		report.Version = 1
	}
	if err := s.store.Create(s.reportKey(report.Id, report.Version), report); err != nil {
		return Report{}, err
	}
	return report, nil
}

func (s *ReportService) Read(id string) (Report, error) {
	versions, err := s.listVersions(id)
	if err != nil {
		return Report{}, err
	}
	if len(versions) == 0 {
		return Report{}, errors.New("not found")
	}
	return versions[len(versions)-1], nil
}

func (s *ReportService) ReadVersion(id string, version int) (Report, error) {
	return s.store.Read(s.reportKey(id, version))
}

func (s *ReportService) Update(report Report, mode UpdateMode) (Report, error) {
	if report.Id == "" {
		return Report{}, errors.New("report id is required")
	}
	if mode == "" {
		mode = UpdateModeCreateNewVersion
	}

	switch mode {
	case UpdateModeOverwriteVersion:
		if report.Version <= 0 {
			return Report{}, errors.New("version is required for overwrite mode")
		}
		if err := s.store.Update(s.reportKey(report.Id, report.Version), report); err != nil {
			return Report{}, err
		}
		return report, nil
	case UpdateModeCreateNewVersion:
		latestVersion, err := s.latestVersionNumber(report.Id)
		if err != nil {
			return Report{}, err
		}
		report.Version = latestVersion + 1
		if err := s.store.Create(s.reportKey(report.Id, report.Version), report); err != nil {
			return Report{}, err
		}
		return report, nil
	default:
		return Report{}, fmt.Errorf("unknown update mode %q", mode)
	}
}

var reportService *ReportService

func init() {
	var err error
	reportService, err = NewReportService("reports.json")
	if err != nil {
		log.Fatalf("report service init error: %v", err)
	}
}
