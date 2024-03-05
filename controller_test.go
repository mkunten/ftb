package main

// import (
// 	"encoding/json"
// 	"os"
// 	"path/filepath"
// 	"testing"
//
// 	cmp "github.com/google/go-cmp/cmp"
// )
//
// const srcDir = "testdata/src"
// const expectDir = "testdata/expected"
//
// // test
// if _, err := es.Indices.Analyze().CharFilter("icu_normalizer").Tokenizer("kuromoji_tokenizer").Text("日本の文化①1１ｶﾞガがかﾞ").Do(context.Background()); err != nil {
// 	panic(err)
// }

// // test
// bt, err := createESDataFromDir("/home/mkunten/Downloads/dev/ocr/0001-000101")
// if err != nil {
// 	panic(err)
// }
// bi := &BookInfo{
// 	Bid:      "100000001",
// 	Cid:      "34000",
// 	Iid:      "0001-000101",
// 	StartPos: 1,
// 	EndPos:   23,
// }
// err = es.IndexBookData(bt, bi)
// if err != nil {
// 	panic(err)
// }

// // set BookInfo
// bi := &BookInfo{
// 	Bid:      "100000001",
// 	Cid:      "3045041",
// 	Iid:      "0001-000101",
// 	StartPos: 1,
// 	EndPos:   23,
// }
//
// func TestESIndexBookText *testing.T) {
//   t.Run("ESIndexBookText", func(t *testing.T) {
//     testESIndexBookText(t,
//   })
// }
