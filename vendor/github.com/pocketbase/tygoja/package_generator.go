package tygoja

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/packages"
)

// PackageGenerator is responsible for generating the code for a single input package.
type PackageGenerator struct {
	conf  *Config
	pkg   *packages.Package
	types []string

	generatedTypes map[string]struct{}
	unknownTypes   map[string]struct{}
	imports        map[string][]string // path -> []names/aliases
}

// Generate generates the typings for a single package.
func (g *PackageGenerator) Generate() (string, error) {
	s := new(strings.Builder)

	namespace := packageNameFromPath(g.pkg.ID)

	s.WriteString("\n")
	for _, f := range g.pkg.Syntax {
		if f.Doc == nil || len(f.Doc.List) == 0 {
			continue
		}
		g.writeCommentGroup(s, f.Doc, 0)
	}
	g.writeStartModifier(s, 0)
	s.WriteString("namespace ")
	s.WriteString(namespace)
	s.WriteString(" {\n")

	// register the aliased imports within the package namespace
	// (see https://www.typescriptlang.org/docs/handbook/namespaces.html#aliases)
	loadedAliases := map[string]struct{}{}
	for _, file := range g.pkg.Syntax {
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"' `)

			pgkName := packageNameFromPath(path)
			alias := pgkName

			if imp.Name != nil && imp.Name.Name != "" && imp.Name.Name != "_" {
				alias = imp.Name.Name

				if _, ok := loadedAliases[alias]; ok {
					continue // already registered
				}

				loadedAliases[alias] = struct{}{}

				g.writeIndent(s, 1)
				s.WriteString("// @ts-ignore\n")
				g.writeIndent(s, 1)
				s.WriteString("import ")
				s.WriteString(alias)
				s.WriteString(" = ")
				s.WriteString(pgkName)
				s.WriteString("\n")
			}

			// register the import to export its package later
			if !exists(g.imports[path], alias) {
				if g.imports[path] == nil {
					g.imports[path] = []string{}
				}
				g.imports[path] = append(g.imports[path], alias)
			}
		}

		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl: // FuncDecl can be package level function or struct method
				g.writeFuncDecl(s, x, 1)
				return false
			case *ast.GenDecl: // GenDecl can be an import, type, var, or const expression
				if x.Tok == token.VAR || x.Tok == token.IMPORT {
					return false // ignore variables and import statements for now
				}

				g.writeGroupDecl(s, x, 1)
				return false
			}

			return true
		})
	}

	s.WriteString("}\n")

	return s.String(), nil
}
