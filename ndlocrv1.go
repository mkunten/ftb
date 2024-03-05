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

type NdlOcrV1Line struct {
	BoundingBox NdlOcrV1BoundingBox `json:"boundingBox"`
	Confidence  int                 `json:"confidence"`
	ID          int                 `json:"id"`
	IsTextline  string              `json:"isTextline"`
	IsVertical  string              `json:"isVertical"`
	Text        string              `json:"text"`
}

type NdlOcrV1Page []NdlOcrV1Line

type NdlOcrV1Book []NdlOcrV1Page

type NdlOcrV1BoundingBox [4][2]int

// NdlOcrV12BookText convert ndlkotenocr result to *BookText
func NdlOcrV12BookText(dir string, start, end int) (*BookText, error) {
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

		var book NdlOcrV1Book
		if err := json.Unmarshal(raw, &book); err != nil {
			return nil, err
		}

		for _, page := range book {
			pbs = append(pbs, pos)
			for _, line := range page {
				sb.WriteString(line.Text)
				lbs = append(lbs, pos)
				pos += utf8.RuneCountInString(line.Text)
				x := line.BoundingBox[0][0]
				y := line.BoundingBox[0][1]
				bbs = append(bbs, &BB{
					X:      x,
					Y:      y,
					Width:  line.BoundingBox[3][0] - x,
					Height: line.BoundingBox[3][1] - y,
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
