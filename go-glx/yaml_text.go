// Copyright 2025 Oracynth, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package glx

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// ErrQuotedTextCollision means a quoting token also occurred in the archive's
// own text, so the encoding could not be restored unambiguously.
var ErrQuotedTextCollision = errors.New("quoted-text token collided with archive text")

// needsQuotedText reports whether go-yaml would write text as a literal block
// scalar that it then cannot read back: go-yaml can choose an unmarked block
// scalar for a multiline string whose first non-blank character is a tab, and
// reject that same tab while reading the file.
func needsQuotedText(text string) bool {
	return strings.HasPrefix(strings.TrimLeft(text, "\n\r "), "\t") && strings.ContainsAny(text, "\n\r")
}

// marshalYAML is yaml.Marshal, except that every string value needsQuotedText
// matches is written double-quoted, wherever it occurs: notes, properties,
// titles. (Map keys are entity and property IDs and are left alone.)
//
// go-yaml offers no per-scalar style for a struct's string field, and
// yaml.Node.Encode goes through the same failing text, so marshalYAML encodes
// a copy of the value in which each such string is a unique plain-scalar
// token, then swaps every token for the double-quoted original. Only the
// containers on the path to a replaced string are copied; the caller's value
// is never modified. The output uses the requested indentation in both paths.
func marshalYAML(value any, indent string) ([]byte, error) {
	quoter := textQuoter{}
	rewritten, changed := quoter.rewrite(reflect.ValueOf(value))
	if !changed {
		return encodeYAML(value, indent)
	}
	encoded, err := encodeYAML(rewritten.Interface(), indent)
	if err != nil {
		return nil, err
	}

	return quoter.restore(encoded)
}

// defaultYAMLIndent is the indentation glx writes when SerializerOptions.Indent
// is empty or invalid: two spaces, as in the specification and the examples (#1427).
const defaultYAMLIndent = 2

// encodeYAML encodes value with an indentation string made up of spaces.
func encodeYAML(value any, indent string) ([]byte, error) {
	width := len(indent)
	if width == 0 || strings.Trim(indent, " ") != "" {
		width = defaultYAMLIndent
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(width)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type textQuoter struct {
	nonce     string
	originals []string
}

func (q *textQuoter) token(index int) string {
	return "glxquoted" + q.nonce + "x" + strconv.Itoa(index) + "x"
}

// rewrite returns value with each string needsQuotedText matches replaced by
// a token, copying only what contains one, and reports whether it did.
func (q *textQuoter) rewrite(value reflect.Value) (reflect.Value, bool) {
	switch value.Kind() {
	case reflect.String:
		return q.rewriteString(value)
	case reflect.Pointer, reflect.Interface:
		return q.rewriteIndirect(value)
	case reflect.Struct:
		return q.rewriteStruct(value)
	case reflect.Map:
		return q.rewriteMap(value)
	case reflect.Slice, reflect.Array:
		return q.rewriteSequence(value)
	default:
		return value, false
	}
}

func (q *textQuoter) rewriteString(value reflect.Value) (reflect.Value, bool) {
	text := value.String()
	if !needsQuotedText(text) || !utf8.ValidString(text) {
		return value, false
	}
	if q.nonce == "" {
		var random [8]byte
		_, _ = rand.Read(random[:])
		q.nonce = hex.EncodeToString(random[:])
	}
	replaced := reflect.New(value.Type()).Elem()
	replaced.SetString(q.token(len(q.originals)))
	q.originals = append(q.originals, text)

	return replaced, true
}

func (q *textQuoter) rewriteIndirect(value reflect.Value) (reflect.Value, bool) {
	if value.IsNil() {
		return value, false
	}
	inner, changed := q.rewrite(value.Elem())
	if !changed {
		return value, false
	}
	if value.Kind() == reflect.Pointer {
		pointer := reflect.New(value.Type().Elem())
		pointer.Elem().Set(inner)

		return pointer, true
	}
	wrapped := reflect.New(value.Type()).Elem()
	wrapped.Set(inner)

	return wrapped, true
}

func (q *textQuoter) rewriteStruct(value reflect.Value) (reflect.Value, bool) {
	var copied reflect.Value
	for index := range value.NumField() {
		if !value.Type().Field(index).IsExported() {
			continue
		}
		field, changed := q.rewrite(value.Field(index))
		if !changed {
			continue
		}
		if !copied.IsValid() {
			copied = reflect.New(value.Type()).Elem()
			copied.Set(value)
		}
		copied.Field(index).Set(field)
	}

	return orOriginal(copied, value)
}

func (q *textQuoter) rewriteMap(value reflect.Value) (reflect.Value, bool) {
	var copied reflect.Value
	for iterator := value.MapRange(); iterator.Next(); {
		entry, changed := q.rewrite(iterator.Value())
		if !changed {
			continue
		}
		if !copied.IsValid() {
			copied = reflect.MakeMapWithSize(value.Type(), value.Len())
			for original := value.MapRange(); original.Next(); {
				copied.SetMapIndex(original.Key(), original.Value())
			}
		}
		copied.SetMapIndex(iterator.Key(), entry)
	}

	return orOriginal(copied, value)
}

func (q *textQuoter) rewriteSequence(value reflect.Value) (reflect.Value, bool) {
	var copied reflect.Value
	for index := range value.Len() {
		element, changed := q.rewrite(value.Index(index))
		if !changed {
			continue
		}
		if !copied.IsValid() {
			copied = reflect.New(value.Type()).Elem()
			if value.Kind() == reflect.Slice {
				copied.Set(reflect.MakeSlice(value.Type(), value.Len(), value.Len()))
			}
			reflect.Copy(copied, value)
		}
		copied.Index(index).Set(element)
	}

	return orOriginal(copied, value)
}

func orOriginal(copied, original reflect.Value) (reflect.Value, bool) {
	if copied.IsValid() {
		return copied, true
	}

	return original, false
}

// restore swaps each token for its original text as a YAML double-quoted
// scalar. strconv.Quote's escapes (\t, \n, \", \\, \xNN, \uNNNN, \UNNNNNNNN,
// \a, \b, \f, \v, \r) all mean the same in YAML, and only valid UTF-8 is
// tokenized. A token the encoder dropped (a `yaml:"-"` field) is skipped.
func (q *textQuoter) restore(encoded []byte) ([]byte, error) {
	for index, original := range q.originals {
		token := []byte(q.token(index))
		if bytes.Count(encoded, token) > 1 {
			return nil, ErrQuotedTextCollision
		}
		encoded = bytes.Replace(encoded, token, []byte(strconv.Quote(original)), 1)
	}

	return encoded, nil
}

// yamlKeyLess orders string mapping keys exactly as go-yaml's encoder does
// (go.yaml.in/yaml/v3 sorter.go, keyList.Less, string branch): runs of digits
// compare numerically, so "person-2" sorts before "person-10". A batched
// encoding has to split collections in this order, or concatenated batches
// would list keys differently from one yaml.Marshal of the whole map.
func yamlKeyLess(a, b string) bool {
	ar, br := []rune(a), []rune(b)
	digits := false
	for i := 0; i < len(ar) && i < len(br); i++ {
		if ar[i] == br[i] {
			digits = unicode.IsDigit(ar[i])

			continue
		}
		al, bl := unicode.IsLetter(ar[i]), unicode.IsLetter(br[i])
		if al && bl {
			return ar[i] < br[i]
		}
		if al || bl {
			if digits {
				return al
			}

			return bl
		}

		return digitRunLess(ar, br, i)
	}

	return len(ar) < len(br)
}

// digitRunLess compares the digit runs of ar and br that start at their first
// difference i, as go-yaml does (including its treatment of a run that began
// before i with a nonzero digit).
func digitRunLess(ar, br []rune, i int) bool {
	var start int64
	if ar[i] == '0' || br[i] == '0' {
		for j := i - 1; j >= 0 && unicode.IsDigit(ar[j]); j-- {
			if ar[j] != '0' {
				start = 1

				break
			}
		}
	}
	an, ai := digitRun(ar, i, start)
	bn, bi := digitRun(br, i, start)
	if an != bn {
		return an < bn
	}
	if ai != bi {
		return ai < bi
	}

	return ar[i] < br[i]
}

// digitRun reads the run of digits in runes from start, beginning from value,
// and returns its number and the index just past it.
func digitRun(runes []rune, start int, value int64) (int64, int) {
	end := start
	for ; end < len(runes) && unicode.IsDigit(runes[end]); end++ {
		value = value*10 + int64(runes[end]-'0')
	}

	return value, end
}
