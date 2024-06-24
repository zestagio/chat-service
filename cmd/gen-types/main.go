package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"
)

//go:embed templates/types.go.tpl
var templates embed.FS

var tpl = template.Must(template.ParseFS(templates, "templates/types.go.tpl"))

func main() {
	if len(os.Args) != 4 {
		log.Fatalf("invalid args count: %d", len(os.Args)-1)
	}

	pkg, types, out := os.Args[1], strings.Split(os.Args[2], ","), os.Args[3]
	if err := run(pkg, types, out); err != nil {
		log.Fatal(err)
	}

	p, _ := os.Getwd()
	fmt.Printf("%v generated\n", filepath.Join(p, out))
}

func run(pkg string, types []string, outFile string) error {
	tplContext := map[string]any{
		"pkg":     pkg,
		"types":   types,
		"argType": strings.Join(types, " | "),
	}

	buf := new(bytes.Buffer)

	if err := tpl.Execute(buf, tplContext); err != nil {
		return fmt.Errorf("cannot render template: %w", err)
	}

	if err := os.WriteFile(outFile, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("cannot write result: %w", err)
	}

	return nil
}
