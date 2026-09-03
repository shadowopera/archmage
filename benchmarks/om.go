package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/goccy/go-yaml"
)

// Pair is one entry of an OM.
type Pair struct {
	Key string
	Val any
}

// OM is an ordered map. Config files have to come out byte-identical for a
// given seed, which Go's map type cannot promise, so every object in the
// generated tree data is built as an OM instead.
//
// Keys are always strings, including numeric ones. Archmage parses a quoted
// number as an integer wherever an integer key is declared, and JSON has no
// other option anyway.
type OM []Pair

// MarshalJSONTo implements encoding/json/v2's MarshalerTo.
func (m OM) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for _, p := range m {
		if err := enc.WriteToken(jsontext.String(p.Key)); err != nil {
			return err
		}
		if err := json.MarshalEncode(enc, p.Val); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndObject)
}

// MarshalYAML implements goccy/go-yaml's InterfaceMarshaler.
func (m OM) MarshalYAML() (any, error) {
	out := make(yaml.MapSlice, len(m))
	for i, p := range m {
		out[i] = yaml.MapItem{Key: p.Key, Value: p.Val}
	}
	return out, nil
}

// set appends or replaces a key, keeping insertion order.
func (m OM) set(key string, val any) OM {
	for i := range m {
		if m[i].Key == key {
			m[i].Val = val
			return m
		}
	}
	return append(m, Pair{key, val})
}
