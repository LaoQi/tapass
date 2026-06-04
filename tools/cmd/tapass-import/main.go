package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/tapass/tapass-tools/internal/importer"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <keepass.xml> <output.tap>\n", os.Args[0])
		os.Exit(1)
	}

	xmlPath := os.Args[1]
	tapPath := os.Args[2]

	if _, err := os.Stat(xmlPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot access %s: %v\n", xmlPath, err)
		os.Exit(1)
	}

	password, err := readPassword("Enter master password: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	confirm, err := readPassword("Confirm master password: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if string(password) != string(confirm) {
		fmt.Fprintln(os.Stderr, "Error: passwords do not match")
		os.Exit(1)
	}

	stats, err := importer.Import(xmlPath, tapPath, string(password))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Import complete: %d groups, %d entries, %d TOTP", stats.Groups, stats.Entries, stats.TOTP)
	if stats.Skipped > 0 {
		fmt.Printf(", %d skipped (recycle bin)", stats.Skipped)
	}
	fmt.Println()
}

func readPassword(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	fd := int(syscall.Stdin)
	if term.IsTerminal(fd) {
		pw, err := term.ReadPassword(fd)
		fmt.Println()
		return pw, err
	}

	var buf [256]byte
	n, err := os.Stdin.Read(buf[:])
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}
