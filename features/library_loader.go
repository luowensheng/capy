package features

import "github.com/luowensheng/capy/domain"

type LibraryLoader struct {
	Load func(path string) (domain.Library, error)
}
