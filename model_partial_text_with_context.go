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

func NewPartialTextWithContext(id string, bt *BookText, s, t string) (*PartialtextWithContext, error) {
	idx := strings.Index(bt.Text, t)
	if idx == -1 {
		return nil, fmt.Errorf("partial text not found: id:%s; sourceid:%s; searched:%s", id, bt.Bid, bt.Text[:48])
	}

	bPos := len([]rune(bt.Text[:idx]))
	bPageIdx := sort.Search(len(bt.Pbs),
		func(i int) bool { return bt.Pbs[i] > bPos }) - 1
	bPageLineIdx := slices.Index(bt.Lbs, bt.Pbs[bPageIdx]) // != -1
	bLineIdx := sort.Search(len(bt.Lbs),
		func(i int) bool { return bt.Lbs[i] > bPos }) - 1

	ePos := bPos + len([]rune(t)) - 1
	ePageIdx := sort.Search(len(bt.Pbs),
		func(i int) bool { return bt.Pbs[i] > ePos }) - 1
	ePageLineIdx := slices.Index(bt.Lbs, bt.Pbs[ePageIdx]) // != -1
	eLineIdx := sort.Search(len(bt.Lbs),
		func(i int) bool { return bt.Lbs[i] > ePos }) - 1

	imageIds := make([]string, 0, eLineIdx-bLineIdx+1)
	p := bPageIdx
	for i := bLineIdx; i <= eLineIdx; i++ {
		if bt.Lbs[i] == bt.Pbs[p+1] {
			p += 1
		}
		imageIds = append(imageIds, bt.Images[p])
	}

	return &PartialtextWithContext{
		Id:       id,
		Pages:    []int{bPageIdx, ePageIdx},
		Lines:    []int{bLineIdx - bPageLineIdx, eLineIdx - ePageLineIdx},
		Text:     s,
		BBs:      bt.BBs[bLineIdx : eLineIdx+1],
		ImageIds: imageIds,
		Key:      fmt.Sprintf("%s_%04d_%04d", id, bPageIdx+1, bLineIdx+1),
	}, nil
}
