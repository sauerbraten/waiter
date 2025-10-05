package jsonfile

import (
	"bufio"
	"errors"
	"io"
	"os"
)

type commentFilter struct {
	file     io.ByteReader
	inString bool // true if current position points inside a string; used to only strip whitespace outside of strings
}

func newCommentFilter(fileName string) (cf *commentFilter, err error) {
	var file *os.File
	file, err = os.Open(fileName)

	cf = &commentFilter{file: bufio.NewReader(file)}

	return
}

// Reads from the file and strips whitespace outside of strings as well as comments. With this method, fileTilter implements io.Reader.
func (cf *commentFilter) Read(p []byte) (n int, err error) {
	var b, c byte

	for n < len(p) {
		b, err = cf.file.ReadByte()
		if err != nil {
			return
		}

		if cf.inString {
			// use byte as-is
			p[n] = b
			n++

			// check if this is the end of the string
			if rune(b) == '"' {
				cf.inString = !cf.inString
			}
		} else {
			switch rune(b) {
			case '/':
				// this is a comment, next byte has to be '/' as well, else it's invalid JSON
				c, err = cf.file.ReadByte()
				if err != nil {
					return
				}

				if c == '/' {
					// skip until new line
					for c, err = cf.file.ReadByte(); err == nil && rune(c) != '\n'; c, err = cf.file.ReadByte() {
					}

					if err != nil {
						return
					}
				} else {
					// '/' is an illegal character in JSON outside of a string
					err = errors.New("illegal character '/' outside of string")
					return
				}

			case ' ', '\t', '\n':
				// skip whitespace

			case '"':
				// use byte
				p[n] = b
				n++
				// entering or exiting a string
				cf.inString = !cf.inString

			default:
				// use byte as-is
				p[n] = b
				n++
			}
		}
	}

	return
}
