package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

/* BookMetadata */
type BookMetadata struct {
	Bid         string        `json:"bid"`
	Cid         string        `json:"cid"`
	ELevel      ELevel        `json:"elevel"`
	Tags        []string      `json:"tags"`
	Metadata    []*LabelValue `json:"metadata"`
	Label       string        `json:"label"`
	Attribution string        `json:"attribution"`
	License     string        `json:"license"`
}

type BookMetadataForUnmarshal struct {
	BookMetadata
	Sequences []struct {
		Canvases []struct {
			Images []struct {
				Resource struct {
					ID string `json:"@id"`
				} `json:"resource"`
			} `json:"images"`
		} `json:"canvases"`
	} `json:"sequences"`
}

/* LabelValue */
type LabelValue struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

/* BB */
type BB struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"w"`
	Height int `json:"h"`
}

/* BookText */
type BookText struct {
	Metadata  *BookMetadata `json:"metadata"`
	Text      string        `json:"text"`
	Pbs       []int         `json:"pbs"`
	Lbs       []int         `json:"lbs"`
	BBs       []*BB         `json:"bbs"`
	Images    []string      `json:"images"`
	MecabType string        `json:"mecabType"`
	Mecabed   []string      `json:"mecabed"`
	StartPos  int           `json:"-"`
	EndPos    int           `json:"-"`
}

func NewBookText(rp *RegisterParam) (bt *BookText, err error) {
	switch rp.Type {
	case "ndlocrv1":
		bt, err = NdlOcrV12BookText(rp.LocalPath, rp.StartPos, rp.EndPos)
	case "ndlocrv2":
		bt, err = NdlOcrV22BookText(rp.LocalPath, rp.StartPos, rp.EndPos, false)
	case "ndlocrv2detail":
		bt, err = NdlOcrV22BookText(rp.LocalPath, rp.StartPos, rp.EndPos, true)
	case "ndlocrv3":
		bt, err = NdlOcrV32BookText(rp.LocalPath, rp.StartPos, rp.EndPos, false)
	case "ndlocrv3detail":
		bt, err = NdlOcrV32BookText(rp.LocalPath, rp.StartPos, rp.EndPos, true)
	default:
		return nil, fmt.Errorf("wrong type: %s", rp.Type)
	}
	if err != nil {
		return nil, err
	}

	if bt.Metadata == nil {
		bt.Metadata = &BookMetadata{}
	}
	bt.Metadata.Bid = rp.Bid
	bt.Metadata.Cid = rp.Cid
	bt.Metadata.ELevel = OCR
	bt.Metadata.Tags = []string{rp.Type}
	bt.StartPos = rp.StartPos
	bt.EndPos = rp.EndPos

	if err = bt.SetMecabType(rp.MecabType); err != nil {
		return nil, err
	}

	return bt, nil
}

func (bt *BookText) GetId_() string {
	return fmt.Sprintf("%s_%s_%s", bt.Metadata.Bid, bt.Metadata.ELevel,
		strings.Join(bt.Metadata.Tags, "_"))
}

func (bt *BookText) FetchKokushoMetadata() error {
	url := "https://kokusho.nijl.ac.jp/biblio/" + bt.Metadata.Bid + "/manifest"
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http request failed: %s: %s", url, err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("response reading failed: %s: %s", url, err)
	}

	var mdu BookMetadataForUnmarshal
	if err := json.Unmarshal(data, &mdu); err != nil {
		return fmt.Errorf("response parsing failed: %s: %s", url, err)
	}

	if bt.Metadata == nil {
		bt.Metadata = &BookMetadata{}
	}
	bt.Metadata.Metadata = mdu.Metadata
	bt.Metadata.Label = mdu.Label
	bt.Metadata.Attribution = mdu.Attribution
	bt.Metadata.License = mdu.License

	canvases := mdu.Sequences[0].Canvases
	start := bt.StartPos - 1
	if start < 0 {
		start = 0
	}
	end := len(canvases)
	if bt.EndPos != 0 && bt.EndPos < end {
		end = bt.EndPos
	}
	bt.Images = make([]string, end-start)
	for i := 0; i < end-start; i++ {
		bt.Images[i] = canvases[i+start].Images[0].Resource.ID
		idx := strings.Index(bt.Images[i], ".tif/")
		bt.Images[i] = bt.Images[i][0 : idx+4]
	}

	return nil
}

func (bt *BookText) SetMecabType(mecabType string) error {
	if mecabType == "" {
		return nil
	}

	keys, err := MecabFilter(mecabType, bt.Text)
	if err != nil {
		return err
	}

	bt.MecabType = mecabType
	bt.Mecabed = keys
	return nil
}

func (bt *BookText) GetText(page, line int) string {
	if page >= len(bt.Pbs) {
		return ""
	}
	lidx := slices.Index(bt.Lbs, bt.Pbs[page-1]) + line - 1
	if lidx >= len(bt.Lbs) {
		return ""
	}
	startPos := bt.Lbs[lidx]
	if lidx == len(bt.Lbs)-1 {
		return string([]rune(bt.Text)[startPos:])
	}
	endPos := bt.Lbs[lidx+1]
	return string([]rune(bt.Text)[startPos:endPos])
}
