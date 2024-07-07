package backs

import "io/fs"

// TODO: expand to handle all backfs interactions?
// currently this is just a shared cache for BackMapping
type BackProvider interface {
	Backs() BackMapping
}

type backProvider struct {
	backfs  fs.FS
	mapping BackMapping
}

func NewBackProvider(backfs fs.FS) *backProvider {
	provider := new(backProvider)
	provider.backfs = backfs

	mapping, err := GetBacks(backfs)
	if err != nil {
		panic(err)
	}

	provider.mapping = mapping

	return provider
}

func (b *backProvider) Backs() BackMapping {
	return b.mapping
}
