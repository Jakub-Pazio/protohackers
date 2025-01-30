package main

import (
	"bean/pkg/pserver"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
)

var portNumber = flag.Int("port", 4242, "Port number of server")

func main() {
	flag.Parse()
	db := newDatabase()
	log.Fatal(pserver.ListenServeUDP(db.handler, *portNumber))
}

type Database struct {
	db map[string]string

	mu sync.Mutex
}

func newDatabase() *Database {
	return &Database{
		db: make(map[string]string),
		mu: sync.Mutex{},
	}
}

func (d *Database) setValue(key string, value string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.db[key] = value
	return nil
}

func (d *Database) getValue(key string) (string, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	val, ok := d.db[key]
	return val, ok
}

func (d *Database) handler(msg string) string {
	parts := strings.Split(msg, "=")
	if len(parts) == 1 {
		log.Printf("got Retrieve to key: %s\n", parts[0])
		if parts[0] == "version" {
			return "version=Jakub's Key-Store v0.0.1"
		}
		val, ok := d.getValue(parts[0])
		if !ok {
			return fmt.Sprintf("%s=", parts[0])
		}
		return fmt.Sprintf("%s=%s", parts[0], val)
	} else {
		log.Printf("Insert, key: %q, value:%q\n", parts[0],
			strings.Join(parts[1:], "="))
		_ = d.setValue(parts[0], strings.Join(parts[1:], "="))
		return ""
	}
}
