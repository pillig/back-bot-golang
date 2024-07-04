package backs

import "errors"

//go:generate go run golang.org/x/tools/cmd/stringer@v0.22.0 -type=rarity
type rarity int

// Remember to rerun `go generate rarity.go` if you modify this block!
const (
	Rollback rarity = 1
	Rare     rarity = 10
	Uncommon rarity = 90
	Common   rarity = 400
)

var rarities = [...]rarity{Rollback, Rare, Uncommon, Common}

func lookUpRarity(name string) (rarity, error) {
	for _, rarity := range rarities {
		if rarity.String() == name {
			return rarity, nil
		}
	}

	return 0, errors.New("unknown rarity")
}

func maxRarity() int {
	max := 0
	for _, rarity := range rarities {
		rarity := int(rarity)
		if rarity > max {
			max = rarity
		}
	}
	return max
}
