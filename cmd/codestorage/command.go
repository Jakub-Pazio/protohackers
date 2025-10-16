package main

import (
	"fmt"
	"strings"
)

type Method string

const (
	HelpMethod Method = "HELP"
	GetMethod  Method = "GET"
	PutMethod  Method = "PUT"
	ListMethod Method = "LIST"
)

type MethodParseError string

func (c MethodParseError) Error() string {
	return fmt.Sprintf("illegal method: %s", string(c))
}

func ParseMethod(s string) (Method, error) {
	capital := strings.ToUpper(s)

	switch capital {
	case "HELP":
		return HelpMethod, nil
	case "GET":
		return GetMethod, nil
	case "PUT":
		return PutMethod, nil
	case "LIST":
		return ListMethod, nil
	default:
		return "", MethodParseError(s)
	}
}

func MethodUsage(m Method) string {
	switch m {
	case "HELP":
		return "usage: HELP|GET|PUT|LIST"
	case "GET":
		return "usage: GET file [revision]"
	case "PUT":
		return "usage: PUT file length newline data"
	case "LIST":
		return "usage: LIST dir"
	default:
		return "usage: bad command"
	}
}
