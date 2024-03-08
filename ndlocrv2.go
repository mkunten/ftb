package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

type NdlOcrV2Line [5]interface{}

type NdlOcrV2Page []NdlOcrV2Line

type NdlOcrV2Book []NdlOcrV2Page

type NdlOcrV2ImgInfo struct {
	ImgWidth  int    `json:"img_width"`
	ImgHeight int    `json:"img_height"`
	Img_path  string `json:"img_path"`
	ImgName   string `json:"img_name"`
}

type NdlOcrV2BookDetail struct {
	Contents NdlOcrV2Page    `json:"contents"`
	ImgInfo  NdlOcrV2ImgInfo `json:"imginfo"`
}

// OCRResult2BookText convert ndlkotenocr result to *BookText
func NdlOcrV22BookText(
	dir string, start, end int, isDetail bool,
) (*BookText, error) {
	fmt.Println("ndlocrv2:", dir, start, end, isDetail)
	files, err := filepath.Glob(filepath.Join(dir, "**/json/*.json"))
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files found under %s", dir)
	}

	sort.Strings(files)
	if start <= 1 {
		if end != 0 && end < len(files) {
			files = files[:end]
		}
	} else if end != 0 && end < len(files) {
		files = files[start-1 : end]
	} else {
		files = files[start-1:]
	}

	var (
		sb  strings.Builder
		lbs []int
		pbs []int
		bbs []*BB
	)
	pos := 0

	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		var contents NdlOcrV2Book

		if isDetail {
			var book NdlOcrV2BookDetail
			if err := json.Unmarshal(raw, &book); err != nil {
				return nil, err
			}
			contents = NdlOcrV2Book{
				book.Contents,
			}
		} else {
			if err := json.Unmarshal(raw, &contents); err != nil {
				return nil, err
			}
		}

		for _, page := range contents {
			pbs = append(pbs, pos)
			for _, line := range page {
				if len(line) != 5 {
					continue
				}
				sb.WriteString(line[4].(string))
				lbs = append(lbs, pos)
				pos += utf8.RuneCountInString(line[4].(string))
				x := int(line[0].(float64))
				y := int(line[1].(float64))
				w := int(line[2].(float64)) - x
				h := int(line[3].(float64)) - y
				bbs = append(bbs, &BB{
					X:      x,
					Y:      y,
					Width:  w,
					Height: h,
				})
			}
		}
	}
	bt := &BookText{
		Text: sb.String(),
		Pbs:  pbs,
		Lbs:  lbs,
		BBs:  bbs,
	}

	// f, _ := json.MarshalIndent(bt, "", "  ")
	// dest := filepath.Base(dir) + ".json"
	// _ = os.WriteFile(dest, f, 0644)
	return bt, nil
}
