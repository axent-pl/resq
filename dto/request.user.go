package dto

import (
	"axent.pl/resq/model"
)

type UsersListQueryDTO struct {
	Search *string `form:"search"`
	Role   *string `form:"role"`
	Page   int     `form:"page"`
	Size   int     `form:"size"`
	SortBy string  `form:"sort_by"`
	Order  string  `form:"order"`
}

func (d *UsersListQueryDTO) MapToModel() (model.UserFilter, model.PagingQuery, error) {
	filter := model.UserFilter{
		Role:   d.Role,
		Search: d.Search,
	}
	paging := model.PagingQuery{
		Number: d.Page,
		Size:   d.Size,
		SortBy: d.SortBy,
		Order:  d.Order,
	}
	return filter, paging, nil
}

type UserCreateRequestDTO struct {
	Username    string `json:"username" form:"username" validate:"required"`
	DisplayName string `json:"display_name" form:"display_name" validate:"required"`
	Email       string `json:"email" form:"email" validate:"required"`
	Phone       string `json:"phone" form:"phone"`
	Role        string `json:"role" form:"role"`
}

func (d *UserCreateRequestDTO) MapToModel() model.User {
	return model.User{
		Username:    d.Username,
		DisplayName: d.DisplayName,
		Email:       d.Email,
		Phone:       d.Phone,
		Role:        d.Role,
	}
}
