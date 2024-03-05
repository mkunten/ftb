package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
)

/* TextSearchParam */
type TextSearchParam struct {
	Words   []string `query:"q" form:"q"`
	ELevels []ELevel `query:"el" form:"el"`
	Tags    []string `query:"tag" form:"tag"`
	Bids    []string `query:"bid" form:"bid"`
}

func (sp *TextSearchParam) GetQuery() *types.Query {
	qw := []types.Query{}
	for _, w := range sp.Words {
		qw = append(qw, types.Query{
			MatchPhrase: map[string]types.MatchPhraseQuery{
				cfg.IndexName: {
					Query: w,
				},
			},
		})
	}

	q := &types.Query{
		Bool: &types.BoolQuery{
			Filter: []types.Query{{
				Bool: &types.BoolQuery{
					Should: qw,
				},
			}},
		},
	}

	if len(sp.ELevels) > 0 {
		elstrs := make([]string, len(sp.ELevels))
		for idx, el := range sp.ELevels {
			elstrs[idx] = el.String()
		}
		q.Bool.Filter = append(q.Bool.Filter, types.Query{
			Nested: &types.NestedQuery{
				Path: "metadata",
				Query: &types.Query{
					Terms: &types.TermsQuery{
						TermsQuery: map[string]types.TermsQueryField{
							"metadata.elevel": elstrs,
						},
					},
				},
			},
		})
	}

	if len(sp.Tags) > 0 {
		q.Bool.Filter = append(q.Bool.Filter, types.Query{
			Nested: &types.NestedQuery{
				Path: "metadata",
				Query: &types.Query{
					Terms: &types.TermsQuery{
						TermsQuery: map[string]types.TermsQueryField{
							"metadata.tag": sp.Tags,
						},
					},
				},
			},
		})
	}

	if len(sp.Bids) > 0 {
		q.Bool.Filter = append(q.Bool.Filter, types.Query{
			Nested: &types.NestedQuery{
				Path: "metadata",
				Query: &types.Query{
					Terms: &types.TermsQuery{
						TermsQuery: map[string]types.TermsQueryField{
							"metadata.bid": sp.Bids,
						},
					},
				},
			},
		})
	}

	return q
}

/* TextSearchResult */
type TextSearchResult struct {
	Filters struct {
		Keyword TextSearchKeywordFilter `json:"keyword"`
		Tag     []LabelValue            `json:"tag"`
	} `json:"filters"`
	Bibl  map[string]*BookMetadata `json:"bibl"`
	Count struct {
		Total int `json:"total"`
	} `json:"count"`
	Matches []*PartialtextWithContext `json:"match"`
}

/* BookSource for es search result */
type BookSource struct {
	Metadata *BookMetadata `json:"metadata"`
	Text     string        `json:"text"`
	Pbs      []int         `json:"pbs"`
	Lbs      []int         `json:"lbs"`
	BBs      []*BB         `json:"bbs"`
	Images   []string      `json:"images"`
}

type TextSearchKeywordFilter map[string]map[string]map[string]map[string]int
type Q2Data struct {
	Id         string
	BookSource *BookSource
	Match      string
}

// NewTextSearchResult
func NewTextSearchResult(sp *TextSearchParam, res *search.Response) (*TextSearchResult, error) {
	var (
		bibls   = map[string]*BookMetadata{}
		kwf     = TextSearchKeywordFilter{}
		tag     = []LabelValue{}
		matches = []*PartialtextWithContext{}
	)

	var errs []string
	var wg1 sync.WaitGroup
	var wg2 sync.WaitGroup
	q1 := make(chan types.Hit, cfg.BulkWorkerNum)
	q2 := make(chan *Q2Data, 256)

	// prepare workers
	for i := 0; i < cfg.BulkWorkerNum; i++ {
		// csv row => BookText
		wg1.Add(1)
		go func(
			pwg1 *sync.WaitGroup,
			q1 chan types.Hit,
			q2 chan *Q2Data,
			bibls map[string]*BookMetadata,
			errs *[]string,
		) {
			defer pwg1.Done()
			for {
				hit, ok := <-q1
				if !ok {
					break
				}

				var bs BookSource
				if err := json.Unmarshal(hit.Source_, &bs); err != nil {
					mu.Lock()
					*errs = append(*errs, err.Error())
					mu.Unlock()
					break
				}

				if _, ok := bibls[hit.Id_]; !ok {
					mu.Lock()
					bibls[hit.Id_] = bs.Metadata
					mu.Unlock()
				}

				if hit.Id_[:9] != bs.Metadata.Bid {
					mu.Lock()
					*errs = append(
						*errs,
						fmt.Sprintf("q1:id:%s; bid:%s", hit.Id_, bs.Metadata.Bid),
					)
					mu.Unlock()
					continue
				}

				for _, match := range hit.Highlight["text"] {
					q2 <- &Q2Data{
						Id:         hit.Id_,
						BookSource: &bs,
						Match:      match,
					}
				}
			}
		}(&wg1, q1, q2, bibls, &errs)

		wg2.Add(1)
		go func(
			pwg2 *sync.WaitGroup,
			q2 chan *Q2Data,
			kwf TextSearchKeywordFilter,
			pmatches *[]*PartialtextWithContext,
			errs *[]string,
		) {
			defer pwg2.Done()
			for {
				q, ok := <-q2
				if !ok {
					break
				}

				id := q.Id
				bs := q.BookSource
				elevel := bs.Metadata.ELevel.String()
				s := q.Match
				t := ""
				offset := 0

				for {
					end := strings.Index(s[offset:], "<em class=\"hlt")
					if end == -1 {
						break
					}
					end += offset
					t += s[offset:end]
					offset = end + 11
					end = offset + strings.Index(s[offset:], "\">")
					key := s[offset:end]
					offset = end + 2
					end = offset + strings.Index(s[offset:], "</em>")
					word := s[offset:end]
					t += word
					offset = end + 5
					mu.Lock()
					if _, ok := kwf[key]; !ok {
						kwf[key] = map[string]map[string]map[string]int{
							word: {
								elevel: {
									id: 1,
								},
							},
						}
					} else if _, ok := kwf[key][word]; !ok {
						kwf[key][word] = map[string]map[string]int{
							elevel: {
								id: 1,
							},
						}
					} else if _, ok := kwf[key][word][elevel]; !ok {
						kwf[key][word][elevel] = map[string]int{
							id: 1,
						}
					} else if _, ok := kwf[key][word][elevel][id]; !ok {
						kwf[key][word][elevel][id] = 1
					} else {
						kwf[key][word][elevel][id] += 1
					}
					mu.Unlock()
				}

				t += s[offset:]

				pwc, err := NewPartialTextWithContext(id, bs, s, t)
				if err != nil {
					mu.Lock()
					*errs = append(*errs, err.Error())
					mu.Unlock()
					continue
				}

				mu.Lock()
				*pmatches = append(*pmatches, pwc)
				mu.Unlock()
			}
		}(&wg2, q2, kwf, &matches, &errs)
	}

	// put data into workers
	for _, hit := range res.Hits.Hits {
		q1 <- hit
	}
	close(q1)
	wg1.Wait()
	close(q2)
	wg2.Wait()

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Key < matches[j].Key
	})

	if len(errs) > 0 {
		return nil, fmt.Errorf(strings.Join(errs, "\n"))
	}

	return &TextSearchResult{
		Filters: struct {
			Keyword TextSearchKeywordFilter `json:"keyword"`
			Tag     []LabelValue            `json:"tag"`
		}{
			Keyword: kwf,
			Tag:     tag,
		},
		Bibl:    bibls,
		Matches: matches,
	}, nil
}
