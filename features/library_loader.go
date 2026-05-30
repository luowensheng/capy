package features

import "github.com/olivierdevelops/capy/domain"

type LibraryLoader struct {
	Load func(path string) (domain.Library, error)
}
