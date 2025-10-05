// Package jsonfile provides a way to parse JSON files including single line comments indicated by // at the beginning.
package jsonfile

import "encoding/json"

// Parses a JSON file at fileName into the provided interface, which must be of a pointer type.
func ParseFile(fileName string, v interface{}) (err error) {
	cf, err := newCommentFilter(fileName)
	if err != nil {
		return
	}

	// read filtered JSON and unmarshal it into the provided interface
	err = json.NewDecoder(cf).Decode(v)
	return
}
