package model

type PagingQuery struct {
	Number int
	Size   int
	SortBy string
	Order  string
}

type PagingResult struct {
	Number     int
	Size       int
	TotalItems int64
	TotalPages int
	NextPage   int
	PrevPage   int
	SortBy     string
	Order      string
}
