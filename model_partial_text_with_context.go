package main

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

type PartialtextWithContext struct {
	Id       string   `json:"id"`
	Pages    []int    `json:"pages"`
	Lines    []int    `json:"lines"`
	Text     string   `json:"text"`
	BBs      []*BB    `json:"bbs"`
	ImageIds []string `json:"imageIDs"`
	Key      string   `json:"-"`
}

func NewPartialTextWithContext(id string, bs *BookSource, s, t string) (*PartialtextWithContext, error) {
	idx := strings.Index(bs.Text, t)
	if idx == -1 {
		return nil, fmt.Errorf("partial text not found: id:%s; sourceid:%s; searched:%s", id, bs.Metadata.Bid, bs.Text[:48])
	}

	bPos := len([]rune(bs.Text[:idx]))
	bPageIdx := sort.Search(len(bs.Pbs),
		func(i int) bool { return bs.Pbs[i] > bPos }) - 1
	bPageLineIdx := slices.Index(bs.Lbs, bs.Pbs[bPageIdx]) // != -1
	bLineIdx := sort.Search(len(bs.Lbs),
		func(i int) bool { return bs.Lbs[i] > bPos }) - 1

	ePos := bPos + len([]rune(t)) - 1
	ePageIdx := sort.Search(len(bs.Pbs),
		func(i int) bool { return bs.Pbs[i] > ePos }) - 1
	ePageLineIdx := slices.Index(bs.Lbs, bs.Pbs[ePageIdx]) // != -1
	eLineIdx := sort.Search(len(bs.Lbs),
		func(i int) bool { return bs.Lbs[i] > ePos }) - 1

	imageIds := make([]string, 0, eLineIdx-bLineIdx+1)
	p := bPageIdx
	for i := bLineIdx; i <= eLineIdx; i++ {
		if bs.Lbs[i] == bs.Pbs[p+1] {
			p += 1
		}
		imageIds = append(imageIds, bs.Images[p])
	}

	return &PartialtextWithContext{
		Id:       id,
		Pages:    []int{bPageIdx, ePageIdx},
		Lines:    []int{bLineIdx - bPageLineIdx, eLineIdx - ePageLineIdx},
		Text:     s,
		BBs:      bs.BBs[bLineIdx : eLineIdx+1],
		ImageIds: imageIds,
		Key:      fmt.Sprintf("%s_%04d_%04d", id, bPageIdx+1, bLineIdx+1),
	}, nil
}
