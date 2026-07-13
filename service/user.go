package service

import (
	"context"
	"fmt"
	"strings"

	"axent.pl/resq/model"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) ListUsers(ctx context.Context, filter model.UserFilter, paging model.PagingQuery) ([]model.User, model.PagingResult, error) {
	paging = normalizePaging(paging)

	query := applyUserFilter(s.db.WithContext(ctx).Model(&model.User{}), filter)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, model.PagingResult{}, err
	}

	var users []model.User
	listQuery := applyUserFilter(s.db.WithContext(ctx).Model(&model.User{}), filter)
	err := listQuery.
		Order(userOrderClause(paging)).
		Limit(paging.Size).
		Offset((paging.Number - 1) * paging.Size).
		Find(&users).
		Error
	if err != nil {
		return nil, model.PagingResult{}, err
	}

	return users, pageResult(paging, total), nil
}

func (s *UserService) CreateUser(ctx context.Context, user model.User) (model.User, error) {
	if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
		return model.User{}, err
	}

	return s.ReadUser(ctx, user.ID)
}

func (s *UserService) ReadUser(ctx context.Context, id uint) (model.User, error) {
	var user model.User
	err := s.db.WithContext(ctx).
		First(&user, id).
		Error
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

func applyUserFilter(query *gorm.DB, filter model.UserFilter) *gorm.DB {
	if filter.Role != nil && strings.TrimSpace(*filter.Role) != "" {
		query = query.Where("role = ?", strings.TrimSpace(*filter.Role))
	}
	if filter.Search != nil && strings.TrimSpace(*filter.Search) != "" {
		search := "%" + strings.TrimSpace(*filter.Search) + "%"
		query = query.Where(
			"username LIKE ? OR display_name LIKE ? OR email LIKE ? OR phone LIKE ?",
			search,
			search,
			search,
			search,
		)
	}

	return query
}

func userOrderClause(paging model.PagingQuery) string {
	sortBy := map[string]string{
		"id":           "id",
		"username":     "username",
		"display_name": "display_name",
		"email":        "email",
		"phone":        "phone",
		"role":         "role",
		"created_at":   "created_at",
		"updated_at":   "updated_at",
	}[paging.SortBy]
	if sortBy == "" {
		sortBy = "username"
	}

	return fmt.Sprintf("%s %s", sortBy, strings.ToUpper(paging.Order))
}
