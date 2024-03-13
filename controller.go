package main

import (
	"fmt"
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
		BindError()
	if len(sp.Words) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			fmt.Errorf("query missing"))
	}

	data, err := es.SearchText(&sp)

	sr, err := NewTextSearchResult(&sp, data)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	return c.JSON(http.StatusOK, sr)
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
