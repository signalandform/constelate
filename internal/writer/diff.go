package writer

import (
	"fmt"
	"strings"
)

// Diff renders a unified-style line diff of Old vs New for one write op.
// Files here are small config files, so a plain LCS table is fine.
func Diff(op Op) []string {
	if op.Kind != OpWrite {
		return []string{describe(op)}
	}
	a := splitLines(string(op.Old))
	b := splitLines(string(op.New))
	if op.Old == nil {
		a = nil
	}
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var out []string
	out = append(out, fmt.Sprintf("--- %s", op.Path), fmt.Sprintf("+++ %s", op.Path))
	i, j := 0, 0
	ctx := 0 // unchanged lines emitted since the last change, to trim context
	const keep = 2
	var pending []string
	flush := func() {
		if len(pending) > 2*keep+1 {
			out = append(out, pending[:keep]...)
			out = append(out, fmt.Sprintf("@@ %d lines @@", len(pending)-2*keep))
			out = append(out, pending[len(pending)-keep:]...)
		} else {
			out = append(out, pending...)
		}
		pending = nil
	}
	for i < n || j < m {
		switch {
		case i < n && j < m && a[i] == b[j]:
			pending = append(pending, " "+a[i])
			i++
			j++
			ctx++
		case j < m && (i >= n || lcs[i][j+1] >= lcs[i+1][j]):
			flush()
			out = append(out, "+"+b[j])
			j++
			ctx = 0
		default:
			flush()
			out = append(out, "-"+a[i])
			i++
			ctx = 0
		}
	}
	flush()
	return out
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}
