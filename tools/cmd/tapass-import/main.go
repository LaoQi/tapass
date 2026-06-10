package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/tapass/tapass-tools/internal/importer"
	"github.com/tapass/tapass-tools/version"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("tapass-import %s\n", version.String())
		return
	}

	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input.xml|input.kdbx> <output.tap>\n", os.Args[0])
		os.Exit(1)
	}

	inputPath := os.Args[1]
	tapPath := os.Args[2]

	if _, err := os.Stat(inputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot access %s: %v\n", inputPath, err)
		os.Exit(1)
	}

	var stats *importer.ImportStats
	var err error

	if importer.IsKDBX(inputPath) {
		stats, err = importFromKDBX(inputPath, tapPath)
	} else {
		stats, err = importFromXML(inputPath, tapPath)
	}

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

func importFromXML(xmlPath, tapPath string) (*importer.ImportStats, error) {
	password, err := readPassword("Enter master password: ")
	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}

	confirm, err := readPassword("Confirm master password: ")
	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}

	if string(password) != string(confirm) {
		return nil, fmt.Errorf("passwords do not match")
	}

	return importer.Import(xmlPath, tapPath, string(password))
}

func importFromKDBX(kdbxPath, tapPath string) (*importer.ImportStats, error) {
	kdbxPw, err := readPassword("Enter KeePass password: ")
	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}

	tapPw, err := readPassword("Enter new tapass password: ")
	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}

	confirm, err := readPassword("Confirm tapass password: ")
	if err != nil {
		return nil, fmt.Errorf("read password: %w", err)
	}

	if string(tapPw) != string(confirm) {
		return nil, fmt.Errorf("passwords do not match")
	}

	return importer.ImportKDBX(kdbxPath, tapPath, string(kdbxPw), string(tapPw))
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
