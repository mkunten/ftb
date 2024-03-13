package main

import "fmt"

/* RegisterParam */
type RegisterParam struct {
	Type      string   `form:"type"`
	Bid       string   `form:"bid"`
	Cid       string   `form:"cid"`
	MecabType string   `form:"mecabType"`
	Iid       string   `form:"iid"`
	LocalPath []string `form:"localPath"`
	StartPos  []int    `form:"startPos"`
	EndPos    []int    `form:"endPos"`
}

/* OCRInfo */
type OCRInfo struct {
	LocalPath string `form:"localPath"`
	StartPos  int    `form:"startPos"`
	EndPos    int    `form:"endPos"`
}

func NewOCRInfos(rp *RegisterParam) ([]OCRInfo, error) {
	cnt := len(rp.LocalPath)
	if len(rp.StartPos) != cnt || len(rp.EndPos) != cnt {
		return nil, fmt.Errorf("lengths of localPath, startPos and endPos not same")
	}
	ois := make([]OCRInfo, cnt)

	for i, lp := range rp.LocalPath {
		ois[i].LocalPath = lp
		ois[i].StartPos = rp.StartPos[i]
		ois[i].EndPos = rp.EndPos[i]
	}

	return ois, nil
}
