package main

import (
	"fmt"
	"math"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

// /* GET */

// GetCount
func (es *ES) GetCount(c echo.Context) error {
	data, err := es.CountRecord()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	cnt, err := NewRecordCount(data)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, cnt)
}

// GetNgramSearch
func (es *ES) GetNgramSearch(c echo.Context) error {
	var sp TextSearchParam
	err := echo.QueryParamsBinder(c).
		BindWithDelimiter("q[]", &sp.Words, ",").
		BindWithDelimiter("q", &sp.Words, ",").
		CustomFunc("el[]", ELevelValueBinder(&sp)).
		CustomFunc("el", ELevelValueBinder(&sp)).
		BindWithDelimiter("tag[]", &sp.Tags, ",").
		BindWithDelimiter("tag", &sp.Tags, ",").
		BindWithDelimiter("bid[]", &sp.Bids, ",").
		BindWithDelimiter("bid", &sp.Bids, ",").
		Int("page", &sp.Page).
		Int("perPage", &sp.PerPage).
		BindError()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Errorf("query error: %s", err))
	}

	if len(sp.Words) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Errorf("query missing"))
	}

	var sr *TextSearchResult

	key := sp.GetCacheKey()
	cache, found := es.Cache.Get(key)
	if found {
		sr = cache.(*TextSearchResult)
	} else {
		data, err := es.SearchText(&sp)

		sr, err = NewTextSearchResult(&sp, data)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}

		es.Cache.Set(key, sr, 1)
	}

	total := len(sr.Matches)
	page := sp.Page
	if page == 0 {
		page = 1
	}
	perPage := sp.PerPage
	if perPage == 0 {
		perPage = 20
	}
	from := (page - 1) * perPage
	if from > total || from < 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Errorf("page should be: 1 <= page=%d <= %d",
				page, int(math.Ceil(float64(total)/float64(perPage)))))
	}
	till := from + perPage
	if till > total {
		till = total
	}

	return c.JSON(http.StatusOK, &TextSearchResult{
		Filters: sr.Filters,
		Bibl:    sr.Bibl,
		Matches: sr.Matches[from:till],
		Total:   total,
		Page:    page,
		PerPage: perPage,
	})
}

// /* POST */

// PostRegister
func (es *ES) PostRegister(c echo.Context) error {
	// get params
	var rp RegisterParam
	if err := c.Bind(&rp); err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest, fmt.Errorf("bind param: %s", err))
	}

	if len(rp.LocalPath) == 0 {
		file, err := c.FormFile("file")
		if err != nil {
			return echo.NewHTTPError(
				http.StatusBadRequest, fmt.Errorf("param \"file\": %s", err))
		}

		// prepare a tmp dir
		destdir, err := os.MkdirTemp("", "tmp-")
		if err != nil {
			return echo.NewHTTPError(
				http.StatusBadRequest, fmt.Errorf("make a tmp dir: %s", err))
		}
		rp.LocalPath = []string{destdir}

		// unzip the uploaded file at the tmp dir
		if err := unzipUploaded(file, destdir); err != nil {
			return echo.NewHTTPError(
				http.StatusBadRequest, fmt.Errorf("unzipUploaded: %s", err))
		}
	}

	// create es bt
	bt, err := NewBookText(&rp)
	if err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest, fmt.Errorf("createESDataFromDir: %s", err))
	}

	// set metadata from manifest
	if err := bt.FetchKokushoMetadata(); err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest, fmt.Errorf("get metadata: %s", err))
	}

	// index it
	if err := es.IndexBookData(bt); err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest, fmt.Errorf("IndexBookData: %s", err))
	}

	return c.JSON(http.StatusOK, rp)
}

// PostBulkRegister
func (es *ES) PostBulkRegister(c echo.Context) error {
	// get params
	var brp BulkRegisterParam
	if err := c.Bind(&brp); err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest, fmt.Errorf("bind param: %s", err))
	}

	fh, err := c.FormFile("listcsv")
	if err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest, fmt.Errorf("param \"listcsv\": %s", err))
	}
	brp.ListFileHeader = fh

	csv, err := brp.BulkIndexData()
	if err != nil {
		return echo.NewHTTPError(
			http.StatusBadRequest, fmt.Errorf("bulk error: %s", err))
	}

	return c.JSON(http.StatusOK, csv)
}
