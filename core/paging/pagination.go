package paging

import (
	"math"
)

type Pagination struct {
	perPage     int
	totalAmount int
	currentPage int
	totalPage   int
	baseUrl     string

	// render parts
	firstPart  []string
	middlePart []string
	lastPart   []string
}

func New(totalAmount, perPage, currentPage int, baseUrl string) *Pagination {
	if currentPage == 0 {
		currentPage = 1
	}

	n := int(math.Ceil(float64(totalAmount) / float64(perPage)))
	if currentPage > n {
		currentPage = n
	}

	return &Pagination{
		perPage:     perPage,
		totalAmount: totalAmount,
		currentPage: currentPage,
		totalPage:   n,
		baseUrl:     baseUrl,
	}
}

func (p *Pagination) TotalPages() int {
	return p.totalPage
}

func (p *Pagination) HasPages() bool {
	return p.TotalPages() > 1
}
