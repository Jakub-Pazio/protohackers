package main

import "testing"

func TestAddFile(t *testing.T) {
	root := CreateRoot()

	root.AddFile("/aa", "some text")

	rootSize := len(root.Children)
	want := 1

	if rootSize != want {
		t.Errorf("size is %d, want %d\n", rootSize, want)
	}

	file := root.Children[0]
	revision := len(file.Revisions)
	want = 1

	if revision != want {
		t.Errorf("revision is %d, want %d\n", revision, want)
	}

	root.AddFile("/aa", "second rev")
	rootSize = len(root.Children)
	want = 1

	if rootSize != want {
		t.Errorf("size is %d, want %d\n", rootSize, want)
	}

	file = root.Children[0]
	revision = len(file.Revisions)
	want = 2

	if revision != want {
		t.Errorf("revision is %d, want %d\n", revision, want)
	}
}

func TestAddDirectoryAndFile(t *testing.T) {
	root := CreateRoot()

	root.AddFile("/foo/bar", "ddd")

	size := len(root.Children)
	want := 1

	if size != want {
		t.Errorf("got %d, want %d\n", size, want)
	}

	foo := root.Children[0]

	if !foo.Directory {
		t.Error(" fooShould be directory")
	}

	size = len(foo.Children)

	if size != want {
		t.Errorf("got %d, want %d\n", size, want)
	}

	bar := foo.Children[0]

	if bar.Name != "bar" {
		t.Errorf("got %q, want %q\n", bar.Name, "bar")
	}

	if len(bar.Revisions) != want {
		t.Errorf("revision is not %d but %d\n", want, len(bar.Revisions))
	}

	root.AddFile("/foo/bar", "eeeeeee")

	bar = foo.Children[0]

	if bar.Name != "bar" {
		t.Errorf("got %q, want %q\n", bar.Name, "bar")
	}

	want = 2
	if len(bar.Revisions) != want {
		t.Errorf("revision is not %d but %d\n", want, len(bar.Revisions))
	}

}
