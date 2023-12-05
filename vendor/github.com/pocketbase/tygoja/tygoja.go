package tygoja

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Tygoja is a generator for one or more input packages,
// responsible for linking them together if necessary.
type Tygoja struct {
	conf *Config

	parent           *Tygoja
	implicitPackages map[string][]string
	generatedTypes   map[string][]string
}

// New initializes a new Tygoja generator from the specified config.
func New(config Config) *Tygoja {
	config.InitDefaults()

	return &Tygoja{
		conf:             &config,
		implicitPackages: map[string][]string{},
		generatedTypes:   map[string][]string{},
	}
}

// Generate executes the generator and produces the related TS files.
func (g *Tygoja) Generate() (string, error) {
	// extract config packages
	configPackages := make([]string, 0, len(g.conf.Packages))
	for p, types := range g.conf.Packages {
		if len(types) == 0 {
			continue // no typings
		}
		configPackages = append(configPackages, p)
	}

	// load packages info
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedSyntax | packages.NeedFiles | packages.NeedDeps | packages.NeedImports | packages.NeedTypes,
	}, configPackages...)
	if err != nil {
		return "", err
	}

	var s strings.Builder

	// Heading
	if g.parent == nil {
		s.WriteString("// GENERATED CODE - DO NOT MODIFY BY HAND\n")

		// write base types
		// ---
		s.WriteString("type ")
		s.WriteString(BaseTypeDict)
		s.WriteString(" = { [key:string | number | symbol]: any; }\n")

		s.WriteString("type ")
		s.WriteString(BaseTypeAny)
		s.WriteString(" = any\n")
		// ---

		if g.conf.Heading != "" {
			s.WriteString(g.conf.Heading)
		}
	}

	for i, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return "", fmt.Errorf("%+v", pkg.Errors)
		}

		if len(pkg.GoFiles) == 0 {
			return "", fmt.Errorf("no input go files for package index %d", i)
		}

		if len(g.conf.Packages[pkg.ID]) == 0 {
			// ignore the package as it has no typings
			continue
		}

		pkgGen := &PackageGenerator{
			conf:           g.conf,
			pkg:            pkg,
			types:          g.conf.Packages[pkg.ID],
			generatedTypes: map[string]struct{}{},
			unknownTypes:   map[string]struct{}{},
			imports:        map[string][]string{},
		}

		code, err := pkgGen.Generate()
		if err != nil {
			return "", err
		}

		for t := range pkgGen.generatedTypes {
			g.generatedTypes[pkg.ID] = append(g.generatedTypes[pkg.ID], t)
		}

		for t := range pkgGen.unknownTypes {
			parts := strings.Split(t, ".")
			var tPkg string
			var tName string

			if len(parts) == 0 {
				continue
			}

			if len(parts) == 2 {
				// type from external package
				tPkg = parts[0]
				tName = parts[1]
			} else {
				// unexported type from the current package
				tName = parts[0]

				// already mapped for export
				if pkgGen.isTypeAllowed(tName) {
					continue
				}

				tPkg = packageNameFromPath(pkg.ID)

				// add to self import later
				pkgGen.imports[pkg.ID] = []string{tPkg}
			}

			for p, aliases := range pkgGen.imports {
				for _, alias := range aliases {
					if tName != "" && alias == tPkg && !g.isGenerated(p, tName) && !exists(g.implicitPackages[p], tName) {
						if g.implicitPackages[p] == nil {
							g.implicitPackages[p] = []string{}
						}
						g.implicitPackages[p] = append(g.implicitPackages[p], tName)
						break
					}
				}
			}
		}

		s.WriteString(code)
	}

	// recursively try to generate the found unknown types
	if len(g.implicitPackages) > 0 {
		subConfig := *g.conf
		subConfig.Heading = ""
		if (subConfig.TypeMappings) == nil {
			subConfig.TypeMappings = map[string]string{}
		}

		// extract the nonempty package definitions
		subConfig.Packages = make(map[string][]string, len(g.implicitPackages))
		for p, types := range g.implicitPackages {
			if len(types) == 0 {
				continue
			}
			subConfig.Packages[p] = types
		}

		subGenerator := New(subConfig)
		subGenerator.parent = g
		subResult, err := subGenerator.Generate()
		if err != nil {
			return "", err
		}

		s.WriteString(subResult)
	}

	return s.String(), nil
}

func (g *PackageGenerator) markAsGenerated(t string) {
	g.generatedTypes[t] = struct{}{}
}

func (g *Tygoja) isGenerated(pkg string, name string) bool {
	if g.parent != nil && g.parent.isGenerated(pkg, name) {
		return true
	}

	if len(g.generatedTypes[pkg]) == 0 {
		return false
	}

	for _, t := range g.generatedTypes[pkg] {
		if t == name {
			return true
		}
	}

	return false
}

// isTypeAllowed checks whether the provided type name is allowed by the generator "types".
func (g *PackageGenerator) isTypeAllowed(name string) bool {
	name = strings.TrimSpace(name)

	if name == "" {
		return false
	}

	for _, t := range g.types {
		if t == name || t == "*" {
			return true
		}
	}

	return false
}

var versionRegex = regexp.MustCompile(`^v\d+$`)

// packageNameFromPath extracts and normalizes the imported package identifier.
//
// For example:
//
// "github.com/labstack/echo/v5" -> "echo"
// "github.com/go-ozzo/ozzo-validation/v4" -> "ozzo_validation"
func packageNameFromPath(path string) string {
	name := filepath.Base(strings.Trim(path, `"' `))

	if versionRegex.MatchString(name) {
		name = filepath.Base(filepath.Dir(path))
	}

	return strings.ReplaceAll(name, "-", "_")
}

// exists checks if search exists in list.
func exists[T comparable](list []T, search T) bool {
	for _, v := range list {
		if v == search {
			return true
		}
	}

	return false
}
