package dto

import "axent.pl/resq/model"

type UserListTemplateDTO struct {
	*BaseTemplateDTO
	Users         []model.User
	Filter        model.UserFilter
	Paging        model.PagingResult
	PrevPageQuery string
	NextPageQuery string
}

type UserFormTemplateDTO struct {
	*BaseTemplateDTO
	Form   UserCreateRequestDTO
	Errors map[string][]string
}

type UserReadTemplateDTO struct {
	*BaseTemplateDTO
	User model.User
}
