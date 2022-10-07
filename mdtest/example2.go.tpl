package main

import (
	"github.com/alexflint/go-arg/v2"
	{{if contains .Code "fmt."}}"fmt"{{end}}
    {{if contains .Code "strings."}}"strings"{{end}}
)

{{.Code}}
