package features

type TemplateRenderer struct {
	Render func(tpl string, data map[string]any) (string, error)
}
