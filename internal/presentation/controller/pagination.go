package controller

import commondto "github.com/QuantDevelopers/Backend/internal/application/dto/common"

func GetOffsetLimit(page, pageSize, defaultPage, defaultPageSize int) (int, int) {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}

	return (page - 1) * pageSize, pageSize
}

type PaginationMeta = commondto.PaginationMeta

type PaginatedResponse[T any] struct {
	Data       []T             `json:"data"`
	Pagination *PaginationMeta `json:"pagination"`
}

func NewPaginatedResponse[T any](data []T, totalItems int64, offset, limit int) *PaginatedResponse[T] {
	meta := commondto.BuildPaginationMeta(totalItems, offset, limit)
	return &PaginatedResponse[T]{
		Data:       data,
		Pagination: &meta,
	}
}
