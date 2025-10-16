package main

import (
	"fmt"
	"log"
	"slices"
	"strings"
)

type Node struct {
	Name      string
	Directory bool
	Revisions []string
	Children  []*Node
}

func CreateRoot() Node {
	return Node{
		Name:      "",
		Directory: true,
		Children:  make([]*Node, 0),
	}
}

func CreateDirectory(name string) Node {
	return Node{
		Name:      name,
		Directory: true,
		Children:  make([]*Node, 0),
	}
}

func CreateFile(name, content string) Node {
	r := make([]string, 0)
	r = append(r, content)
	return Node{
		Name:      name,
		Revisions: r,
	}
}

func (n *Node) AddDirectory(name string) *Node {
	dir := CreateDirectory(name)
	n.Children = append(n.Children, &dir)
	return &dir
}

func (n *Node) AddRevision(content string) {
	if content == n.Revisions[len(n.Revisions)-1] {
		return
	}
	n.Revisions = append(n.Revisions, content)
}

func (n *Node) AddFile(name, content string) (int, error) {
	if !n.Directory {
		return 0, fmt.Errorf("not a directory")
	}

	if !isDirectory(name) {
		if len(name) == 0 || name[0] == '/' {
			name = name[1:]
		}
		log.Printf("adding file: %q, to directory: %q, with content: %q\n", name, n.Name, content)
		for _, child := range n.Children {
			if !child.Directory && child.Name == name {
				child.AddRevision(content)
				return len(child.Revisions), nil
			}
		}

		newFile := CreateFile(name, content)
		n.Children = append(n.Children, &newFile)
		return 1, nil
	}

	dir, cutName := splitName(name)

	for _, child := range n.Children {
		if child.Directory && child.Name == dir {
			return child.AddFile(cutName, content)
		}
	}

	newdir := n.AddDirectory(dir)

	r, err := newdir.AddFile(cutName, content)
	if err != nil {
		n.RemoveDir(dir)
	}

	return r, err
}

func (n *Node) RemoveDir(dir string) {
	n.Children = slices.DeleteFunc(n.Children, func(no *Node) bool {
		return no.Directory && no.Name == dir
	})
}

func finalDir(path string) bool {
	slashes := strings.Count(path, "/")
	last := path[len(path)-1]
	if slashes < 2 || (slashes == 2 && last == '/') {
		return true
	}
	return false
}

func splitName(name string) (string, string) {
	log.Printf("Spriting string %q\n", name)

	slashIndex := strings.Index(name[1:], "/")
	slashIndex += 1
	dir, rest := name[1:slashIndex], name[slashIndex:]

	log.Printf("directory: %q, rest: %q\n", dir, rest)

	return dir, rest
}

func isDirectory(name string) bool {
	return strings.Contains(name[1:], "/")
}
