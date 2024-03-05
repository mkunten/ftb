package main

import (
	"archive/zip"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/shogo82148/go-mecab"
)

func Int2Pt(i int) *int {
	return &i
}

func Str2Pt(s string) *string {
	return &s
}

func MecabFilter(mecabType, text string) ([]string, error) {
	mecabTypes := []string{
		"jodai",
		"chuko",
		"waka",
		"chusei-bungo",
		"chusei-kougo",
		"kinsei-bungo",
		"kinsei-edo",
		"kinsei-kamigata",
		"kindai-bungo",
		"qkana",
		"novel",
	}

	if mecabType != "" && !slices.Contains(mecabTypes, mecabType) {
		return nil, fmt.Errorf("unexpected mecab type: \"%s\"", mecabType)
	}

	tagger, err := mecab.New(map[string]string{
		"dicdir": filepath.Join(cfg.MecabDir, "unidic-"+mecabType),
	})
	if err != nil {
		return nil, fmt.Errorf("mecab not initialized: %s", err)
	}
	defer tagger.Destroy()

	tagger.Parse("")

	node, err := tagger.ParseToNode(text)
	if err != nil {
		return nil, fmt.Errorf("mecab parse error: %s", err)
	}

	keys := []string{}
	for ; !node.IsZero(); node = node.Next() {
		f := strings.Split(node.Feature(), ",")
		if len(f) < 27 {
			keys = append(keys, node.Surface())
		} else {
			keys = append(keys, f[0]+":"+f[1]+":"+f[2]+":"+f[3]+":"+f[10])
		}
	}

	return keys, nil
}

func unzipUploaded(file *multipart.FileHeader, destdir string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	r, err := zip.NewReader(src, file.Size)
	if err != nil {
		return err
	}

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(destdir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
