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

func UserListHandler(userService *service.UserService, templateService *service.TemplateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Unmarshal DTO
		requestDTO, err := dto.UnmarshalDTO[dto.UsersListQueryDTO](r)
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
		users, pagingResult, err := userService.ListUsers(r.Context(), filter, pagingQuery)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Map TemplateDTO
		data := dto.UserListTemplateDTO{
			BaseTemplateDTO: &dto.BaseTemplateDTO{
				Title: "Users",
			},
			Users:         users,
			Filter:        filter,
			Paging:        pagingResult,
			PrevPageQuery: updatedQuery(r.URL.Query(), "page", strconv.Itoa(pagingResult.PrevPage)),
			NextPageQuery: updatedQuery(r.URL.Query(), "page", strconv.Itoa(pagingResult.NextPage)),
		}
		// Render
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templateService.Render(r, w, "users_list", templateVariantFromRequest(r), data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func UserReadHandler(userService *service.UserService, templateService *service.TemplateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Unmarshal
		rawUserID := router.PathParam(r, "id")
		userID, err := strconv.ParseUint(rawUserID, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			slog.Warn("user read error", "error", err)
			return
		}
		// Call Service
		user, err := userService.ReadUser(r.Context(), uint(userID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			slog.Warn("user read error", "error", err)
			return
		}
		// Map TemplateDTO
		data := dto.UserReadTemplateDTO{
			BaseTemplateDTO: &dto.BaseTemplateDTO{
				Title: user.Username,
			},
			User: user,
		}
		// Render
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templateService.Render(r, w, "users_read", templateVariantFromRequest(r), data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func UserFormHandler(userService *service.UserService, templateService *service.TemplateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Map TemplateDTO
		form := dto.UserCreateRequestDTO{}
		data := dto.UserFormTemplateDTO{
			BaseTemplateDTO: &dto.BaseTemplateDTO{
				Title: "New user",
			},
			Form:   form,
			Errors: nil,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := templateService.Render(r, w, "users_form", templateVariantFromRequest(r), data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func UserCreateHandler(userService *service.UserService, templateService *service.TemplateService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		createRequestDTO, err := dto.UnmarshalAndValidateDTO[dto.UserCreateRequestDTO](r)
		if err != nil {
			slog.Warn("user create unmarshal error", "unmarshal_errors", err)
			var validationErr dto.ValidationError
			if errors.As(err, &validationErr) {
				renderUserCreateFormErrors(w, r, templateService, createRequestDTO, validationErr.Errors)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		user, err := userService.CreateUser(r.Context(), createRequestDTO.MapToModel())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if templateVariantFromRequest(r) != service.TemplateVariantXHR {
			http.Redirect(w, r, "/users/"+strconv.FormatUint(uint64(user.ID), 10), http.StatusSeeOther)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("HX-Trigger", "user-created")
		data := dto.UserReadTemplateDTO{
			BaseTemplateDTO: &dto.BaseTemplateDTO{
				Title: user.Username,
			},
			User: user,
		}
		if err := templateService.Render(r, w, "users_read", service.TemplateVariantXHR, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func renderUserCreateFormErrors(w http.ResponseWriter, r *http.Request, templateService *service.TemplateService, form dto.UserCreateRequestDTO, errs map[string][]string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	data := dto.UserFormTemplateDTO{
		BaseTemplateDTO: &dto.BaseTemplateDTO{
			Title: "New user",
		},
		Form:   form,
		Errors: errs,
	}
	if renderErr := templateService.Render(r, w, "users_form", service.TemplateVariantXHR, data); renderErr != nil {
		http.Error(w, renderErr.Error(), http.StatusInternalServerError)
	}
}
