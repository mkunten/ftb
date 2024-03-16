package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	cmp "github.com/google/go-cmp/cmp"
)

func TestOcrV2Result2BookText(t *testing.T) {
	t.Parallel()

	t.Run("OcrV2Result2BookText", func(t *testing.T) {
		testOcrV2Result2BookText(t, "0001-000101", 1, 23, false)
		testOcrV2Result2BookText(t, "0001-000102", 1, 101, false)
	})
}

func TestOcrV2DetailResult2BookText(t *testing.T) {
	t.Parallel()

	t.Run("OcrV2DetailResult2BookText", func(t *testing.T) {
		testOcrV2Result2BookText(t, "0001-000101", 1, 23, true)
	})
}

func testOcrV2Result2BookText(t *testing.T, dir string, startPos, endPos int, isDetail bool) {
	t.Helper()

	s := "ndlocrv2"
	if isDetail {
		s += "detail"
	}
	bt, err := NdlOcrV22BookText([]OCRInfo{{
		LocalPath: filepath.Join(srcDir, s, dir),
		StartPos:  startPos,
		EndPos:    endPos,
	}}, isDetail)
	if err != nil {
		t.Fatal(err)
	}

	// // update testdata
	// bytes, _ := json.Marshal(bt)
	// if err := os.WriteFile(
	// 	filepath.Join(expectDir, s, dir)+".json",
	// 	bytes,
	// 	0666,
	// ); err != nil {
	// 	t.Fatal(err)
	// }

	expect, err := getOcrV2ExpectBookText(dir, isDetail)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(&bt, &expect); diff != "" {
		t.Errorf("NdlOcrV22BookText(%s) mismatch (-want +got):\n%s", dir, diff)
	}
}

func TestOcrV2BookTextGetText(t *testing.T) {
	t.Parallel()

	t.Run("BookTextGetText", func(t *testing.T) {
		testOcrV2BookTextGetText(t, "0001-000101", 18, 3, "鯰の為のその日は暮ぬ秋の風", false)
		testOcrV2BookTextGetText(t, "0001-000102", 1, 4, "尾陽蓬ニ左檀木堂主人荷今子集を", false)
	})
}

func testOcrV2BookTextGetText(t *testing.T, dir string, page, line int, expect string, isDetail bool) {
	t.Helper()

	bt, err := getOcrV2ExpectBookText(dir, isDetail)
	if err != nil {
		t.Fatal(err)
	}

	got := bt.GetText(page, line)

	if got != expect {
		t.Errorf("(*BookText).GetText(%d, %d) => \"%s\", want \"%s\"", page, line, got, expect)
	}
}

func getOcrV2ExpectBookText(dir string, isDetail bool) (*BookText, error) {
	s := "ndlocrv2"
	if isDetail {
		s += "detail"
	}
	raw, err := os.ReadFile(filepath.Join(expectDir, s, dir) + ".json")
	if err != nil {
		return nil, err
	}
	var expect *BookText
	if err := json.Unmarshal(raw, &expect); err != nil {
		return nil, err
	}
	return expect, nil
}
