package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
)

/* BulkRegisterParam */
type BulkRegisterParam struct {
	Type           string `form:"type"`
	AbortOnError   bool   `form:"abortOnError"`
	ListFileHeader *multipart.FileHeader
}

/* BulkRegisterCsv */
type BulkRegisterCsv []BulkRegisterCsvItem

/* BulkRegisterCsvItem */
type BulkRegisterCsvItem struct {
	Bid   string
	Cid   string
	Iid   string
	Vol   string
	Start string
	End   string
}

const (
	CsvBid int = iota
	CsvCid
	CsvIid
	CsvVol
	CsvStartPos
	CsvEndPos
	CsvMecabType
)

var mu sync.Mutex

type BulkResult struct {
	Message []string `json:"message"`
	Error   []string `json:"error"`
}

func (br *BulkResult) AddMsgf(format string, a ...any) {
	mu.Lock()
	br.Message = append(br.Message, fmt.Sprintf(format, a...))
	mu.Unlock()
}

func (br *BulkResult) AddErrf(format string, a ...any) {
	mu.Lock()
	br.Error = append(br.Error, fmt.Sprintf(format, a...))
	mu.Unlock()
}

// IndexData
func (brp *BulkRegisterParam) BulkIndexData() (*BulkResult, error) {
	msgs := &BulkResult{}

	//ioutil.ReadDir(cfg.BulkSourceDir)
	if filepath.Ext(brp.ListFileHeader.Filename) != ".csv" {
		return nil, fmt.Errorf("%s: must be '.csv'", brp.ListFileHeader.Filename)
	}

	f, err := brp.ListFileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("%s: cannot open file", brp.ListFileHeader.Filename)
	}
	defer f.Close()

	r := csv.NewReader(f)
	// header: bid,cid,iid,vol,start,end
	if row, err := r.Read(); err != nil || row[0] != "bid" {
		return nil, fmt.Errorf("first line must be header")
	}

	var wg1 sync.WaitGroup
	var wg2 sync.WaitGroup
	q1 := make(chan RegisterParam, cfg.BulkWorkerNum)
	q2 := make(chan *BookText, cfg.BulkESUnitNum)

	// prepare workers
	for i := 0; i < cfg.BulkWorkerNum; i++ {
		// csv row => BookText
		wg1.Add(1)
		go func(wg1 *sync.WaitGroup, q1 chan RegisterParam, q2 chan *BookText, msgs *BulkResult) {
			defer wg1.Done()
			for {
				rp, ok := <-q1
				if !ok {
					break
				}

				bt, err := NewBookText(&rp)
				if err != nil {
					msgs.AddErrf("new %s: %s", rp.Bid, err)
					continue
				}
				if err := bt.FetchKokushoMetadata(); err != nil {
					msgs.AddErrf("new %s: %s", rp.Bid, err)
					continue
				}
				q2 <- bt
			}
		}(&wg1, q1, q2, msgs)
	}

	// prepare receiver
	// BookText => ES
	wg2.Add(1)
	go BulkIndexBookDataWorker(&wg2, q2, msgs)

	rp := RegisterParam{}

	// put csv row into workers
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}

		if row[CsvBid] != rp.Bid {
			if rp.Bid != "" {
				q1 <- rp
			}

			path := row[CsvIid]
			if cfg.IsBulkSubdir {
				if i := strings.Index(row[CsvIid], "-"); i != -1 {
					path = filepath.Join(row[CsvIid][:i], path)
				}
			}
			path = filepath.Join(cfg.BulkSourceDir, path)

			sp, _ := strconv.Atoi(row[CsvStartPos])
			ep, _ := strconv.Atoi(row[CsvEndPos])
			mt := ""
			if len(row) == 7 {
				mt = row[CsvMecabType]
			}
			rp.MecabType = mt
			rp.LocalPath = []string{path}
			rp.StartPos = []int{sp}
			rp.EndPos = []int{ep}

			rp = RegisterParam{
				Type:      brp.Type,
				Bid:       row[CsvBid],
				Cid:       row[CsvCid],
				MecabType: mt,
				Iid:       row[CsvIid],
				LocalPath: []string{path},
				StartPos:  []int{sp},
				EndPos:    []int{ep},
			}
		} else {
			path := row[CsvIid]
			if cfg.IsBulkSubdir {
				if i := strings.Index(row[CsvIid], "-"); i != -1 {
					path = filepath.Join(row[CsvIid][:i], path)
				}
			}
			path = filepath.Join(cfg.BulkSourceDir, path)

			sp, _ := strconv.Atoi(row[CsvStartPos])
			ep, _ := strconv.Atoi(row[CsvEndPos])

			rp.LocalPath = append(rp.LocalPath, path)
			rp.StartPos = append(rp.StartPos, sp)
			rp.EndPos = append(rp.EndPos, ep)
		}
	}

	if rp.Bid != "" {
		q1 <- rp
	}
	close(q1)
	wg1.Wait()
	close(q2)
	wg2.Wait()

	return msgs, nil
}

func BulkIndexBookDataWorker(wg2 *sync.WaitGroup, q2 chan *BookText, msgs *BulkResult) {
	defer wg2.Done()

	escfg := elasticsearch.Config{
		Addresses: cfg.ESAddresses,
		Logger: &elastictransport.ColorLogger{
			Output:             os.Stdout,
			EnableRequestBody:  false,
			EnableResponseBody: true,
		},
	}

	c, err := elasticsearch.NewClient(escfg)
	if err != nil {
		msgs.AddErrf("elasticsearch.NewClient: %s", err)
		return
	}

	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         cfg.IndexName,
		Client:        c,
		NumWorkers:    cfg.BulkWorkerNum,
		FlushBytes:    5e+6,
		FlushInterval: 30 * time.Second,
	})
	if err != nil {
		msgs.AddErrf("esutil.NewBulkIndexer: %s", err)
		return
	}

	start := time.Now().UTC()
	var countSuccessful uint64 = 0

	for {
		bt, ok := <-q2
		if !ok {
			break
		}

		data, err := json.Marshal(bt)
		if err != nil {
			msgs.AddErrf("cannot encode BookText:%s: %s", bt.Bid, err)
			break
		}

		if err := bi.Add(
			context.Background(),
			esutil.BulkIndexerItem{
				Action:     "index",
				DocumentID: bt.GetId_(),
				Body:       bytes.NewReader(data),
				OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
					atomic.AddUint64(&countSuccessful, 1)
				},
				OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
					if err != nil {
						msgs.AddErrf("ERROR: %s", err)
					} else {
						msgs.AddErrf("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
					}
				},
			},
		); err != nil {
			msgs.AddErrf("BulkIndexer add: %s", err)
			return
		}
	}

	if err := bi.Close(context.Background()); err != nil {
		msgs.AddErrf("BulkIndexer close: %s", err)
		return
	}

	biStats := bi.Stats()
	dur := time.Since(start)

	// Report
	if biStats.NumFailed > 0 {
		msgs.AddMsgf(
			"Indexed [%s] documents with [%s] errors in %s (%s docs/sec)",
			humanize.Comma(int64(biStats.NumFlushed)),
			humanize.Comma(int64(biStats.NumFailed)),
			dur.Truncate(time.Millisecond),
			humanize.Comma(int64(1000.0/float64(dur/time.Millisecond)*float64(biStats.NumFlushed))),
		)
		return
	}

	msgs.AddMsgf(
		"Sucessfuly indexed [%s] documents in %s (%s docs/sec)",
		humanize.Comma(int64(biStats.NumFlushed)),
		dur.Truncate(time.Millisecond),
		humanize.Comma(int64(1000.0/float64(dur/time.Millisecond)*float64(biStats.NumFlushed))),
	)

	fmt.Println(msgs.Message)

	return
}
