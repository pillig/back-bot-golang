package model

import "errors"

//go:generate go run golang.org/x/tools/cmd/stringer@v0.22.0 -type=Rarity
type Rarity int

// Remember to rerun `go generate rarity.go` if you modify this block!
const (
	Rollback Rarity = 1
	Rare     Rarity = 10
	Uncommon Rarity = 90
	Common   Rarity = 400
)

var Rarities = [...]Rarity{Rollback, Rare, Uncommon, Common}

func LookUpRarity(name string) (Rarity, error) {
	for _, rarity := range Rarities {
		if rarity.String() == name {
			return rarity, nil
		}
	}

	return 0, errors.New("unknown rarity")
}

func MaxRarity() int {
	max := 0
	for _, rarity := range Rarities {
		rarity := int(rarity)
		if rarity > max {
			max = rarity
		}
	}
	return max
}
