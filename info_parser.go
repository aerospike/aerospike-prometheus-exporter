package main

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// InfoParser provides a reader for Aerospike cluster's response for any of the metric
type InfoParser struct {
	*bufio.Reader
}

// NewInfoParser provides an instance of the InfoParser
func NewInfoParser(s string) *InfoParser {
	return &InfoParser{bufio.NewReader(strings.NewReader(s))}
}

// Expect validates the expected value against the one returned by the InfoParser
func (ip *InfoParser) Expect(s string) error {
	bytes := make([]byte, len(s))
	v, err := ip.Read(bytes)
	if err != nil {
		return err
	}
	if string(bytes) != s {
		return fmt.Errorf("InfoParser: Wrong value. Expected %s, found %s", s, v)
	}
	return nil
}

// ReadUntil reads bytes from the InfoParser by handeling some edge-cases
func (ip *InfoParser) ReadUntil(delim byte) (string, error) {
	v, err := ip.ReadBytes(delim)

	switch len(v) {
	case 0:
		return string(v), err
	case 1:
		if v[0] == delim {
			return "", err
		}
		return string(v), err
	}
	return string(v[:len(v)-1]), err
}

// ReadFloat reads from the InfoParser and converts the result to Float
func (ip *InfoParser) ReadFloat(delim byte) (float64, error) {
	s, err := ip.ReadUntil(delim)
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(s, 64)
}
