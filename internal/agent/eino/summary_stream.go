package eino

import (
	"strings"
	"unicode/utf8"
)

// summaryProgress walks a possibly-truncated JSON string accumulating from a
// streaming tool call and returns however much of the "summary" field is
// already legible. The intent is to surface partial summary text to the user
// while the model is still generating the rest of the structured plan.
//
// Behavior:
//   - Returns the empty string until the parser has actually entered the
//     summary string literal.
//   - Decodes JSON escape sequences (\n, \", \uXXXX, etc.) so the caller sees
//     plain text suitable for an SSE delta.
//   - If the string ends mid-multibyte UTF-8 sequence, that incomplete tail is
//     trimmed so partial bytes never reach the SSE writer.
//   - Idempotent for callers that simply diff against the previously emitted
//     prefix.
type summaryProgress struct {
	complete bool
	value    strings.Builder
}

func extractSummarySoFar(raw string) string {
	prog := summaryProgress{}
	prog.parse(raw)
	return trimIncompleteUTF8(prog.value.String())
}

// parse scans `raw` looking for the value of the top-level "summary" key.
// We don't need a real JSON parser — we walk character-by-character and bail
// once we have all we can extract.
func (p *summaryProgress) parse(raw string) {
	idx := indexOfSummaryKey(raw)
	if idx < 0 {
		return
	}
	// Position cursor just after the opening quote of the value string.
	i := idx
	// Skip past the colon and any whitespace.
	for i < len(raw) && raw[i] != ':' {
		i++
	}
	if i >= len(raw) {
		return
	}
	i++
	for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t' || raw[i] == '\n' || raw[i] == '\r') {
		i++
	}
	if i >= len(raw) || raw[i] != '"' {
		return
	}
	i++
	// Now read characters until an unescaped closing quote or end of input.
	for i < len(raw) {
		ch := raw[i]
		if ch == '\\' {
			if i+1 >= len(raw) {
				return
			}
			esc := raw[i+1]
			switch esc {
			case '"':
				p.value.WriteByte('"')
				i += 2
			case '\\':
				p.value.WriteByte('\\')
				i += 2
			case '/':
				p.value.WriteByte('/')
				i += 2
			case 'n':
				p.value.WriteByte('\n')
				i += 2
			case 'r':
				p.value.WriteByte('\r')
				i += 2
			case 't':
				p.value.WriteByte('\t')
				i += 2
			case 'b':
				p.value.WriteByte('\b')
				i += 2
			case 'f':
				p.value.WriteByte('\f')
				i += 2
			case 'u':
				if i+6 > len(raw) {
					return
				}
				r := decodeUnicodeEscape(raw[i+2 : i+6])
				if r < 0 {
					return
				}
				p.value.WriteRune(rune(r))
				i += 6
			default:
				return
			}
			continue
		}
		if ch == '"' {
			p.complete = true
			return
		}
		p.value.WriteByte(ch)
		i++
	}
}

// indexOfSummaryKey returns the index of the byte right after the closing
// quote of the literal `"summary"` key. -1 if not found yet. We intentionally
// only match the top-level key — inside item.reason or warnings strings the
// substring `summary` won't appear.
func indexOfSummaryKey(raw string) int {
	// Look for "summary" preceded by either `{` or `,` and followed by `:`.
	const key = `"summary"`
	for offset := 0; offset < len(raw); {
		i := strings.Index(raw[offset:], key)
		if i < 0 {
			return -1
		}
		i += offset
		// Look back for the previous non-whitespace byte; must be `{` or `,`.
		j := i - 1
		for j >= 0 && (raw[j] == ' ' || raw[j] == '\t' || raw[j] == '\n' || raw[j] == '\r') {
			j--
		}
		if j >= 0 && (raw[j] == '{' || raw[j] == ',') {
			return i + len(key)
		}
		offset = i + len(key)
	}
	return -1
}

func decodeUnicodeEscape(hex string) int {
	if len(hex) != 4 {
		return -1
	}
	var out int
	for k := 0; k < 4; k++ {
		c := hex[k]
		var v int
		switch {
		case c >= '0' && c <= '9':
			v = int(c - '0')
		case c >= 'a' && c <= 'f':
			v = int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			v = int(c-'A') + 10
		default:
			return -1
		}
		out = out<<4 | v
	}
	return out
}

// trimIncompleteUTF8 drops any trailing bytes that don't form a complete UTF-8
// rune. The streaming JSON can split a multibyte CJK character across two
// chunks; emitting the partial bytes would print "?" on the client.
func trimIncompleteUTF8(s string) string {
	for len(s) > 0 {
		r, size := utf8.DecodeLastRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	return s
}
