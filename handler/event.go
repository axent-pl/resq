package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"axent.pl/resq/dto"
	"axent.pl/resq/router"
	"axent.pl/resq/service"
)

func EventListHandler(eventService *service.EventService, templateService *service.TemplateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Unmarshal DTO
		requestDTO, err := dto.UnmarshalDTO[dto.EventsListQueryDTO](r)
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
		events, pagingResult, err := eventService.ListEvents(r.Context(), filter, pagingQuery)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Map TemplateDTO
		data := dto.EventListTemplateDTO{
			BaseTemplateDTO: &dto.BaseTemplateDTO{
				Title: "Events",
			},
			Events:        events,
			Filter:        filter,
			Paging:        pagingResult,
			PrevPageQuery: updatedQuery(r.URL.Query(), "page", strconv.Itoa(pagingResult.PrevPage)),
			NextPageQuery: updatedQuery(r.URL.Query(), "page", strconv.Itoa(pagingResult.NextPage)),
		}
		// Render
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templateService.Render(r, w, "events_list", templateVariantFromRequest(r), data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func EventsReadHandler(eventService *service.EventService, templateService *service.TemplateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Unmarshal
		rawEventID := router.PathParam(r, "id")
		eventID, err := strconv.ParseUint(rawEventID, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			slog.Warn("event read error", "error", err)
			return
		}
		// Call Service
		event, err := eventService.GetEvent(r.Context(), uint(eventID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			slog.Warn("event read error", "error", err)
			return
		}
		// Map TemplateDTO
		data := dto.EventReadTemplateDTO{
			BaseTemplateDTO: &dto.BaseTemplateDTO{
				Title: event.Location,
			},
			Event: event,
		}
		// Render
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templateService.Render(r, w, "events_read", templateVariantFromRequest(r), data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func EventFormHandler(eventService *service.EventService, templateService *service.TemplateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Map TemplateDTO
		form := dto.EventCreateRequestDTO{}
		data := dto.EventFormTemplateDTO{
			BaseTemplateDTO: &dto.BaseTemplateDTO{
				Title: "New event",
			},
			Form:   form,
			Errors: nil,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templateService.Render(r, w, "events_form", templateVariantFromRequest(r), data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func EventCreateHandler(eventService *service.EventService, templateService *service.TemplateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		createRequestDTO, err := dto.UnmarshalDTO[dto.EventCreateRequestDTO](r)
		if err != nil {
			slog.Warn("event create unmarshal error", "unmarshal_errors", err)
			var validationErr dto.ValidationError
			if errors.As(err, &validationErr) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusUnprocessableEntity)
				data := dto.EventFormTemplateDTO{
					BaseTemplateDTO: &dto.BaseTemplateDTO{
						Title: "New event",
					},
					Form:   createRequestDTO,
					Errors: validationErr.Errors,
				}
				if renderErr := templateService.Render(r, w, "events_form", service.TemplateVariantXHR, data); renderErr != nil {
					http.Error(w, renderErr.Error(), http.StatusInternalServerError)
				}
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := dto.ValidateDTO(createRequestDTO); err != nil {
			slog.Warn("event create validation error", "validation_errors", err)
			var validationErr dto.ValidationError
			if errors.As(err, &validationErr) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusUnprocessableEntity)
				data := dto.EventFormTemplateDTO{
					BaseTemplateDTO: &dto.BaseTemplateDTO{
						Title: "New event",
					},
					Form:   createRequestDTO,
					Errors: validationErr.Errors,
				}
				if renderErr := templateService.Render(r, w, "events_form", service.TemplateVariantXHR, data); renderErr != nil {
					http.Error(w, renderErr.Error(), http.StatusInternalServerError)
				}
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		event, err := eventService.CreateEvent(r.Context(), createRequestDTO.MapToModel())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if templateVariantFromRequest(r) != service.TemplateVariantXHR {
			http.Redirect(w, r, "/event/"+strconv.FormatUint(uint64(event.ID), 10), http.StatusSeeOther)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", "event-created")
		data := dto.EventReadTemplateDTO{
			BaseTemplateDTO: &dto.BaseTemplateDTO{
				Title: event.Name,
			},
			Event: event,
		}
		if err := templateService.Render(r, w, "events_read", service.TemplateVariantXHR, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
