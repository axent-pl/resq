package handler

import (
	"net/http"
	"net/url"
	"strings"

	"axent.pl/resq/dto"
	"axent.pl/resq/service"
)

func templateVariantFromRequest(r *http.Request) service.TemplateVariant {
	if strings.EqualFold(r.Header.Get("HX-Request"), "true") {
		return service.TemplateVariantXHR
	}
	if strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") {
		return service.TemplateVariantXHR
	}
	return service.TemplateVariantStandard
}

func optionsMapToSelectOptions(options map[string]string, selected *string, withEmpty bool) []dto.SelectOptionDTO {
	selectedValue := ""
	if selected != nil {
		selectedValue = *selected
	}
	sOptions := []dto.SelectOptionDTO{}
	if withEmpty {
		sOptions = append(sOptions, dto.SelectOptionDTO{
			Value: "",
			Label: "",
		})
	}
	for k, v := range options {
		sOptions = append(sOptions, dto.SelectOptionDTO{
			Value: k,
			Label: v,
		})
	}
	for i := range sOptions {
		sOptions[i].Selected = sOptions[i].Value == selectedValue
	}
	return sOptions
}

func updatedQuery(values url.Values, key string, val string) string {
	next := make(url.Values, len(values))
	for key, value := range values {
		next[key] = append([]string(nil), value...)
	}
	next.Set(key, val)
	return next.Encode()
}
