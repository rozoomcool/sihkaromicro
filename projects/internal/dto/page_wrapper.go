package dto

type PageWrapper[T any] struct {
	Total int64
	Data  []T
}
