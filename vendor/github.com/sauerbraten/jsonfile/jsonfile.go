// Package jsonfile provides a way to parse JSON files including single line comments indicated by // at the beginning.
package jsonfile

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
)

type fileFilter struct {
	file     io.ByteReader
	pos      int  // current position in the file
	inString bool // true if current position points inside a string; used to only strip whitespace outside of strings
}

func newFileFilter(fileName string) (ff *fileFilter, err error) {
	var file *os.File
	file, err = os.Open(fileName)

	ff = &fileFilter{file: bufio.NewReader(file)}

	return
}

// Reads from the file and strips whitespace outside of strings as well as comments. With this method, fileTilter implements io.Reader.
func (ff *fileFilter) Read(p []byte) (n int, err error) {
	skippedBytes := 0
	var b, c byte

	for n < len(p) {
		b, err = ff.file.ReadByte()
		if err != nil {
			return
		}

		if ff.inString {
			// use byte as-is
			p[n] = b
			n++

			// check if this is the end of the string
			if rune(b) == '"' {
				ff.inString = !ff.inString
			}
		} else {
			switch rune(b) {
			case '/':
				// this is a comment, next byte has to be '/' as well, else it's invalid JSON
				c, err = ff.file.ReadByte()
				if err != nil {
					return
				}
				skippedBytes += 2

				if c == '/' {
					// skip until new line
					for c, err = ff.file.ReadByte(); err == nil && rune(c) != '\n'; c, err = ff.file.ReadByte() {
						skippedBytes++
					}

					if err != nil {
						return
					}
					skippedBytes++
				} else {
					// '/' is an illegal character in JSON outside of a string
					err = errors.New("illegal character '/' outside of string")
					return
				}

			case ' ', '\t', '\n':
				// skip whitespace
				skippedBytes++

			case '"':
				// use byte
				p[n] = b
				n++
				// entering or exiting a string
				ff.inString = !ff.inString

			default:
				// use byte as-is
				p[n] = b
				n++
			}
		}
	}

	// advance position in file stream
	ff.pos += skippedBytes + n

	return
}

// Parses a JSON file at fileName into the provided interface, which must be of a pointer type.
func ParseFile(fileName string, v interface{}) (err error) {
	ff, err := newFileFilter(fileName)
	if err != nil {
		return
	}

	// read filtered JSON and unmarshal it into the provided interface
	err = json.NewDecoder(ff).Decode(v)
	return
}
