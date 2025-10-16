package main

import (
	"bean/pkg/pserver"
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"slices"
	"strconv"
	"strings"
)

var portNumber = flag.Int("port", 4242, "Port number of server")

func main() {
	flag.Parse()

	server := &StorageServer{root: CreateRoot()}

	handler := pserver.WithMiddleware(
		server.handleConnection,
		pserver.LoggingMiddleware,
	)

	log.Fatal(pserver.ListenServe(handler, *portNumber))
}

var globalWasError = false

var noSuchRevisionError = fmt.Errorf("no such revision")
var noSuchFileError = fmt.Errorf("no such file")
var illegalFileNameError = fmt.Errorf("illegal file name")
var illegalDirectoryNameError = fmt.Errorf("illegal dir name")

type StorageServer struct {
	root Node
}

func (s *StorageServer) handleConnection(conn net.Conn) {
	br := bufio.NewReader(conn)

	for {
		fmt.Printf("s.root: %v\n", s.root)
		if !globalWasError {
			showPrompt(conn)
		} else {
			globalWasError = false
		}
		line, err := readLine(br)
		if err != nil {
			log.Printf("Error reading line: %v\n", err)
			break
		}

		first, args := ParseLine(line)

		method, err := ParseMethod(first)

		if err != nil {
			writeError(conn, err)
			conn.Close()
			return
		}

		switch method {
		case HelpMethod:
			writeHelpMessage(conn)
		case GetMethod:
			if len(args) != 1 && len(args) != 2 {
				writeUsageMessage(conn, method)
				continue
			}

			fileName := args[0]

			if !IsPrintableASCII(fileName) {
				writeError(conn, illegalFileNameError)
				continue
			}

			revisions, err := s.handleGet(fileName, s.root)
			if err != nil {
				writeError(conn, err)
				continue
			}

			revision := len(revisions)

			if len(args) == 2 {
				r, err := parseRevision(args[1])
				if err == nil {
					revision = r
				} else {
					revision = -1
				}
			}

			if len(revisions) < revision || revision < 1 {
				writeError(conn, noSuchRevisionError)
				continue
			}

			writeFile(conn, revisions[revision-1])

		case PutMethod:
			log.Printf("put args: %d\n", len(args))
			log.Printf("%+v\n", args)
			if !(len(args) == 1 || len(args) == 2) {
				writeUsageMessage(conn, method)
				continue
			}

			if !IsPrintableASCII(args[0]) {
				log.Printf("IS THIS ERROR?? %q\n", args[0])
				writeError(conn, illegalFileNameError)
				continue
			}

			log.Printf("%q was legal\n", args[0])

			rev, err := s.handlePut(args, br)
			if err != nil {
				writeError(conn, err)
				continue
			}
			writeRevision(rev, conn)
		case ListMethod:
			if len(args) != 1 {
				writeUsageMessage(conn, method)
				continue
			}

			if !IsPrintableASCII(args[0]) {
				writeError(conn, illegalDirectoryNameError)
				continue
			}

			entries := s.handleList(args, s.root)
			slices.Sort(entries)

			writeOkSize(conn, len(entries))
			writeEntries(conn, entries)
		}
	}

	conn.Close()
}

func writeFile(conn net.Conn, s string) {
	writeOkSize(conn, len(s))
	conn.Write([]byte(s))
}

func parseRevision(s string) (int, error) {
	if s == "" {
		return 0, noSuchRevisionError
	}

	if s[0] == 'r' {
		s = s[1:]
	}

	n, err := strconv.Atoi(s)

	if err != nil {
		return 0, noSuchRevisionError
	}

	return n, nil
}

func writeRevision(rev int, conn net.Conn) {
	msg := fmt.Sprintf("OK r%d\n", rev)

	conn.Write([]byte(msg))
}

func showPrompt(conn net.Conn) {
	msg := "READY\n"

	conn.Write([]byte(msg))
}

func writeOkSize(conn net.Conn, n int) {
	msg := fmt.Sprintf("OK %d\n", n)

	conn.Write([]byte(msg))
}

func writeEntries(conn net.Conn, entries []string) {
	for _, e := range entries {
		conn.Write([]byte(e))
	}
}

func (s *StorageServer) handleList(args []string, n Node) []string {
	dir := args[0]
	dir = strings.TrimSpace(dir)

	if dir == "/" {
		fmt.Printf("len(n.Children): %v\n", len(n.Children))
		res := make([]string, 0)
		for _, ch := range n.Children {
			if ch.Directory {
				entry := fmt.Sprintf("%s/ DIR\n", ch.Name)
				res = append(res, entry)
			} else {
				entry := fmt.Sprintf("%s r%d\n", ch.Name, len(ch.Revisions))
				res = append(res, entry)
			}
		}
		return res
	}

	if finalDir(dir) {
		log.Printf("Final dir is: %q\n", dir)
		if dir[len(dir)-1] == '/' {
			dir = dir[:len(dir)-1]
		}
		if dir[0] == '/' {
			dir = dir[1:]
		}
		for _, child := range n.Children {
			if child.Name == dir && child.Directory {
				res := make([]string, 0)
				for _, ch := range child.Children {
					if ch.Directory {
						entry := fmt.Sprintf("%s/ DIR\n", ch.Name)
						res = append(res, entry)
					} else {
						entry := fmt.Sprintf("%s r%d\n", ch.Name, len(ch.Revisions))
						res = append(res, entry)
					}
				}
				return res
			}
		}
		return []string{}
	}
	dir, rest := splitName(dir)

	for _, child := range n.Children {
		if child.Name == dir && child.Directory {
			return s.handleList([]string{rest}, *child)
		}
	}

	return []string{}
}

func (s *StorageServer) handlePut(args []string, br *bufio.Reader) (int, error) {
	filename := args[0]
	var lenInt int
	var content string

	if len(args) == 2 {
		length := args[1]
		length = strings.TrimSpace(length)
		log.Printf("bytes tring: %q\n", length)
		parsed, err := strconv.Atoi(length)
		fmt.Printf("parsed: %v\n", parsed)
		if err == nil {
			lenInt = parsed
		}
	}

	log.Printf("Bytes to read: %d\n", lenInt)

	if lenInt > 0 {
		buf := make([]byte, lenInt)
		// i, err := br.Read(buf)
		i, err := io.ReadFull(br, buf)
		if err != nil {
			return 0, fmt.Errorf("could not read file content: %v", err)
		}

		if i != lenInt {
			return 0, fmt.Errorf("file size %d, read %d bytes", lenInt, i)
		}

		content = string(buf)
	}

	return s.root.AddFile(filename, content)
}

func (s *StorageServer) handleGet(file string, node Node) ([]string, error) {
	if file[0] == '/' {
		file = file[1:]
	}

	if !strings.Contains(file, "/") {
		// treat file as file name and see if current node has it, if yes return
		for _, child := range node.Children {
			if child.Name == file && !child.Directory {
				return child.Revisions, nil
			}
		}
	}

	dir, rest := splitDirectory(file)

	for _, child := range node.Children {
		if child.Name == dir && child.Directory {
			return s.handleGet(rest, *child)
		}
	}

	return nil, noSuchFileError
}

func splitDirectory(file string) (string, string) {
	split := strings.SplitN(file, "/", 2)

	return split[0], split[1]
}

func ParseLine(line string) (string, []string) {
	line = strings.TrimSpace(line)
	parts := strings.Split(line, " ")
	return parts[0], parts[1:]
}

func writeHelpMessage(conn net.Conn) {
	usage := MethodUsage(HelpMethod)
	msg := fmt.Sprintf("OK %s\n", usage)

	conn.Write([]byte(msg))
}

func writeUsageMessage(conn net.Conn, method Method) {
	usage := MethodUsage(method)
	msg := fmt.Sprintf("ERR %s\n", usage)

	conn.Write([]byte(msg))
}

func readLine(br *bufio.Reader) (string, error) {
	return br.ReadString('\n')
}

func writeError(conn net.Conn, err error) {
	msg := fmt.Sprintf("ERR %s\n", err.Error())

	conn.Write([]byte(msg))
	globalWasError = true
}

func IsPrintableASCII(s string) bool {
	prevSlash := false
	for i := range s {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z':
			prevSlash = false
		case c >= 'a' && c <= 'z':
			prevSlash = false
		case c == '/':
			if prevSlash {
				return false
			}
			prevSlash = true
		case c >= '0' && c <= '9':
			prevSlash = true
		case c == '.':
			prevSlash = false
		default:
			return false
		}
	}
	return true
}
