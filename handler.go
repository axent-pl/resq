package main

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/axent-pl/resq/utils"
)

const XHR_EVENT_REPORT_UPDATED utils.XhrEvent = "reportUpdated"

func getFilter(r *http.Request) ReportFilter {
	filters := make([]ReportFilter, 0)

	// Author
	if r.URL.Query().Get("author") != "" {
		author := r.URL.Query().Get("author")
		filters = append(filters, func(report Report) bool {
			return report.Author == author
		})
	}

	// IncidentTime day
	if r.URL.Query().Get("day") != "" {
		incidentTimeDay := r.URL.Query().Get("day")
		filters = append(filters, func(report Report) bool {
			return report.IsFromDate(incidentTimeDay)
		})
	}

	return func(report Report) bool {
		for _, f := range filters {
			if !f(report) {
				return false
			}
		}
		return true
	}
}

func exportReportsHandler(w http.ResponseWriter, r *http.Request) {
	reports, err := reportService.List()
	if err != nil {
		slog.Error("could not fetch reports for export", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("reports-%s.xlsx", time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	zw := zip.NewWriter(w)
	if err := writeReportsXLSX(zw, reports); err != nil {
		slog.Error("could not write reports xlsx", "error", err.Error())
		return
	}
	if err := zw.Close(); err != nil {
		slog.Error("could not close reports xlsx", "error", err.Error())
	}
}

func writeReportsXLSX(zw *zip.Writer, reports []Report) error {
	files := []struct {
		name string
		body string
	}{
		{
			name: "[Content_Types].xml",
			body: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
				`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
				`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
				`<Default Extension="xml" ContentType="application/xml"/>` +
				`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
				`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
				`</Types>`,
		},
		{
			name: "_rels/.rels",
			body: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
				`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
				`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
				`</Relationships>`,
		},
		{
			name: "xl/workbook.xml",
			body: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
				`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
				`<sheets><sheet name="Reports" sheetId="1" r:id="rId1"/></sheets>` +
				`</workbook>`,
		},
		{
			name: "xl/_rels/workbook.xml.rels",
			body: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
				`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
				`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
				`</Relationships>`,
		},
	}

	for _, file := range files {
		fw, err := zw.Create(file.name)
		if err != nil {
			return err
		}
		if _, err := fw.Write([]byte(file.body)); err != nil {
			return err
		}
	}

	fw, err := zw.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(fw, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`); err != nil {
		return err
	}

	headers := []string{
		"Id", "Version", "Author", "TeamID", "DeviceLat", "DeviceLng", "IncidentTime", "IncidentLocation",
		"PatientName", "PatientAge", "PatientPESEL", "PatientSex", "Symptoms", "Allergies", "Medications",
		"Past", "LastIntake", "Events", "HR", "SpO2", "BPSys", "BPDia", "Glucose", "Notes", "Interventions", "Handoff",
	}
	if err := writeXLSXRow(fw, headers, nil); err != nil {
		return err
	}

	for _, report := range reports {
		values := []string{
			report.Id,
			"",
			report.Author,
			report.TeamID,
			report.DeviceLat,
			report.DeviceLng,
			report.IncidentTime.Format("2006-01-02 15:04:05"),
			report.IncidentLocation,
			report.PatientName,
			"",
			report.PatientPESEL,
			report.PatientSex,
			report.Symptoms,
			report.Allergies,
			report.Medications,
			report.Past,
			report.LastIntake,
			report.Events,
			"",
			"",
			"",
			"",
			"",
			report.Notes,
			report.Interventions,
			report.Handoff,
		}
		numbers := map[int]int{
			1:  report.Version,
			9:  report.PatientAge,
			18: report.HR,
			19: report.SpO2,
			20: report.BPSys,
			21: report.BPDia,
			22: report.Glucose,
		}
		if err := writeXLSXRow(fw, values, numbers); err != nil {
			return err
		}
	}

	_, err = fmt.Fprint(fw, `</sheetData></worksheet>`)
	return err
}

func writeXLSXRow(w io.Writer, values []string, numbers map[int]int) error {
	if _, err := fmt.Fprint(w, `<row>`); err != nil {
		return err
	}
	for i, value := range values {
		if number, ok := numbers[i]; ok {
			if _, err := fmt.Fprintf(w, `<c><v>%d</v></c>`, number); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprint(w, `<c t="inlineStr"><is><t>`); err != nil {
			return err
		}
		if err := xmlEscapeText(w, value); err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, `</t></is></c>`); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, `</row>`)
	return err
}

func xmlEscapeText(w io.Writer, text string) error {
	return xml.EscapeText(w, []byte(text))
}

func listReportsHandler(w http.ResponseWriter, r *http.Request) {
	session := SessionFromRequest(r)
	data := ListPageData{
		CSRFToken:   "dummy_csrf",
		CurrentUser: session.Username,
	}
	data.Authors = []string{"demo.user", "team.lead"}
	data.Filters.Author = r.URL.Query().Get("author")
	data.Filters.Day = r.URL.Query().Get("day")

	reports, err := reportService.FindLatestBy(getFilter(r))
	if err != nil {
		slog.Error("could not fetch reports list", "error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data.Reports = reports

	data.Pagination.Page = 1
	data.Pagination.TotalPages = 1

	if err := ExecuteTemplate(w, "list", utils.IsXhr(r), data); err != nil {
		slog.Error(fmt.Sprintf("could not execute 'list' template: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func viewReportHandler(w http.ResponseWriter, r *http.Request) {
	session := SessionFromRequest(r)
	id := r.PathValue("id")
	version := r.URL.Query().Get("v")
	report, err := readReportWithOptionalVersion(id, version)
	if err != nil {
		slog.Error(fmt.Sprintf("could not find report: %v", err))
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := DetailPageData{
		CSRFToken:   "dummy_csrf",
		CurrentUser: session.Username,
		Report:      report,
	}

	if err := ExecuteTemplate(w, "detail", utils.IsXhr(r), data); err != nil {
		slog.Error(fmt.Sprintf("could not execute 'detail' template: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func historyHandler(w http.ResponseWriter, r *http.Request) {
	session := SessionFromRequest(r)
	id := r.PathValue("id")
	versions, err := reportService.ListVersions(id)
	if err != nil {
		slog.Error(fmt.Sprintf("could not fetch report versions: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(versions) == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	data := HistoryPageData{
		CSRFToken:   "dummy_csrf",
		CurrentUser: session.Username,
		ReportID:    id,
		Reports:     versions,
	}
	if err := ExecuteTemplate(w, "history", utils.IsXhr(r), data); err != nil {
		slog.Error(fmt.Sprintf("could not execute 'history' template: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func createReportHandler(w http.ResponseWriter, r *http.Request) {
	session := SessionFromRequest(r)
	if r.Method == http.MethodGet {
		data := FormPageData{
			CSRFToken:   "dummy_csrf",
			CurrentUser: session.Username,
			Action:      r.URL.Path,
			Mode:        "create",
			Report: Report{
				IncidentTime: time.Now(),
				Author:       session.Username,
			},
		}

		if err := ExecuteTemplate(w, "form", utils.IsXhr(r), data); err != nil {
			slog.Error(fmt.Sprintf("could not execute 'form' template: %v", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if r.Method == http.MethodPost {
		report := &Report{}
		if err := utils.Unmarshal(r, report); err != nil {
			slog.Error(fmt.Sprintf("could not unmarshal report data: %v", err))
			http.Error(w, "bad request", 400)
			return
		}
		report.Author = session.Username
		report.IncidentTime = time.Now()
		if _, err := reportService.Create(*report); err != nil {
			var ve ValidationErrors
			if errors.As(err, &ve) {
				data := FormPageData{
					CSRFToken:   "dummy_csrf",
					CurrentUser: session.Username,
					Action:      r.URL.Path,
					Mode:        "create",
					Report: Report{
						IncidentTime: time.Now(),
						Author:       session.Username,
					},
					ValidationErrors: ve.Errors,
				}
				slog.Info("did not validate", "validation_errors", ve)
				if errTpl := ExecuteTemplate(w, "form", utils.IsXhr(r), data); errTpl != nil {
					slog.Error(fmt.Sprintf("could not execute 'form' template: %v", errTpl))
					http.Error(w, errTpl.Error(), http.StatusInternalServerError)
				}
				return
			}
			slog.Error(fmt.Sprintf("could not create report: %v", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/reports", http.StatusSeeOther)
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func editReportHandler(w http.ResponseWriter, r *http.Request) {
	session := SessionFromRequest(r)
	id := r.PathValue("id")
	version := r.URL.Query().Get("v")
	report, err := readReportWithOptionalVersion(id, version)
	if err != nil {
		slog.Error(fmt.Sprintf("could not find report: %v", err))
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if r.Method == http.MethodGet {
		data := FormPageData{
			CSRFToken:   "dummy_csrf",
			CurrentUser: session.Username,
			Mode:        "edit",
			Action:      r.URL.Path,
			Report:      report,
		}
		if err := ExecuteTemplate(w, "form", utils.IsXhr(r), data); err != nil {
			slog.Error(fmt.Sprintf("could not execute 'form' template: %v", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if r.Method == http.MethodPost {
		updatedReport := &Report{}
		if err := utils.Unmarshal(r, updatedReport); err != nil {
			slog.Error(fmt.Sprintf("could not unmarshal report data: %v", err))
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		updatedReport.Id = id
		updatedReport.Version = report.Version
		updatedReport.Author = session.Username
		if _, err := reportService.Update(*updatedReport, UpdateModeOverwriteVersion); err != nil {
			slog.Error(fmt.Sprintf("could not update report: %v", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if utils.IsXhr(r) {
			utils.TriggerXhrEvent(w, XHR_EVENT_REPORT_UPDATED)
			viewReportHandler(w, r)
			return
		}
		http.Redirect(w, r, "/reports", http.StatusSeeOther)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func newVersionHandler(w http.ResponseWriter, r *http.Request) {
	session := SessionFromRequest(r)
	id := r.PathValue("id")
	baseVersion := r.URL.Query().Get("v")
	baseReport, err := readReportWithOptionalVersion(id, baseVersion)
	if err != nil {
		slog.Error(fmt.Sprintf("could not find base report: %v", err))
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if r.Method == http.MethodGet {
		baseReport.Version++
		data := FormPageData{
			CSRFToken:   "dummy_csrf",
			CurrentUser: session.Username,
			Action:      r.URL.Path,
			Mode:        "new-version",
			Report:      baseReport,
		}
		if err := ExecuteTemplate(w, "form", utils.IsXhr(r), data); err != nil {
			slog.Error(fmt.Sprintf("could not execute 'form' template: %v", err))
		}
		return
	}

	if r.Method == http.MethodPost {
		updatedReport := &Report{}
		if err := utils.Unmarshal(r, updatedReport); err != nil {
			slog.Error(fmt.Sprintf("could not unmarshal report data: %v", err))
			http.Error(w, "bad request", 400)
			return
		}
		updatedReport.Id = id
		updatedReport.Version = baseReport.Version + 1
		updatedReport.Author = session.Username
		if _, err := reportService.Update(*updatedReport, ""); err != nil {
			slog.Error(fmt.Sprintf("could not create new report version: %v", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if utils.IsXhr(r) {
			utils.TriggerXhrEvent(w, XHR_EVENT_REPORT_UPDATED)
			viewReportHandler(w, r)
			return
		}
		http.Redirect(w, r, "/reports", http.StatusSeeOther)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func readReportWithOptionalVersion(id string, version string) (Report, error) {
	if version == "" {
		return reportService.Read(id)
	}
	v, err := strconv.Atoi(version)
	if err != nil {
		return Report{}, fmt.Errorf("invalid version %q", version)
	}
	return reportService.ReadVersion(id, v)
}
