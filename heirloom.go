package heirloom

import (
	"bytes"
	"errors"
	"html/template"
	"strings"
)

// Heirloom is a collection of templates.
type Heirloom struct {
	templates map[string]*template.Template
	funcs     template.FuncMap
}

// New initialize a new Heirloom.
func New() *Heirloom {
	return &Heirloom{
		templates: make(map[string]*template.Template),
		funcs:     template.FuncMap{},
	}
}

// FuncMap is copied here from html/template for convenience.
type FuncMap map[string]interface{}

// Funcs will add the given map to all subsequently parsed templates.
func (h *Heirloom) Funcs(f FuncMap) {
	h.funcs = template.FuncMap(f)
}

// Parse add the given template with the given name to the collection. It
// can be used later in inheritance chain.
func (h *Heirloom) Parse(name, src string) error {
	// We adding fake methods here, so the parsing of the templates
	// themselves is not going to fail.
	t, err := template.New(name).Funcs(h.funcs).Funcs(template.FuncMap{
		"yield": func() template.HTML {
			return ""
		},
		"inherits": func(_ string) string {
			return ""
		},
	}).Parse(src)
	if err != nil {
		return err
	}

	// When parsing is done, add the template to the collection.
	h.templates[name] = t
	return nil
}

// Render execute the given template (and the inheritance chain) with the given
// data.
func (h *Heirloom) Render(name string, data interface{}) (string, error) {
	// The "initial" parent is the requested template. It starts by inheriting an
	// empty buffer.
	var parent = name
	var buffer = bytes.Buffer{}

	// While there is a parent in the inheritance chain….
	for parent != "" {
		// Look for it in the collection.
		t, found := h.templates[parent]
		if !found {
			return "", errors.New("unknown template " + name)
		}

		// Clone it so we can operate on it safely in case of concurrency.
		t, _ = t.Clone()

		// Save the buffer's content.
		yield := buffer.String()

		// Reset the parent and the buffer.
		parent = ""
		buffer.Reset()

		// Erase the fake methods with ones that actually use this scope.
		t.Funcs(template.FuncMap{
			// inherits will add the parent to the chain.
			"inherits": func(p string) string {
				parent = p
				return ""
			},
			// yield will print the content of the buffer.
			"yield": func() template.HTML {
				return template.HTML(strings.TrimSpace(yield))
			},
		})

		// Execute the template.
		err := t.Execute(&buffer, data)
		if err != nil {
			return "", err
		}
	}

	// Return the content of the buffer.
	return buffer.String(), nil
}
