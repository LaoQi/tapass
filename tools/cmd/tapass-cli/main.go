package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/tapass/tapass-tools/vault"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "create":
		cmdCreate()
	case "set":
		cmdSet()
	case "get":
		cmdGet()
	case "delete":
		cmdDelete()
	case "list":
		cmdList()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `tapass - tapass V1 password manager CLI

Usage:
  tapass create <file> <password>
  tapass set    <file> <password> <key> <value>
  tapass get    <file> <password> <key>
  tapass delete <file> <password> <key>
  tapass list   <file> <password>
`)
}

func cmdCreate() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "usage: tapass create <file> <password>\n")
		os.Exit(1)
	}
	file := os.Args[2]
	password := os.Args[3]

	if err := vault.Create(file, password); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("vault created:", file)
}

func cmdSet() {
	if len(os.Args) != 6 {
		fmt.Fprintf(os.Stderr, "usage: tapass set <file> <password> <key> <value>\n")
		os.Exit(1)
	}
	file := os.Args[2]
	password := os.Args[3]
	key := os.Args[4]
	value := os.Args[5]

	v, err := vault.Open(file, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := v.Set(key, []byte(value)); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func cmdGet() {
	if len(os.Args) != 5 {
		fmt.Fprintf(os.Stderr, "usage: tapass get <file> <password> <key>\n")
		os.Exit(1)
	}
	file := os.Args[2]
	password := os.Args[3]
	key := os.Args[4]

	v, err := vault.Open(file, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	val, ok := v.Get(key)
	if !ok {
		fmt.Fprintf(os.Stderr, "key not found: %s\n", key)
		os.Exit(1)
	}
	fmt.Println(string(val))
}

func cmdDelete() {
	if len(os.Args) != 5 {
		fmt.Fprintf(os.Stderr, "usage: tapass delete <file> <password> <key>\n")
		os.Exit(1)
	}
	file := os.Args[2]
	password := os.Args[3]
	key := os.Args[4]

	v, err := vault.Open(file, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := v.Delete(key); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("deleted:", key)
}

func cmdList() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "usage: tapass list <file> <password>\n")
		os.Exit(1)
	}
	file := os.Args[2]
	password := os.Args[3]

	v, err := vault.Open(file, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	entries := v.List()
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		e := entries[k]
		ts := time.UnixMilli(int64(e.Timestamp)).Format("2006-01-02 15:04:05")
		typeName := "text"
		if e.Type == 2 {
			typeName = "blob"
		}
		fmt.Printf("%s  %s  %s  %s\n", ts, typeName, k, string(e.Value))
	}
}
