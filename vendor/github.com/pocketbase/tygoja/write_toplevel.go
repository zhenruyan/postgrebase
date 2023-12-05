package tygoja

import (
	"fmt"
	"go/ast"
	"strings"
)

type groupContext struct {
	isGroupedDeclaration bool
	doc                  *ast.CommentGroup
	groupValue           string
	groupType            string
	iotaValue            int
	iotaOffset           int
}

// Writing of function declarations, which are expressions like
// "func Count() int"
// or
// "func (s *Counter) total() int"
func (g *PackageGenerator) writeFuncDecl(s *strings.Builder, decl *ast.FuncDecl, depth int) {
	if decl.Name == nil || len(decl.Name.Name) == 0 || decl.Name.Name[0] < 'A' || decl.Name.Name[0] > 'Z' {
		return // unexported function/method
	}

	originalMethodName := decl.Name.Name
	methodName := originalMethodName
	if g.conf.MethodNameFormatter != nil {
		methodName = g.conf.MethodNameFormatter(methodName)
	}

	if decl.Recv == nil {
		if !g.conf.WithPackageFunctions {
			return // skip package level functions
		}

		if !g.isTypeAllowed(originalMethodName) {
			return
		} else {
			g.markAsGenerated(originalMethodName)
		}

		// write package level function in the format "type FuncName = { (args): result }"
		if decl.Doc != nil {
			g.writeCommentGroup(s, decl.Doc, depth)
		}

		g.writeStartModifier(s, depth)
		s.WriteString("interface ")
		s.WriteString(methodName)

		if decl.Type.TypeParams != nil {
			g.writeTypeParamsFields(s, decl.Type.TypeParams.List)
		}

		s.WriteString(" {")
		g.writeType(s, decl.Type, depth+1, false)
		s.WriteString("}\n")
	} else if len(decl.Recv.List) == 1 {
		// write struct method as new interface method
		// (note that TS will "merge" the definitions of multiple interfaces with the same name)

		// treat pointer and value receivers the same
		recvType := decl.Recv.List[0].Type
		if p, isPointer := recvType.(*ast.StarExpr); isPointer {
			recvType = p.X
		}

		var recvName string
		switch recv := recvType.(type) {
		case *ast.Ident:
			recvName = recv.Name
		case *ast.IndexExpr:
			if v, ok := recv.X.(*ast.Ident); ok {
				recvName = v.Name
			}
		}

		if !g.isTypeAllowed(recvName) {
			return
		} else {
			g.markAsGenerated(recvName)
		}

		g.writeStartModifier(s, depth)
		s.WriteString("interface ")

		g.writeType(s, recvType, depth, false)

		s.WriteString(" {\n")
		if decl.Doc != nil {
			g.writeCommentGroup(s, decl.Doc, depth+1)
		}
		g.writeIndent(s, depth+1)
		s.WriteString(methodName)
		g.writeType(s, decl.Type, depth+1, false)
		s.WriteString("\n")
		g.writeIndent(s, depth)
		s.WriteString("}\n")
	}
}

func (g *PackageGenerator) writeGroupDecl(s *strings.Builder, decl *ast.GenDecl, depth int) {
	// We need a bit of state to handle syntax like
	// const (
	//   X SomeType = iota
	//   _
	//   Y
	//   Foo string = "Foo"
	//   _
	//   AlsoFoo
	// )
	group := &groupContext{
		isGroupedDeclaration: len(decl.Specs) > 1,
		doc:                  decl.Doc,
		groupType:            "",
		groupValue:           "",
		iotaValue:            -1,
	}

	for _, spec := range decl.Specs {
		g.writeSpec(s, spec, group, depth)
	}
}

func (g *PackageGenerator) writeSpec(s *strings.Builder, spec ast.Spec, group *groupContext, depth int) {
	// e.g. "type Foo struct {}" or "type Bar = string"
	ts, ok := spec.(*ast.TypeSpec)
	if ok {
		g.writeTypeSpec(s, ts, group, depth)
	}

	// e.g. "const Foo = 123"
	vs, ok := spec.(*ast.ValueSpec)
	if ok && g.conf.WithConstants {
		g.writeValueSpec(s, vs, group, depth)
	}
}

// Writing of type specs, which are expressions like
// "type X struct { ... }"
// or
// "type Bar = string"
func (g *PackageGenerator) writeTypeSpec(s *strings.Builder, ts *ast.TypeSpec, group *groupContext, depth int) {
	var typeName string
	if ts.Name != nil {
		typeName = ts.Name.Name
	}

	if !g.isTypeAllowed(typeName) {
		return
	} else {
		g.markAsGenerated(typeName)
	}

	if ts.Doc != nil { // The spec has its own comment, which overrules the grouped comment.
		g.writeCommentGroup(s, ts.Doc, depth)
	} else if group.isGroupedDeclaration {
		g.writeCommentGroup(s, group.doc, depth)
	}

	switch v := ts.Type.(type) {
	case *ast.StructType:
		// eg. "type X struct { ... }"

		var extendTypeName string

		// convert embeded structs to "extends SUB_TYPE" declaration
		//
		// note: we don't use "extends A, B, C" form but intersecion subtype
		// with all embeded structs to avoid methods merge conflicts
		// eg. bufio.ReadWriter has different Writer.Read() and Reader.Read()
		if v.Fields != nil {
			var embeds []*ast.Field
			for _, f := range v.Fields.List {
				if len(f.Names) == 0 || f.Names[0].Name == "" {
					embeds = append(embeds, f)
				}
			}

			if len(embeds) > 0 {
				extendTypeName = "_sub" + PseudorandomString(5)

				genericArgs := map[string]struct{}{}
				identSB := new(strings.Builder)
				embedsSB := new(strings.Builder)
				for i, f := range embeds {
					if i > 0 {
						embedsSB.WriteString("&")
					}

					typ := f.Type
					if p, isPointer := typ.(*ast.StarExpr); isPointer {
						typ = p.X
					}

					identSB.Reset()
					g.writeType(identSB, typ, depth, true)
					ident := identSB.String()

					if idx := strings.Index(ident, "<"); idx > 1 { // has at least 2 characters for <>
						genericArgs[ident[idx+1:len(ident)-1]] = struct{}{}
					}

					embedsSB.WriteString(ident)
				}

				if len(genericArgs) > 0 {
					args := make([]string, 0, len(genericArgs))
					for g := range genericArgs {
						args = append(args, g)
					}
					extendTypeName = extendTypeName + "<" + strings.Join(args, ",") + ">"
				}

				g.writeIndent(s, depth)
				s.WriteString("type ")
				s.WriteString(extendTypeName)
				s.WriteString(" = ")
				s.WriteString(embedsSB.String())
				s.WriteString("\n")
			}
		}

		g.writeStartModifier(s, depth)
		s.WriteString("interface ")
		s.WriteString(typeName)

		if ts.TypeParams != nil {
			g.writeTypeParamsFields(s, ts.TypeParams.List)
		}

		if extendTypeName != "" {
			s.WriteString(" extends ")
			s.WriteString(extendTypeName)
		}

		s.WriteString(" {\n")
		g.writeStructFields(s, v.Fields.List, depth)
		g.writeIndent(s, depth)
		s.WriteString("}")
	case *ast.InterfaceType:
		// eg. "type X interface { ... }"

		g.writeStartModifier(s, depth)
		s.WriteString("interface ")
		s.WriteString(typeName)

		if ts.TypeParams != nil {
			g.writeTypeParamsFields(s, ts.TypeParams.List)
		}

		s.WriteString(" {\n")
		g.writeInterfaceFields(s, v.Methods.List, depth)
		g.writeIndent(s, depth)
		s.WriteString("}")
	case *ast.FuncType:
		// eg. "type Handler func() any"

		g.writeStartModifier(s, depth)
		s.WriteString("interface ")
		s.WriteString(typeName)

		if ts.TypeParams != nil {
			g.writeTypeParamsFields(s, ts.TypeParams.List)
		}

		s.WriteString(" {")
		g.writeFuncType(s, v, depth, false)
		g.writeIndent(s, depth)
		s.WriteString("}")
	default:
		// other Go type declarations like "type JsonArray []any"
		// (note: we don't use "type X = Y", but "interface X extends Y"  syntax to allow later defining methods to the X type)

		var baseType string
		subSB := new(strings.Builder)
		g.writeType(subSB, ts.Type, depth, true)
		switch baseType = subSB.String(); baseType {
		// primitives can't be extended so we use their Object equivivalents
		case "number", "string", "boolean":
			baseType = strings.Title(baseType)
		case "any":
			baseType = BaseTypeAny
		}

		g.writeStartModifier(s, depth)
		s.WriteString("interface ")
		s.WriteString(typeName)

		if ts.TypeParams != nil {
			g.writeTypeParamsFields(s, ts.TypeParams.List)
		}

		s.WriteString(" extends ")

		s.WriteString(baseType)

		s.WriteString("{}")
	}

	if ts.Comment != nil {
		s.WriteString(" // " + ts.Comment.Text())
	} else {
		s.WriteString("\n")
	}
}

// Writing of value specs, which are exported const expressions like
// const SomeValue = 3
func (g *PackageGenerator) writeValueSpec(s *strings.Builder, vs *ast.ValueSpec, group *groupContext, depth int) {
	for i, name := range vs.Names {
		group.iotaValue = group.iotaValue + 1
		if name.Name == "_" {
			continue
		}

		if !g.isTypeAllowed(name.Name) {
			continue
		} else {
			g.markAsGenerated(name.Name)
		}

		if vs.Doc != nil { // The spec has its own comment, which overrules the grouped comment.
			g.writeCommentGroup(s, vs.Doc, depth)
		} else if group.isGroupedDeclaration {
			g.writeCommentGroup(s, group.doc, depth)
		}

		hasExplicitValue := len(vs.Values) > i
		if hasExplicitValue {
			group.groupType = ""
		}

		g.writeStartModifier(s, depth)
		s.WriteString("const ")
		s.WriteString(name.Name)
		if vs.Type != nil {
			s.WriteString(": ")

			tempSB := &strings.Builder{}
			g.writeType(tempSB, vs.Type, depth, true)
			typeString := tempSB.String()

			s.WriteString(typeString)
			group.groupType = typeString
		} else if group.groupType != "" && !hasExplicitValue {
			s.WriteString(": ")
			s.WriteString(group.groupType)
		}

		s.WriteString(" = ")

		if hasExplicitValue {
			val := vs.Values[i]
			tempSB := &strings.Builder{}
			g.writeType(tempSB, val, depth, true)

			valueString := tempSB.String()
			if isProbablyIotaType(valueString) {
				iotaV, err := basicIotaOffsetValueParse(valueString)
				if err != nil {
					// fallback
					group.groupValue = valueString
				} else {
					group.iotaOffset = iotaV
					group.groupValue = "iota"
					valueString = fmt.Sprint(group.iotaValue + group.iotaOffset)
				}
			} else {
				group.groupValue = valueString
			}
			s.WriteString(valueString)
		} else { // We must use the previous value or +1 in case of iota
			valueString := group.groupValue
			if group.groupValue == "iota" {
				valueString = fmt.Sprint(group.iotaValue + group.iotaOffset)
			}
			s.WriteString(valueString)
		}

		if vs.Comment != nil {
			s.WriteString(" // " + vs.Comment.Text())
		} else {
			s.WriteByte('\n')
		}
	}
}
