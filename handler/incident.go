package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"axent.pl/resq/dto"
	"axent.pl/resq/model"
	"axent.pl/resq/router"
	"axent.pl/resq/service"
)

func IncidentListHandler(incidentService *service.IncidentService, templateService *service.TemplateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Unmarshal DTO
		requestDTO, err := dto.UnmarshalDTO[dto.IncidentsListQueryDTO](r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Validate DTO
		if err = dto.ValidateDTO(requestDTO); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Map DTO to Model
		filter, pagingQuery, err := requestDTO.MapToModel()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Call Service
		incidents, pagingResult, err := incidentService.ListIncidents(r.Context(), filter, pagingQuery)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Map TemplateDTO
		data := dto.IncidentListTemplateDTO{
			BaseTemplateDTO: &dto.BaseTemplateDTO{
				Title: "Incidents",
			},
			Incidents:     incidents,
			Filter:        filter,
			Paging:        pagingResult,
			PrevPageQuery: updatedQuery(r.URL.Query(), "page", strconv.Itoa(pagingResult.PrevPage)),
			NextPageQuery: updatedQuery(r.URL.Query(), "page", strconv.Itoa(pagingResult.NextPage)),
			StatusOptions: optionsMapToSelectOptions(model.Incident{}.StatusOptions(), requestDTO.Status, true),
		}
		// Render
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templateService.Render(r, w, "incidents_list", templateVariantFromRequest(r), data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func IncidentReadHandler(incidentService *service.IncidentService, templateService *service.TemplateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Unmarshal
		rawIncidentID := router.PathParam(r, "id")
		incidentID, err := strconv.ParseUint(rawIncidentID, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			slog.Warn("incident read error", "error", err)
			return
		}
		// Call Service
		incident, err := incidentService.GetIncident(r.Context(), uint(incidentID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			slog.Warn("incident read error", "error", err)
			return
		}
		// Map TemplateDTO
		data := dto.IncidentReadTemplateDTO{
			BaseTemplateDTO: &dto.BaseTemplateDTO{
				Title: incident.Location,
			},
			Incident: incident,
		}
		// Render
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templateService.Render(r, w, "incidents_read", templateVariantFromRequest(r), data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func IncidentFormHandler(eventService *service.EventService, incidentService *service.IncidentService, templateService *service.TemplateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Map TemplateDTO
		now := time.Now().Truncate(time.Minute)
		form := dto.IncidentCreateRequestDTO{
			OccuredAt: now,
			Status:    "open",
			Patient: dto.PatientCreateRequestDTO{
				Sex: "unknown",
			},
			Report: dto.ReportCreateRequestDTO{
				Type:        "initial_assessment",
				PerformedAt: now,
				VitalMeasurements: dto.VitalMeasurementCreateRequestDTO{
					MeasuredAt: now,
					AVPU:       "alert",
				},
			},
		}
		data := dto.IncidentFormTemplateDTO{
			BaseTemplateDTO: &dto.BaseTemplateDTO{
				Title: "New incident",
			},
			Form:              form,
			Errors:            nil,
			EventOptions:      optionsMapToSelectOptions(eventService.EventOptions(r.Context()), nil, true),
			StatusOptions:     optionsMapToSelectOptions(model.Incident{}.StatusOptions(), &form.Status, false),
			SexOptions:        optionsMapToSelectOptions(model.Patient{}.SexOptions(), &form.Patient.Sex, false),
			ReportTypeOptions: optionsMapToSelectOptions(model.Report{}.TypeOptions(), &form.Report.Type, false),
			AVPUOptions:       optionsMapToSelectOptions(model.VitalMeasurement{}.AVPUOptions(), &form.Report.VitalMeasurements.AVPU, true),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templateService.Render(r, w, "incidents_form", templateVariantFromRequest(r), data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func IncidentCreateHandler(eventService *service.EventService, incidentService *service.IncidentService, templateService *service.TemplateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		createRequestDTO, err := dto.UnmarshalDTO[dto.IncidentCreateRequestDTO](r)
		if err != nil {
			slog.Warn("incident create unmarshal error", "unmarshal_errors", err)
			var validationErr dto.ValidationError
			if errors.As(err, &validationErr) {
				renderIncidentCreateFormErrors(w, r, eventService, incidentService, templateService, createRequestDTO, validationErr.Errors)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := dto.ValidateDTO(createRequestDTO); err != nil {
			slog.Warn("incident create validation error", "validation_errors", err)
			var validationErr dto.ValidationError
			if errors.As(err, &validationErr) {
				renderIncidentCreateFormErrors(w, r, eventService, incidentService, templateService, createRequestDTO, validationErr.Errors)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		incident, err := incidentService.CreateIncident(r.Context(), createRequestDTO.MapToModel())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if templateVariantFromRequest(r) != service.TemplateVariantXHR {
			http.Redirect(w, r, "/incident/"+strconv.FormatUint(uint64(incident.ID), 10), http.StatusSeeOther)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", "incident-created")
		data := dto.IncidentReadTemplateDTO{
			BaseTemplateDTO: &dto.BaseTemplateDTO{
				Title: incident.Location,
			},
			Incident: incident,
		}
		if err := templateService.Render(r, w, "incidents_read", service.TemplateVariantXHR, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func renderIncidentCreateFormErrors(w http.ResponseWriter, r *http.Request, eventService *service.EventService, incidentService *service.IncidentService, templateService *service.TemplateService, form dto.IncidentCreateRequestDTO, errs map[string][]string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	data := dto.IncidentFormTemplateDTO{
		BaseTemplateDTO: &dto.BaseTemplateDTO{
			Title: "New incident",
		},
		Form:              form,
		Errors:            errs,
		EventOptions:      optionsMapToSelectOptions(eventService.EventOptions(r.Context()), dto.UintPointerToStringPointer(form.EventID), true),
		StatusOptions:     optionsMapToSelectOptions(model.Incident{}.StatusOptions(), &form.Status, false),
		SexOptions:        optionsMapToSelectOptions(model.Patient{}.SexOptions(), &form.Patient.Sex, false),
		ReportTypeOptions: optionsMapToSelectOptions(model.Report{}.TypeOptions(), &form.Report.Type, false),
		AVPUOptions:       optionsMapToSelectOptions(model.VitalMeasurement{}.AVPUOptions(), &form.Report.VitalMeasurements.AVPU, true),
	}
	if renderErr := templateService.Render(r, w, "incidents_form", service.TemplateVariantXHR, data); renderErr != nil {
		http.Error(w, renderErr.Error(), http.StatusInternalServerError)
	}
}
