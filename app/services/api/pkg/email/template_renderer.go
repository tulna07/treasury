package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"regexp"
	"strings"
)

//go:embed templates/*.html
var templateFS embed.FS

// TemplateRenderer renders email templates from embedded files.
type TemplateRenderer struct {
	templates map[string]*template.Template
}

// NewTemplateRenderer loads all embedded templates and returns a renderer.
func NewTemplateRenderer() (*TemplateRenderer, error) {
	r := &TemplateRenderer{
		templates: make(map[string]*template.Template),
	}

	baseContent, err := templateFS.ReadFile("templates/base.html")
	if err != nil {
		return nil, fmt.Errorf("read base template: %w", err)
	}

	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return nil, fmt.Errorf("read templates dir: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "base.html" || entry.IsDir() {
			continue
		}

		content, err := templateFS.ReadFile("templates/" + name)
		if err != nil {
			return nil, fmt.Errorf("read template %s: %w", name, err)
		}

		tmpl, err := template.New("base").Parse(string(baseContent))
		if err != nil {
			return nil, fmt.Errorf("parse base for %s: %w", name, err)
		}

		tmpl, err = tmpl.Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", name, err)
		}

		// Store without .html extension
		templateName := strings.TrimSuffix(name, ".html")
		r.templates[templateName] = tmpl
	}

	return r, nil
}

// Render renders a named template with the given data, returning both HTML and plain text.
func (r *TemplateRenderer) Render(templateName string, data interface{}) (html string, text string, err error) {
	tmpl, ok := r.templates[templateName]
	if !ok {
		return "", "", fmt.Errorf("template %q not found", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
		return "", "", fmt.Errorf("execute template %q: %w", templateName, err)
	}

	htmlStr := buf.String()
	textStr := stripHTML(htmlStr)
	return htmlStr, textStr, nil
}

// stripHTML removes HTML tags and normalizes whitespace for plain-text fallback.
func stripHTML(s string) string {
	// Remove style content
	reStyle := regexp.MustCompile(`<style[^>]*>[\s\S]*?</style>`)
	s = reStyle.ReplaceAllString(s, "")

	// Replace <br>, <p>, <div>, <tr> with newlines
	reBlock := regexp.MustCompile(`<(?:br|/p|/div|/tr|/h[1-6])[^>]*>`)
	s = reBlock.ReplaceAllString(s, "\n")

	// Remove remaining tags
	reTag := regexp.MustCompile(`<[^>]+>`)
	s = reTag.ReplaceAllString(s, "")

	// Decode common HTML entities
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")

	// Collapse multiple blank lines
	reBlank := regexp.MustCompile(`\n{3,}`)
	s = reBlank.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
}
