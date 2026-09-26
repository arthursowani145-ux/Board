// canon.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

// CanonicalizeJSON is the ONLY entry point signing code should use.
// It takes raw JSON bytes, decodes them with UseNumber, and returns
// the RFC 8785 (JCS) canonical form.
//
// Why this shape: canonicalization bugs in signed-JSON systems almost
// always come from a caller decoding JSON one way and a verifier decoding
// it another. By making the public API take bytes and do the decode
// itself, there is no way to hand it a map produced by a caller's own
// json.Unmarshal, and therefore no way to accidentally introduce a
// float64 (which different languages format differently) into the
// canonical stream.
func CanonicalizeJSON(data []byte) ([]byte, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("canon: input is not valid UTF-8")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v interface{}
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("canon: decode: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("canon: trailing data after JSON value")
	}
	return canonicalize(v)
}

// canonicalize is unexported on purpose. Callers go through
// CanonicalizeJSON. Tests may call it directly.
func canonicalize(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := writeCanon(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeCanon(buf *bytes.Buffer, v interface{}) error {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case string:
		return writeCanonString(buf, x)

	case json.Number:
		// Only integers. Anything with '.', 'e', or 'E' is a float and is
		// rejected — different languages format floats differently and we
		// refuse to guess. Our schema uses int64 + strings only.
		if _, err := strconv.ParseInt(x.String(), 10, 64); err != nil {
			return fmt.Errorf("canon: non-integer number %q not supported", x.String())
		}
		buf.WriteString(x.String())

	case int:
		buf.WriteString(strconv.FormatInt(int64(x), 10))
	case int64:
		buf.WriteString(strconv.FormatInt(x, 10))

	case float64:
		// Unreachable if all signing paths go through CanonicalizeJSON.
		// If this fires, a caller decoded JSON without UseNumber. Fail
		// loudly rather than silently guess a canonical form.
		return fmt.Errorf("canon: float64 %v reached canonicalizer; caller must use CanonicalizeJSON", x)

	case []interface{}:
		buf.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanon(buf, e); err != nil {
				return err
			}
		}
		buf.WriteByte(']')

	case map[string]interface{}:
		keys := make([]string, 0, len(x))
		for k := range x {
			if !utf8.ValidString(k) {
				return fmt.Errorf("canon: invalid UTF-8 in object key %q", k)
			}
			keys = append(keys, k)
		}
		// JCS: sort by UTF-16 code units. Go's default string sort is by
		// UTF-8 bytes, which agrees for ASCII but NOT for non-ASCII.
		// Implement the real comparator so we stay correct for any key.
		sort.Slice(keys, func(i, j int) bool {
			return utf16Less(keys[i], keys[j])
		})
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonString(buf, k); err != nil {
				return err
			}
			buf.WriteByte(':')
			if err := writeCanon(buf, x[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')

	default:
		return fmt.Errorf("canon: unsupported type %T", v)
	}
	return nil
}

// writeCanonString emits a JSON string with minimal escaping per JCS,
// and rejects invalid UTF-8 outright rather than silently substituting
// U+FFFD for bad bytes. Silent substitution would mean two
// implementations (this one vs. a Python/JS verifier) could canonicalize
// the same malformed input differently — a signature failure invisible
// from the data itself.
func writeCanonString(buf *bytes.Buffer, s string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("canon: invalid UTF-8 in string %q", s)
	}
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(buf, `\u%04x`, r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
	return nil
}

// utf16Less compares two strings by UTF-16 code unit order (JCS rule).
func utf16Less(a, b string) bool {
	au := utf16.Encode([]rune(a))
	bu := utf16.Encode([]rune(b))
	for i := 0; i < len(au) && i < len(bu); i++ {
		if au[i] != bu[i] {
			return au[i] < bu[i]
		}
	}
	return len(au) < len(bu)
}
