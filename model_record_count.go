package main

import (
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
)

type RecordCount struct {
	RecordCount map[string]int `json:"recordCount"`
}

func NewRecordCount(res *search.Response) (*RecordCount, error) {
	fa := res.Aggregations["recordCount"].(*types.FiltersAggregate)
	fb := fa.Buckets.(map[string]types.FiltersBucket)
	rc := RecordCount{
		RecordCount: map[string]int{},
	}
	for key, value := range fb {
		el, err := ELevelString(key)
		if err != nil {
			return nil, err
		}
		rc.RecordCount[el.String()] = int(value.DocCount)
	}
	return &rc, nil
}
