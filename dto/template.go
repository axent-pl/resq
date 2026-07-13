package dto

import (
	"axent.pl/resq/model"
)

type SelectOptionDTO struct {
	Value    string
	Label    string
	Selected bool
}

type TemplaterDTO interface {
	SetLang(lang string)
}

type BaseTemplateDTO struct {
	Lang  string
	Title string
}

func (t *BaseTemplateDTO) SetLang(lang string) {
	t.Lang = lang
}

type IncidentListTemplateDTO struct {
	*BaseTemplateDTO
	Incidents     []model.Incident
	Filter        model.IncidentFilter
	Paging        model.PagingResult
	PrevPageQuery string
	NextPageQuery string
	StatusOptions []SelectOptionDTO
}

type IncidentReadTemplateDTO struct {
	*BaseTemplateDTO
	Incident model.Incident
}

type IncidentFormTemplateDTO struct {
	*BaseTemplateDTO
	Form              IncidentCreateRequestDTO
	Errors            map[string][]string
	EventOptions      []SelectOptionDTO
	StatusOptions     []SelectOptionDTO
	SexOptions        []SelectOptionDTO
	ReportTypeOptions []SelectOptionDTO
	AVPUOptions       []SelectOptionDTO
}
