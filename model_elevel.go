package main

import "strings"

/* ELevel */
type ELevel int

//go:generate go run github.com/alvaroloes/enumer -type=ELevel -json

const (
	OCR ELevel = iota
	PROOF_READ
)

func ELevelValueBinder(sp *TextSearchParam) func(values []string) []error {
	return func(values []string) []error {
		els := []ELevel{}
		for _, value := range values {
			for _, v := range strings.Split(value, ",") {
				el, err := ELevelString(v)
				if err != nil {
					return []error{err}
				}
				els = append(els, el)
			}
		}
		sp.ELevels = els
		return nil
	}
}
