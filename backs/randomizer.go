package backs

import (
	"back-bot/backs/model"
	"fmt"
	"io/fs"
	"math/rand"
)

type BackMapping map[model.Rarity][]model.Back

// GetBacks gets all the file paths assigned to their rarities
func GetBacks(backfs fs.FS) (BackMapping, error) {

	backMap := BackMapping{}
	tiers, err := fs.ReadDir(backfs, ".")
	if err != nil {
		fmt.Printf("what happened to my backs? %x\n", err)
		return nil, err
	}
	for _, tier := range tiers {
		var backs []model.Back
		rarityString := tier.Name()

		rarity, err := model.LookUpRarity(rarityString)
		if err != nil {
			return nil, fmt.Errorf("unknown rarity encountered as member of back_repo: %w", err)
		}

		fs.WalkDir(backfs, rarityString, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				fmt.Printf("err while walking back_repo subdirectory. path: %s err: %s\n", path, err)
			}

			// skip the rarity dir itself
			if path == rarityString {
				return nil
			}

			back, _ := model.GetBack(path)
			backs = append(backs, back)

			return nil
		})

		backMap[rarity] = backs
	}
	return backMap, nil
}

func chooseBack(bl BackMapping) (model.Back, error) {

	max := model.MaxRarity()
	roll := rand.Intn(max)
	fmt.Printf("rolled a %d\n", roll)
	for _, r := range model.Rarities {
		if roll <= int(r) {
			back, err := pickFromBackList(bl, r)
			if err != nil {
				return model.Back{}, err
			}
			return back, nil
		}
	}
	return model.Back{}, fmt.Errorf("no back was able to be chosen")

}

func pickFromBackList(bl BackMapping, rarity model.Rarity) (model.Back, error) {
	val, ok := bl[rarity]
	if ok {
		index := rand.Intn(len(val))
		return val[index], nil
	} else {
		return model.Back{}, fmt.Errorf("no rarity of %s found in rarity list", rarity)
	}
}
