package tygoja

import (
	"go/ast"
	"strings"
)

func (g *PackageGenerator) writeCommentGroup(s *strings.Builder, f *ast.CommentGroup, depth int) {
	if f == nil {
		return
	}

	docLines := strings.Split(f.Text(), "\n")

	g.writeIndent(s, depth)
	s.WriteString("/**\n")

	lastLineIdx := len(docLines) - 1

	var isCodeBlock bool

	emptySB := new(strings.Builder)

	for i, c := range docLines {
		isEndLine := i == lastLineIdx
		isEmpty := len(strings.TrimSpace(c)) == 0
		isIndented := strings.HasPrefix(c, "\t") || strings.HasPrefix(c, "  ")

		// end code block
		if isCodeBlock && (isEndLine || (!isIndented && !isEmpty)) {
			g.writeIndent(s, depth)
			s.WriteString(" * ```\n")
			isCodeBlock = false
		}

		// accumulate empty comment lines
		// (this is done to properly enclose code blocks with new lines)
		if isEmpty {
			g.writeIndent(emptySB, depth)
			emptySB.WriteString(" * \n")
		} else {
			// write all empty lines
			s.WriteString(emptySB.String())
			emptySB.Reset()
		}

		// start code block
		if isIndented && !isCodeBlock && !isEndLine {
			g.writeIndent(s, depth)
			s.WriteString(" * ```\n")
			isCodeBlock = true
		}

		// write comment line
		if !isEmpty {
			g.writeIndent(s, depth)
			s.WriteString(" * ")
			c = strings.ReplaceAll(c, "*/", "*\\/") // An edge case: a // comment can contain */
			s.WriteString(c)
			s.WriteByte('\n')
		}
	}

	g.writeIndent(s, depth)
	s.WriteString(" */\n")
}
