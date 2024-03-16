package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	cmp "github.com/google/go-cmp/cmp"
)

func TestNdlOcrV1Result2BookText(t *testing.T) {
	t.Parallel()

	t.Run("NdlOcrV1Result2BookText", func(t *testing.T) {
		testNdlOcrV12BookText(t, "200004700_1_3045000_YA0-082-001-035-015", 1, 78)
		testNdlOcrV12BookText(t, "200032715_1_3045055_029-0038", 1, 74)
	})
}

func testNdlOcrV12BookText(t *testing.T, dir string, startPos, endPos int) {
	t.Helper()

	bt, err := NdlOcrV12BookText([]OCRInfo{{
		LocalPath: filepath.Join(srcDir, "ndlocrv1", dir),
		StartPos:  startPos,
		EndPos:    endPos,
	}})
	if err != nil {
		t.Fatal(err)
	}

	// // update testdata
	// bytes, _ := json.Marshal(bt)
	// if err := os.WriteFile(
	// 	filepath.Join(expectDir, "ndlocrv1", dir)+".json",
	// 	bytes,
	// 	0666,
	// ); err != nil {
	// 	t.Fatal(err)
	// }

	expect, err := getExpectNdlOcrV1BookText(dir)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(&bt, &expect); diff != "" {
		t.Errorf("OcrResult2BookText(%s) mismatch (-want +got):\n%s", dir, diff)
	}
}

func TestNdlOcrV1GetText(t *testing.T) {
	t.Parallel()

	t.Run("NdlOcrV1GetText", func(t *testing.T) {
		testNdlOcrV1GetText(t, "200004700_1_3045000_YA0-082-001-035-015", 18, 3, "横雲のひま見えゆくに。すさきにたてる松の木たち")
		testNdlOcrV1GetText(t, "200004700_1_3045000_YA0-082-001-035-015", 1, 1808, "扶桑拾葉集巻第十三終")
	})
}

func testNdlOcrV1GetText(t *testing.T, dir string, page, line int, expect string) {
	t.Helper()

	bt, err := getExpectNdlOcrV1BookText(dir)
	if err != nil {
		t.Fatal(err)
	}

	got := bt.GetText(page, line)

	if got != expect {
		t.Errorf("(*BookText).GetText(%d, %d) => \"%s\", want \"%s\"", page, line, got, expect)
	}
}

func getExpectNdlOcrV1BookText(dir string) (*BookText, error) {
	raw, err := os.ReadFile(filepath.Join(expectDir, "ndlocrv1", dir) + ".json")
	if err != nil {
		return nil, err
	}
	var expect *BookText
	if err := json.Unmarshal(raw, &expect); err != nil {
		return nil, err
	}
	return expect, nil
}
