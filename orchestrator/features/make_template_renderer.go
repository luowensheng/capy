package orchfeatures

import (
	"github.com/luowensheng/capy/features"
	"github.com/luowensheng/capy/infra"
)

func MakeTemplateRenderer(eng infra.TemplateEngine) features.TemplateRenderer {
	return features.TemplateRenderer{
		Render: eng.Render,
	}
}
