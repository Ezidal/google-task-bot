package telegram

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Все unique из markup.Data в keyboards.go должны обрабатываться в onCallback.
func TestAllInlineUniquesHandled(t *testing.T) {
	keyboardUniques := extractDataUniques(t, "keyboards.go")
	handlerUniques := extractSwitchCases(t, "callbacks.go")

	var missing []string
	for u := range keyboardUniques {
		if _, ok := handlerUniques[u]; !ok {
			missing = append(missing, u)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("кнопки без обработчика: %v", missing)
	}
}

func extractDataUniques(t *testing.T, file string) map[string]struct{} {
	src, err := os.ReadFile(filepath.Join("internal", "telegram", file))
	if err != nil {
		// test runs from package dir
		src, err = os.ReadFile(file)
	}
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`markup\.Data\([^,]+,\s*"([^"]+)"`)
	out := make(map[string]struct{})
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		out[m[1]] = struct{}{}
	}
	return out
}

func extractSwitchCases(t *testing.T, file string) map[string]struct{} {
	path := file
	if _, err := os.Stat(path); err != nil {
		path = filepath.Join("internal", "telegram", file)
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string]struct{})
	ast.Inspect(f, func(n ast.Node) bool {
		sw, ok := n.(*ast.SwitchStmt)
		if !ok {
			return true
		}
		// switch unique := ...
		for _, clause := range sw.Body.List {
			cc, ok := clause.(*ast.CaseClause)
			if !ok {
				continue
			}
			for _, expr := range cc.List {
				if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					out[strings.Trim(lit.Value, `"`)] = struct{}{}
				}
			}
		}
		return true
	})
	return out
}
