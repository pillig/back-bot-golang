package backs

import (
	"fmt"
	"io/fs"
	"math/rand"
	"path"
	"strings"
)

type back struct {
	path string
}

func (b back) Filename() string {
	return path.Base(b.path)
}

func (b back) Backname() string {
	return strings.Split(b.Filename(), ".")[0]
}

func (b back) Rarity() rarity {
	r, _ := lookUpRarity(path.Base(path.Dir(b.path)))
	return r
}

type BackMapping map[rarity][]back

// GetBacks gets all the file paths assigned to their rarities
func GetBacks(backfs fs.FS) (BackMapping, error) {

	backMap := BackMapping{}
	tiers, err := fs.ReadDir(backfs, ".")
	if err != nil {
		fmt.Printf("what happened to my backs? %x\n", err)
		return nil, err
	}
	for _, tier := range tiers {
		var backs []back
		rarityString := tier.Name()

		rarity, err := lookUpRarity(rarityString)
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

			backs = append(backs, back{path})

			return nil
		})

		backMap[rarity] = backs
	}
	return backMap, nil
}

func chooseBack(bl BackMapping) (string, error) {

	max := maxRarity()
	roll := rand.Intn(max)
	fmt.Printf("rolled a %d\n", roll)
	for _, r := range rarities {
		if roll <= int(r) {
			back, err := pickFromBackList(bl, r)
			if err != nil {
				return "", err
			}
			return back, nil
		}
	}
	return "", fmt.Errorf("no back was able to be chosen")

}

func pickFromBackList(bl BackMapping, rarity rarity) (string, error) {
	val, ok := bl[rarity]
	if ok {
		index := rand.Intn(len(val))
		return val[index].path, nil
	} else {
		return "", fmt.Errorf("no rarity of %s found in rarity list", rarity)
	}
}
