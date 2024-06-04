package backs

import (
	"fmt"
	"math/rand"
	"os"
)

type backRarity struct {
	Type  string
	Value int
}

func getRarities() []backRarity {
	return []backRarity{
		{Type: "Rollback", Value: 1},
		{Type: "Rare", Value: 10},
		{Type: "Uncommon", Value: 90},
		{Type: "Common", Value: 400},
	}
}

func maxRarities() int {
	max := 0
	for _, rarity := range getRarities() {
		if rarity.Value > max {
			max = rarity.Value
		}
	}
	return max
}

type BackMapping map[string][]string

// GetBacks gets all the file paths assigned to their rarities
func GetBacks() (*BackMapping, error) {

	backMap := BackMapping{}
	tiers, err := os.ReadDir("./back_repo")
	if err != nil {
		fmt.Printf("what happened to my backs? %x\n", err)
		return &BackMapping{}, err
	}
	for _, tier := range tiers {
		backs := []string{}
		rarity := tier.Name()
		rarityDir := fmt.Sprintf("./back_repo/%s", rarity)
		backFiles, err := os.ReadDir(rarityDir)
		if err != nil {
			fmt.Printf("something is wrong with my backs: %x\n", err)
			return &BackMapping{}, err
		}
		for _, file := range backFiles {
			backs = append(backs, fmt.Sprintf("%s/%s", rarityDir, file.Name()))
		}
		backMap[rarity] = backs

	}
	return &backMap, nil
}

func chooseBack(bl *BackMapping) (string, error) {

	max := maxRarities()
	roll := rand.Intn(max)
	fmt.Printf("rolled a %d\n", roll)
	for _, r := range getRarities() {
		if roll <= r.Value {
			back, err := pickFromBackList(bl, r.Type)
			if err != nil {
				return "", err
			}
			return back, nil
		}
	}
	return "", fmt.Errorf("no back was able to be chosen")

}

func pickFromBackList(bl *BackMapping, rarity string) (string, error) {
	list := *bl
	val, ok := list[rarity]
	if ok {
		index := rand.Intn(len(val))
		return val[index], nil
	} else {
		return "", fmt.Errorf("no rarity of %s found in rarity list", rarity)
	}
}
