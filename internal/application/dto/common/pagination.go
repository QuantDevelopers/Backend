package commondto

// PaginationMeta describes page bounds and totals for paginated REST responses.
type PaginationMeta struct {
	CurrentPage int   `json:"currentPage"`
	PageSize    int   `json:"pageSize"`
	TotalItems  int64 `json:"totalItems"`
	TotalPages  int   `json:"totalPages"`
	HasNextPage bool  `json:"hasNextPage"`
	HasPrevPage bool  `json:"hasPrevPage"`
}

// BuildPaginationMeta builds PaginationMeta from offset/limit and total record counts.
func BuildPaginationMeta(totalItems int64, offset, limit int) PaginationMeta {
	pageSize := limit
	currentPage := 1
	if limit > 0 {
		currentPage = (offset / limit) + 1
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = int((totalItems + int64(pageSize) - 1) / int64(pageSize))
	}

	return PaginationMeta{
		CurrentPage: currentPage,
		PageSize:    pageSize,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		HasNextPage: currentPage < totalPages,
		HasPrevPage: currentPage > 1,
	}
}
