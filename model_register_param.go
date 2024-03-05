package main

/* RegisterParam */
type RegisterParam struct {
	Type      string `form:"type"`
	Bid       string `form:"bid"`
	Cid       string `form:"cid"`
	MecabType string `form:"mecabType"`
	Iid       string `form:"iid"`
	LocalPath string `form:"localPath"`
	StartPos  int    `form:"startPos"`
	EndPos    int    `form:"endPos"`
}
