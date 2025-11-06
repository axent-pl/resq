// main.go (skrótowo)
package main

import (
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/axent-pl/resq/storage"
	"github.com/axent-pl/resq/utils"
	"github.com/google/uuid"
)

type Report struct {
	Id     string `json:"Id"`
	TeamID string `form:"team_id" json:"TeamId"`
	// Incident
	IncidentTime time.Time `form:"incident_time" json:"IncidentTime"`
	Location     string    `form:"location" json:"Location"`
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

func (s *ReportService) List() ([]Report, error) {
	return s.store.List()
}

func (s *ReportService) Create(report Report) (Report, error) {
	report.Id = uuid.NewString()
	if err := s.store.Create(report.Id, report); err != nil {
		return Report{}, err
	}
	return report, nil
}

type ViewData struct {
	PostAction string
	Values     Report
}

var (
	templateService *template.Template
	reportService   *ReportService
)

func init() {
	templateService = template.Must(
		template.New("assessment.html.tmpl").
			Funcs(template.FuncMap{
				"formatDate": func(t time.Time) string {
					if t.IsZero() {
						return ""
					}
					return t.Format("2006-01-02T15:04")
				},
			}).
			ParseFiles("templates/assessment.html.tmpl"),
	)
	reportServiceInit, err := NewReportService("reports.json")
	if err != nil {
		panic(err)
	}
	reportService = reportServiceInit
}

func formHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := ViewData{
			PostAction: "/",
			Values: Report{
				IncidentTime: time.Now(),
				HR:           80,
				SpO2:         98,
				BPSys:        120,
				BPDia:        80,
				Glucose:      100,
			},
		}
		err := templateService.ExecuteTemplate(w, "assessment.html.tmpl", data)
		if err != nil {
			log.Fatalf("could not execute template: %v", err)
		}
	case http.MethodPost:
		report := &Report{}
		if err := utils.Unmarshal(r, report); err != nil {
			slog.Error(fmt.Sprintf("could not unmarshal report data: %v", err))
		}
		reportService.Create(*report)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", formHandler)
	log.Fatal(http.ListenAndServe(":1234", mux))
}
