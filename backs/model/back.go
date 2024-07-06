package model

import (
	"errors"
	"path"
	"strings"
)

type Back struct {
	path string
}

func GetBack(path string) (Back, error) {
	// TODO: more validation of path?
	if path == "" {
		return Back{}, errors.New("Back path cannot be empty")
	}
	return Back{path}, nil
}

func (b Back) Path() string {
	return b.path
}

func (b Back) Filename() string {
	return path.Base(b.path)
}

func (b Back) Backname() string {
	return strings.Split(b.Filename(), ".")[0]
}

func (b Back) Rarity() Rarity {
	r, _ := LookUpRarity(path.Base(path.Dir(b.path)))
	return r
}
