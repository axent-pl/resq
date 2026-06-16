package main

import (
	"errors"
	"fmt"
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

	report := &Report{}
	if err := utils.Unmarshal(r, report); err != nil {
		slog.Error(fmt.Sprintf("could not unmarshal report data: %v", err))
		http.Error(w, "bad request", 400)
		return
	}
	report.Author = session.Username
	if _, err := reportService.Create(*report); err != nil {
		var ve ValidationErrors
		if errors.As(err, &ve) {
			data := FormPageData{
				CSRFToken:   "dummy_csrf",
				CurrentUser: session.Username,
				Action:      r.URL.Path,
				Mode:        "create1",
				Report: Report{
					IncidentTime: time.Now(),
					Author:       session.Username,
				},
				ValidationErrors: ve.Errors,
			}
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

	report := &Report{}
	if err := utils.Unmarshal(r, report); err != nil {
		slog.Error(fmt.Sprintf("could not unmarshal report data: %v", err))
		http.Error(w, "bad request", 400)
		return
	}
	report.Id = id
	if _, err := reportService.Update(*report, ""); err != nil {
		slog.Error(fmt.Sprintf("could not create new report version: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/reports", http.StatusSeeOther)
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
