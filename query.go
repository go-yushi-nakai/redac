package redac

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	ErrQueryOpenParamNotFound    = errors.New("`{{` not found")
	ErrorQueryCloseParamNotFound = errors.New("`}}` not found")
)

type Query struct {
	Data       string
	Parameters []string
}

func NewQuery(s string) (*Query, error) {
	q := &Query{}
	q.Data = s
	pos := 0
	for {
		key, idx, err := findParameter(s)
		if err == ErrQueryOpenParamNotFound {
			return q, nil
		}
		if err == ErrorQueryCloseParamNotFound {
			return nil, fmt.Errorf("syntax error at idx %d: %w", pos, err)
		}
		q.Parameters = append(q.Parameters, key)
		pos += idx
		s = s[idx:]
	}
}

func (q *Query) GetTemplateParams(args []string) (map[string]any, error) {
	if len(args) != len(q.Parameters) {
		return nil, fmt.Errorf("argument error, expected %+v, got %+v", q.Parameters, args)
	}

	params := map[string]any{}
	for i, key := range q.Parameters {
		params[key] = args[i]
	}
	return params, nil
}

func (q *Query) GetParameterStringForUsage() string {
	if len(q.Parameters) == 0 {
		return "this query has no parameter"
	}

	var v strings.Builder
	for _, p := range q.Parameters {
		v.WriteString(surroundString(p, "<", ">"))
		v.WriteString(" ")
	}
	return v.String()
}

func LoadQueryFromFile(path string) (*Query, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	if bytes.HasPrefix(b, []byte("#!")) {
		for i := 0; i < len(b); i++ {
			if b[i] == '\n' {
				return NewQuery(string(b[i:]))
			}
		}
	}
	return NewQuery(string(b))
}

func findParameter(s string) (string, int, error) {
	open := strings.Index(s, "{{")
	close := strings.Index(s, "}}")
	if open == -1 {
		return "", 0, ErrQueryOpenParamNotFound
	}
	if close == -1 {
		return "", 0, ErrorQueryCloseParamNotFound
	}
	ss := s[open+2 : close]
	key := strings.TrimSpace(ss)
	return key, close + 2, nil
}

func surroundString(s, prefix, suffix string) string {
	var v strings.Builder
	v.WriteString(prefix)
	v.WriteString(s)
	v.WriteString(suffix)
	return v.String()
}
