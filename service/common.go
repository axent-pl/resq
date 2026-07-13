package service

import "axent.pl/resq/model"

func normalizePaging(paging model.PagingQuery) model.PagingQuery {
	if paging.Number < 1 {
		paging.Number = 1
	}
	if paging.Size < 1 {
		paging.Size = 20
	}
	if paging.Size > 100 {
		paging.Size = 100
	}
	if paging.Order != "asc" && paging.Order != "desc" {
		paging.Order = "desc"
	}

	return paging
}

func pageResult(pagingQuery model.PagingQuery, total int64) model.PagingResult {
	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pagingQuery.Size) - 1) / int64(pagingQuery.Size))
	}

	return model.PagingResult{
		Number:     pagingQuery.Number,
		Size:       pagingQuery.Size,
		TotalItems: total,
		TotalPages: totalPages,
		PrevPage:   max(pagingQuery.Number-1, 0),
		NextPage:   min(pagingQuery.Number+1, int(total)),
		SortBy:     pagingQuery.SortBy,
		Order:      pagingQuery.Order,
	}
}
