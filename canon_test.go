package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCanonicalizeBasic(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty object", `{}`, `{}`},
		{"empty array", `[]`, `[]`},
		{"empty string", `""`, `""`},
		{"null", `null`, `null`},
		{"true", `true`, `true`},
		{"false", `false`, `false`},
		{"integer", `1`, `1`},
		{"zero", `0`, `0`},
		{"negative", `-42`, `-42`},
		{"simple string", `"hello"`, `"hello"`},
		{"key order a b", `{"b":1,"a":2}`, `{"a":2,"b":1}`},
		{"key order z y x", `{"z":1,"y":2,"x":3}`, `{"x":3,"y":2,"z":1}`},
		{"nested", `{"a":{"b":{"c":1}}}`, `{"a":{"b":{"c":1}}}`},
		{"array in object", `{"a":[1,2,3]}`, `{"a":[1,2,3]}`},
		{"object in array", `[{"b":1,"a":2}]`, `[{"a":2,"b":1}]`},
		{"whitespace stripped", "{\n  \"a\" : 1\n}", `{"a":1}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := CanonicalizeJSON([]byte(c.in))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestCanonicalizeStrings(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"quote", `"a\"b"`, `"a\"b"`},
		{"backslash", `"a\\b"`, `"a\\b"`},
		{"newline", `"a\nb"`, `"a\nb"`},
		{"tab", `"a\tb"`, `"a\tb"`},
		{"carriage return", `"a\rb"`, `"a\rb"`},
		{"backspace", `"a\bb"`, `"a\bb"`},
		{"form feed", `"a\fb"`, `"a\fb"`},
		{"control char low", `"\u0001"`, `"\u0001"`},
		{"control char high", `"\u001f"`, `"\u001f"`},
		{"unicode not escaped", `"\u00e9"`, "\"\u00e9\""},
		{"emoji not escaped", `"\ud83d\ude00"`, "\"\U0001F600\""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := CanonicalizeJSON([]byte(c.in))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestCanonicalizeRejections(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
	}{
		{"float", []byte(`1.5`)},
		{"float with exponent", []byte(`1e10`)},
		{"float negative", []byte(`-0.5`)},
		{"invalid UTF-8 in string", []byte{'"', 0xff, 0xfe, '"'}},
		{"invalid UTF-8 in key", []byte{'{', '"', 0xff, '"', ':', '1', '}'}},
		{"bad JSON", []byte(`{`)},
		{"trailing garbage", []byte(`1 2`)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := CanonicalizeJSON(c.in)
			if err == nil {
				t.Fatalf("expected error, got none")
			}
		})
	}
}

func TestCanonicalizeUTF16KeyOrder(t *testing.T) {
	// U+10000 (encoded as surrogate pair D800 DC00) sorts BEFORE U+E000 in
	// UTF-16 order because 0xD800 < 0xE000. But in UTF-8 byte order,
	// U+10000 (F0 90 80 80) sorts AFTER U+E000 (EE 80 80).
	// JCS mandates UTF-16 order, so U+10000 must come first.
	in := []byte(`{"\ue000":1,"\ud800\udc00":2}`)
	got, err := CanonicalizeJSON(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// U+10000 (\ud800\udc00) first, then U+E000 (\ue000)
	want := "\"\U00010000\":2,\"\uE000\":1"
	if !bytes.Contains(got, []byte(want)) {
		t.Fatalf("UTF-16 order violated: got %q", got)
	}
}

func TestFloat64Tripwire(t *testing.T) {
	// Bypass CanonicalizeJSON: hand canonicalize a decoded value with a
	// float64. It must refuse, not silently format.
	var v interface{}
	dec := json.NewDecoder(bytes.NewReader([]byte(`{"a":1}`)))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		t.Fatal(err)
	}
	// manually replace with a float64 to simulate a caller who decoded
	// without UseNumber
	v.(map[string]interface{})["a"] = float64(1)
	_, err := canonicalize(v)
	if err == nil {
		t.Fatal("float64 reached canonicalizer without error")
	}
}
