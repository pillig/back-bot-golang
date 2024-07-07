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

// RarityLootValues represents how many "rarity points" a given back
// has for its rarity, derived from the Rarity values themselves, inversely.
var RarityLootValues = make(map[Rarity]int)

// init RarityLootValue using Tom's algorithm: https://github.com/pillig/back-bot/blob/master/LootTools/loottracker.py#L45-L49
func init() {
	const lootMultiplier = 300

	// don't included Rollback rarity
	nonRollbackRarities := Rarities[1:]

	var sum int
	for _, rarity := range nonRollbackRarities {
		sum += int(rarity)
	}

	weight := sum * lootMultiplier / len(nonRollbackRarities)

	for _, rarity := range nonRollbackRarities {
		RarityLootValues[rarity] = weight / int(rarity)
	}
}

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
