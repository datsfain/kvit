package report

import _ "embed"

//go:embed template.html
var TemplateHTML string

//go:embed style.css
var StyleCSS string

//go:embed app.js
var AppJS string
