package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

/* IIIFManifestMetadata */
type IIIFManifestMetadata struct {
	Label       string        `json:"label"`
	Metadata    []*LabelValue `json:"metadata"`
	Attribution string        `json:"attribution"`
	License     string        `json:"license"`
	Sequences   []struct {
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
	Bid    string   `json:"bid"`
	Cid    string   `json:"cid"`
	ELevel ELevel   `json:"elevel"`
	Tags   []string `json:"tags"`
	// derived from IIIF manifest
	Label       string        `json:"label"`
	Metadata    []*LabelValue `json:"metadata"`
	Attribution string        `json:"attribution"`
	License     string        `json:"license"`
	Images      []string      `json:"images"`
	// derived from OCR
	Text string `json:"text"`
	Pbs  []int  `json:"pbs"`
	Lbs  []int  `json:"lbs"`
	BBs  []*BB  `json:"bbs"`
	// derived from MeCab
	MecabType string   `json:"mecabType"`
	Mecabed   []string `json:"mecabed"`
}

/* BookMetadata */
type BookMetadata struct {
	Bid         string        `json:"bid"`
	Cid         string        `json:"cid"`
	ELevel      ELevel        `json:"elevel"`
	Tags        []string      `json:"tags"`
	Label       string        `json:"label"`
	Metadata    []*LabelValue `json:"metadata"`
	Attribution string        `json:"attribution"`
	License     string        `json:"license"`
}

func NewBookText(rp *RegisterParam) (bt *BookText, err error) {
	ois, err := NewOCRInfos(rp)
	if err != nil {
		return nil, err
	}

	switch rp.Type {
	case "ndlocrv1":
		bt, err = NdlOcrV12BookText(ois)
	case "ndlocrv2":
		bt, err = NdlOcrV22BookText(ois, false)
	case "ndlocrv2detail":
		bt, err = NdlOcrV22BookText(ois, true)
	case "ndlocrv3":
		bt, err = NdlOcrV32BookText(ois, false)
	case "ndlocrv3detail":
		bt, err = NdlOcrV32BookText(ois, true)
	default:
		return nil, fmt.Errorf("wrong type: %s", rp.Type)
	}
	if err != nil {
		return nil, err
	}

	bt.Bid = rp.Bid
	bt.Cid = rp.Cid
	bt.ELevel = OCR
	bt.Tags = []string{rp.Type}

	if err = bt.SetMecabType(rp.MecabType); err != nil {
		return nil, err
	}

	return bt, nil
}

func (bt *BookText) GetId_() string {
	return fmt.Sprintf("%s_%s_%s", bt.Bid, bt.ELevel,
		strings.Join(bt.Tags, "_"))
}

func (bt *BookText) GetMetadata() *BookMetadata {
	return &BookMetadata{
		Bid:         bt.Bid,
		Cid:         bt.Cid,
		ELevel:      bt.ELevel,
		Tags:        bt.Tags,
		Label:       bt.Label,
		Metadata:    bt.Metadata,
		Attribution: bt.Attribution,
		License:     bt.License,
	}
}

func (bt *BookText) FetchKokushoMetadata() error {
	url := "https://kokusho.nijl.ac.jp/biblio/" + bt.Bid + "/manifest"
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http request failed: %s: %s", url, err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("response reading failed: %s: %s", url, err)
	}

	var m IIIFManifestMetadata
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("response parsing failed: %s: %s", url, err)
	}

	bt.Label = m.Label
	bt.Metadata = m.Metadata
	bt.Attribution = m.Attribution
	bt.License = m.License

	canvases := m.Sequences[0].Canvases
	bt.Images = make([]string, len(canvases))
	for i := 0; i < len(canvases); i++ {
		id := canvases[i].Images[0].Resource.ID
		idx := strings.Index(id, ".tif/")
		bt.Images[i] = id[0 : idx+4]
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
