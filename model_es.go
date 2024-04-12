package main

import (
	"context"
	"fmt"
	"math"
	"os"

	"github.com/dgraph-io/ristretto"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/get"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/indices/create"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/dynamicmapping"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/highlightertagsschema"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/highlightertype"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/indexoptions"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/termvectoroption"
)

type ES struct {
	Client    *elasticsearch.TypedClient
	Highlight *types.Highlight
	Cache     *ristretto.Cache
}

func (es *ES) Init() error {
	esCfg := elasticsearch.Config{
		Addresses: cfg.ESAddresses,
		Logger: &elastictransport.ColorLogger{
			Output:             os.Stdout,
			EnableRequestBody:  true,
			EnableResponseBody: false,
		},
	}

	c, err := elasticsearch.NewTypedClient(esCfg)
	if err != nil || c == nil {
		return err
	}

	es.Client = c

	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7, // 10M
		MaxCost:     cfg.CacheSize,
		BufferItems: 64,
	})
	if err != nil {
		return err
	}

	es.Cache = cache

	return nil
}

func (es *ES) InitIndex(isForce bool) error {
	exists, err := es.Client.Indices.Exists(cfg.IndexName).
		IsSuccess(context.Background())
	if err != nil {
		return err
	}
	if exists {
		if isForce {
			_, err := es.Client.Indices.Delete(cfg.IndexName).Do(context.Background())
			if err != nil {
				return err
			}
			fmt.Printf("index deleted: %s\n", cfg.IndexName)
		} else {
			return err
		}
	}

	// labelValueProp
	labelValueProp := types.NewNestedProperty()
	labelValueProp.Properties = map[string]types.Property{
		"label": types.NewKeywordProperty(),
		"value": types.NewKeywordProperty(),
	}

	// metadataProp
	metadataProp := types.NewNestedProperty()
	metadataProp.Properties = map[string]types.Property{
		"bid":         types.NewKeywordProperty(),
		"cid":         types.NewKeywordProperty(),
		"elevel":      types.NewKeywordProperty(),
		"tags":        types.NewKeywordProperty(),
		"metadata":    labelValueProp,
		"label":       types.NewKeywordProperty(),
		"attribution": types.NewKeywordProperty(),
		"license":     types.NewKeywordProperty(),
	}

	// textProp
	// icu => bigram
	var (
		tokenizer string = "my_bigram_tokenizer"
		analyzer  string = "my_icu_ngram_analyzer"
	)

	customTokenizer := types.NewNGramTokenizer()
	customTokenizer.MinGram = 2
	customTokenizer.MaxGram = 2

	customAnalyzer := types.NewCustomAnalyzer()
	customAnalyzer.CharFilter = []string{"icu_normalizer"}
	customAnalyzer.Tokenizer = tokenizer

	customIndexOptions := &indexoptions.IndexOptions{}
	customIndexOptions.Name = "positions"

	customTermVector := &termvectoroption.TermVectorOption{}
	customTermVector.Name = "with_positions_offsets"

	textProp := types.NewTextProperty()
	textProp.Analyzer = &analyzer
	textProp.IndexOptions = customIndexOptions
	textProp.TermVector = customTermVector

	s := &types.IndexSettings{
		Analysis: &types.IndexSettingsAnalysis{
			Analyzer: map[string]types.Analyzer{
				analyzer: customAnalyzer,
			},
			Tokenizer: map[string]types.Tokenizer{
				tokenizer: customTokenizer,
			},
		},
		Mapping: &types.MappingLimitSettings{
			NestedObjects: &types.MappingLimitSettingsNestedObjects{
				Limit: Int2Pt(1e+6),
			},
		},
	}

	// bbsProp
	bbsProp := types.NewNestedProperty()
	bbsProp.Properties = map[string]types.Property{
		"x": types.NewIntegerNumberProperty(),
		"y": types.NewIntegerNumberProperty(),
		"w": types.NewIntegerNumberProperty(),
		"h": types.NewIntegerNumberProperty(),
	}

	// see type BookText
	m := &types.TypeMapping{
		Dynamic: &dynamicmapping.Strict,
		Properties: map[string]types.Property{
			"metadata":  metadataProp,
			"text":      textProp,
			"pbs":       types.NewIntegerNumberProperty(),
			"lbs":       types.NewIntegerNumberProperty(),
			"bbs":       bbsProp,
			"images":    types.NewKeywordProperty(),
			"mecabType": types.NewKeywordProperty(),
			"mecabed":   types.NewKeywordProperty(),
		},
	}
	_, err = es.Client.Indices.Create(cfg.IndexName).
		Request(&create.Request{
			Settings: s,
			Mappings: m,
		}).Do(context.Background())
	if err != nil {
		return err
	}

	fmt.Printf("index created: %s\n", cfg.IndexName)

	return nil
}

func (es *ES) IndexBookData(bt *BookText) error {
	res, err := es.Client.Index(cfg.IndexName).
		Id(bt.GetId_()).
		Document(bt).
		Do(context.Background())
	if err != nil {
		return err
	}

	fmt.Printf("document added: %v\n", res)
	return nil
}

func (es *ES) Get(id string) (*get.Response, error) {
	data, err := es.Client.
		Get(cfg.IndexName, id).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (es *ES) CountRecord() (*search.Response, error) {
	filters := map[string]*types.Query{}
	for _, elevel := range ELevelValues() {
		key := elevel.String()
		filters[key] = &types.Query{
			Nested: &types.NestedQuery{
				Path: "metadata",
				Query: &types.Query{
					Match: map[string]types.MatchQuery{
						"metadata.elevel": {
							Query: key,
						},
					},
				},
			},
		}
	}
	data, err := es.Client.Search().
		Index(cfg.IndexName).
		Size(0).
		Aggregations(map[string]types.Aggregations{
			"recordCount": {
				Filters: &types.FiltersAggregation{
					Filters: &filters,
				},
			},
		}).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (es *ES) SearchText(sp *TextSearchParam) (*search.Response, error) {
	data, err := es.Client.Search().
		Index(cfg.IndexName).
		Query(sp.GetESQuery()).
		Highlight(&types.Highlight{
			Fields: map[string]types.HighlightField{
				"text": {
					Type:                  &highlightertype.Fvh,
					BoundaryScannerLocale: Str2Pt("ja-JP"),
					FragmentSize:          Int2Pt(50),
					NumberOfFragments:     Int2Pt(math.MaxInt16),
					NoMatchSize:           Int2Pt(0),
				},
			},
			TagsSchema: &highlightertagsschema.Styled,
		}).
		Sort(&types.SortOptions{
			SortOptions: map[string]types.FieldSort{
				"metadata.bid": {
					Nested: &types.NestedSortValue{
						Path: "metadata",
					},
					Order: &sortorder.Asc,
				},
			},
		}).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	return data, nil
}
