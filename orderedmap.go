// The MIT License (MIT)
//
// Copyright (c) 2023 Cypherpunk LLC and contributors
// Copyright (c) 2017 Ian Coleman
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, Subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or Substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package orderedmap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

type pair struct {
	key   string
	value any
}

func (kv *pair) Key() string {
	return kv.key
}

func (kv *pair) Value() any {
	return kv.value
}

type byPair struct {
	Pairs    []*pair
	LessFunc func(a *pair, j *pair) bool
}

func (a byPair) Len() int           { return len(a.Pairs) }
func (a byPair) Swap(i, j int)      { a.Pairs[i], a.Pairs[j] = a.Pairs[j], a.Pairs[i] }
func (a byPair) Less(i, j int) bool { return a.LessFunc(a.Pairs[i], a.Pairs[j]) }

type OrderedMap struct {
	keys   []string
	values map[string]any
}

func New() *OrderedMap {
	o := OrderedMap{}
	o.keys = []string{}
	o.values = map[string]any{}
	return &o
}

func (o *OrderedMap) Get(key string) any {
	return o.values[key]
}

func (o *OrderedMap) Set(key string, value any) {
	_, ok := o.values[key]
	if !ok {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

func (o *OrderedMap) Delete(key string) {
	// check key is in use
	_, ok := o.values[key]
	if !ok {
		return
	}
	// remove from keys
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
	// remove from values
	delete(o.values, key)
}

func (o *OrderedMap) Keys() []string {
	return o.keys
}

func (o *OrderedMap) Values() []any {
	v := make([]any, len(o.keys))
	for i, k := range o.keys {
		v[i] = o.values[k]
	}
	return v
}

func (o *OrderedMap) KeysValues() map[string]any {
	return o.values
}

func (o *OrderedMap) Len() int {
	return len(o.keys)
}

func (o *OrderedMap) GetValueAt(pos int) any {
	k := o.keys[pos]
	return o.values[k]
}

func (o *OrderedMap) GetKeyAt(pos int) string {
	return o.keys[pos]
}

// SortKeys sorts the map keys using the provided sort func.
func (o *OrderedMap) SortKeys(sortFunc func(keys []string)) {
	sortFunc(o.keys)
}

// Sort sorts the map using the provided less func.
func (o *OrderedMap) Sort(lessFunc func(a *pair, b *pair) bool) {
	pairs := make([]*pair, len(o.keys))
	for i, key := range o.keys {
		pairs[i] = &pair{key, o.values[key]}
	}

	sort.Sort(byPair{pairs, lessFunc})

	for i, pair := range pairs {
		o.keys[i] = pair.key
	}
}

// MarshalJSON must return no duplicates, and should since orderedMap keys are
// unique.
func (o OrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		// add key
		if err := encoder.Encode(k); err != nil {
			return nil, err
		}
		buf.WriteByte(':')
		// add value
		if err := encoder.Encode(o.values[k]); err != nil {
			return nil, err
		}
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func (o *OrderedMap) UnmarshalJSON(b []byte) error {
	err := CheckDuplicate(json.NewDecoder(bytes.NewReader(b)))
	if err != nil {
		return err
	}

	if o.values == nil {
		o.values = map[string]any{}
	}
	err = json.Unmarshal(b, &o.values)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err = dec.Token(); err != nil { // skip '{'
		return err
	}
	o.keys = make([]string, 0, len(o.values))
	return decode(dec, o)
}

// decodeOrderedMap
func decode(dec *json.Decoder, o *OrderedMap) error {
	hasKey := make(map[string]bool, len(o.values))
	for {
		token, err := dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok && delim == '}' {
			return nil
		}
		key := token.(string)
		if hasKey[key] {
			// duplicate key
			for j, k := range o.keys {
				if k == key {
					copy(o.keys[j:], o.keys[j+1:])
					break
				}
			}
			o.keys[len(o.keys)-1] = key
		} else {
			hasKey[key] = true
			o.keys = append(o.keys, key)
		}

		token, err = dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				if values, ok := o.values[key].(map[string]any); ok {
					newMap := OrderedMap{
						keys:   make([]string, 0, len(values)),
						values: values,
					}
					if err = decode(dec, &newMap); err != nil {
						return err
					}
					o.values[key] = newMap
				} else if oldMap, ok := o.values[key].(OrderedMap); ok {
					newMap := OrderedMap{
						keys:   make([]string, 0, len(oldMap.values)),
						values: oldMap.values,
					}
					if err = decode(dec, &newMap); err != nil {
						return err
					}
					o.values[key] = newMap
				} else if err = decode(dec, &OrderedMap{}); err != nil {
					return err
				}
			case '[':
				if values, ok := o.values[key].([]any); ok {
					if err = decodeSlice(dec, values); err != nil {
						return err
					}
				} else if err = decodeSlice(dec, []any{}); err != nil {
					return err
				}
			}
		}
	}
}

func decodeSlice(dec *json.Decoder, s []any) error {
	for index := 0; ; index++ {
		token, err := dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				if index < len(s) {
					if values, ok := s[index].(map[string]any); ok {
						newMap := OrderedMap{
							keys:   make([]string, 0, len(values)),
							values: values,
						}
						if err = decode(dec, &newMap); err != nil {
							return err
						}
						s[index] = newMap
					} else if oldMap, ok := s[index].(OrderedMap); ok {
						newMap := OrderedMap{
							keys:   make([]string, 0, len(oldMap.values)),
							values: oldMap.values,
						}
						if err = decode(dec, &newMap); err != nil {
							return err
						}
						s[index] = newMap
					} else if err = decode(dec, &OrderedMap{}); err != nil {
						return err
					}
				} else if err = decode(dec, &OrderedMap{}); err != nil {
					return err
				}
			case '[':
				if index < len(s) {
					if values, ok := s[index].([]any); ok {
						if err = decodeSlice(dec, values); err != nil {
							return err
						}
					} else if err = decodeSlice(dec, []any{}); err != nil {
						return err
					}
				} else if err = decodeSlice(dec, []any{}); err != nil {
					return err
				}
			case ']':
				return nil
			}
		}
	}
}

// ErrJSONDuplicate allows applications to check for JSON duplicate error.
type ErrJSONDuplicate error

// CheckDuplicate checks for JSON duplicates on ingest (unmarshal).  Note that
// Go maps and structs and Javascript objects (ES6) already require unique
// JSON names.  See the Coze FAQ on duplicates.
//
// Duplicate JSON fields are a security issue that wasn't addressed by the
// original spec that results in surprising behavior and is a source of bugs.
// See the article, "[An Exploration of JSON Interoperability
// Vulnerabilities](https://bishopfox.com/blog/json-interoperability-vulnerabilities)"
// and control-f "duplicate".
//
// Until Go releases the planned revision to the JSON package (See
// https://github.com/go-json-experiment/json), or adds support for erroring on
// duplicates to the current package, this function is needed.
//
// After JSON was widely adopted, Douglas Crockford (JSON's inventor), tried to
// fix this by updating JSON to define "must error on duplicates" as the correct
// behavior,  but it was decided it was too late
// (https://esdiscuss.org/topic/json-duplicate-keys).
//
// Although Douglas Crockford couldn't change the JSON spec to force
// implementations to error on duplicate, his Java JSON implementation errors on
// duplicates. Others implementations behaviors are `last-value-wins`, support
// duplicate keys, or other non-standard behavior. The [JSON
// RFC](https://datatracker.ietf.org/doc/html/rfc8259#section-4) states that
// implementations should not allow duplicate keys.  It then notes the varying behavior
// of existing implementations.
//
// Disallowing duplicates conforms to the small I-JSON RFC. The author of
// I-JSON, Tim Bray, is also the author of current JSON specification (RFC
// 8259).  See also https://github.com/json5/json5-spec/issues/38.
func CheckDuplicate(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return err
	}

	// Is it a delimiter?
	delim, ok := t.(json.Delim)
	if !ok {
		return nil // scaler type, nothing to do
	}

	switch delim {
	case '{':
		keys := make(map[string]bool)
		for d.More() {
			t, err := d.Token() // Get field key.
			if err != nil {
				return err
			}

			key := t.(string)
			if keys[key] { // Check for duplicates.
				return ErrJSONDuplicate(fmt.Errorf("Coze: JSON duplicate field %q", key))
			}
			keys[key] = true

			// Recursive, Check value in case value is object.
			err = CheckDuplicate(d)
			if err != nil {
				return err
			}
		}
		// consume trailing }
		if _, err := d.Token(); err != nil {
			return err
		}

	case '[':
		for d.More() {
			if err := CheckDuplicate(d); err != nil {
				return err
			}
		}
		// consume trailing ]
		if _, err := d.Token(); err != nil {
			return err
		}
	}
	return nil
}
