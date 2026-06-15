package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"golang.org/x/term"
	"github.com/LaoQi/tapass/tools/vault"
	"github.com/LaoQi/tapass/tools/version"
)

var out io.Writer = os.Stderr

var commands = []string{
	"create", "open", "set", "get", "delete", "list", "raw",
	"passwd", "compact", "help", "quit", "exit",
}

type terminal struct {
	history []string
	histIdx int
	vault   *vault.Vault
	path    string
}

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("tapass-cli %s\n", version.String())
		return
	}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: tapass-cli <vault-file>\r\n")
		os.Exit(1)
	}

	t := &terminal{path: os.Args[1]}

	if _, err := os.Stat(t.path); os.IsNotExist(err) {
		fmt.Fprintf(out, "File not found: %s\r\n", t.path)
		fmt.Fprintln(out, "Use 'create' command to create a new vault.")
	} else {
		password := readPassword("Enter password: ")
		data, err := os.ReadFile(t.path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\r\n", err)
			os.Exit(1)
		}
		v, err := vault.Open(data, password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\r\n", err)
			os.Exit(1)
		}
		t.vault = v
		fmt.Fprintf(out, "Vault opened: %s\r\n\r\n", t.path)
	}

	t.run()
}

func atomicWriteFile(path string, data []byte) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename tmp file: %w", err)
	}
	return nil
}

func (t *terminal) saveVault() error {
	data, err := t.vault.MarshalBinary()
	if err != nil {
		return err
	}
	return atomicWriteFile(t.path, data)
}

func (t *terminal) run() {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "terminal error: %v\r\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	reader := bufio.NewReader(os.Stdin)
	lineBuf := ""

	for {
		fmt.Fprintf(out, "\rtapass> %s\033[K", lineBuf)

		b, err := reader.ReadByte()
		if err != nil {
			break
		}

		switch {
		case b == 3:
			lineBuf = ""
			fmt.Fprint(out, "\r\n")
		case b == 4:
			if lineBuf == "" {
				fmt.Fprint(out, "\r\n")
				return
			}
			lineBuf = deleteLastRune(lineBuf)
		case b == 127 || b == 8:
			lineBuf = deleteLastRune(lineBuf)
		case b == '\r' || b == '\n':
			fmt.Fprint(out, "\r\n")
			if strings.TrimSpace(lineBuf) != "" {
				t.history = append(t.history, lineBuf)
				t.histIdx = len(t.history)
				t.execute(lineBuf)
			}
			lineBuf = ""
		case b == '\t':
			lineBuf = t.complete(lineBuf)
		case b == 27:
			seq := make([]byte, 2)
			if _, err := reader.Read(seq); err == nil {
				if seq[0] == '[' {
					switch seq[1] {
					case 'A':
						if t.histIdx > 0 {
							t.histIdx--
							lineBuf = t.history[t.histIdx]
						}
					case 'B':
						if t.histIdx < len(t.history) {
							t.histIdx++
							if t.histIdx < len(t.history) {
								lineBuf = t.history[t.histIdx]
							} else {
								lineBuf = ""
							}
						}
					}
				}
			}
		case b >= 32 && b < 127:
			lineBuf += string(b)
		case b >= 0xC0:
			seq := []byte{b}
			needed := utf8SeqLen(b)
			for i := 0; i < needed; i++ {
				nb, err := reader.ReadByte()
				if err != nil || nb < 0x80 || nb > 0xBF {
					break
				}
				seq = append(seq, nb)
			}
			if len(seq) == 1+needed {
				lineBuf += string(seq)
			}
		}
	}
}

func (t *terminal) complete(line string) string {
	words := strings.Fields(line)
	trailingSpace := len(line) > 0 && line[len(line)-1] == ' '

	if len(words) == 0 || (len(words) == 1 && !trailingSpace) {
		prefix := ""
		if len(words) == 1 {
			prefix = words[0]
		}
		candidates := filterPrefix(commands, prefix)
		if len(candidates) == 1 {
			return candidates[0] + " "
		}
		if len(candidates) > 1 {
			common := longestCommonPrefix(candidates)
			if common != prefix {
				return common
			}
			fmt.Fprint(out, "\r\n")
			for _, c := range candidates {
				fmt.Fprintf(out, "%s  ", c)
			}
			fmt.Fprint(out, "\r\n")
		}
		return line
	}

	cmd := words[0]
	if (cmd == "get" || cmd == "delete" || cmd == "set") && t.vault != nil {
		if (len(words) == 1 && trailingSpace) || (len(words) == 2 && !trailingSpace) {
			prefix := ""
			if len(words) == 2 {
				prefix = words[1]
			}
			keys := vaultKeys(t.vault)
			candidates := filterPrefix(keys, prefix)
			if len(candidates) == 1 {
				result := strings.Join(words[:len(words)-1], " ")
				if result != "" {
					result += " "
				}
				return result + candidates[0] + " "
			}
			if len(candidates) > 1 {
				common := longestCommonPrefix(candidates)
				if common != prefix {
					result := strings.Join(words[:len(words)-1], " ")
					if result != "" {
						result += " "
					}
					return result + common
				}
				fmt.Fprint(out, "\r\n")
				for _, c := range candidates {
					fmt.Fprintf(out, "%s  ", c)
				}
				fmt.Fprint(out, "\r\n")
			}
		}
	}

	return line
}

func filterPrefix(list []string, prefix string) []string {
	var result []string
	for _, s := range list {
		if strings.HasPrefix(s, prefix) {
			result = append(result, s)
		}
	}
	return result
}

func longestCommonPrefix(list []string) string {
	if len(list) == 0 {
		return ""
	}
	prefix := list[0]
	for _, s := range list[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}

func vaultKeys(v *vault.Vault) []string {
	entries := v.List()
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (t *terminal) execute(line string) {
	args := strings.Fields(line)
	if len(args) == 0 {
		return
	}

	cmd := args[0]

	switch cmd {
	case "create":
		t.cmdCreate()
	case "open":
		t.cmdOpen()
	case "set":
		t.cmdSet(args)
	case "get":
		t.cmdGet(args)
	case "delete":
		t.cmdDelete(args)
	case "list":
		t.cmdList()
	case "raw":
		t.cmdRaw()
	case "passwd":
		t.cmdPasswd()
	case "compact":
		t.cmdCompact()
	case "help":
		t.cmdHelp()
	case "quit", "exit":
		fmt.Fprintln(out, "bye.")
		os.Exit(0)
	default:
		fmt.Fprintf(out, "unknown command: %s\r\n", cmd)
	}
}

func (t *terminal) cmdCreate() {
	password1 := readPassword("New password: ")
	password2 := readPassword("Confirm password: ")
	if password1 != password2 {
		fmt.Fprintln(out, "passwords do not match")
		return
	}

	data, err := vault.Create(password1)
	if err != nil {
		fmt.Fprintf(out, "error: %v\r\n", err)
		return
	}

	if err := atomicWriteFile(t.path, data); err != nil {
		fmt.Fprintf(out, "error: %v\r\n", err)
		return
	}

	v, err := vault.Open(data, password1)
	if err != nil {
		fmt.Fprintf(out, "error: %v\r\n", err)
		return
	}
	t.vault = v
	fmt.Fprintf(out, "vault created: %s\r\n", t.path)
}

func (t *terminal) cmdOpen() {
	fmt.Fprint(out, "File path: ")
	path := readLine()
	if path == "" {
		return
	}

	password := readPassword("Password: ")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(out, "error: %v\r\n", err)
		return
	}
	v, err := vault.Open(data, password)
	if err != nil {
		fmt.Fprintf(out, "error: %v\r\n", err)
		return
	}
	t.vault = v
	t.path = path
	fmt.Fprintf(out, "Vault opened: %s\r\n", path)
}

func (t *terminal) cmdSet(args []string) {
	if t.vault == nil {
		fmt.Fprintln(out, "no vault open")
		return
	}
	if len(args) < 3 {
		fmt.Fprintln(out, "usage: set <key> <value>")
		return
	}
	key := args[1]
	value := strings.Join(args[2:], " ")

	t.vault.Set(key, []byte(value))
	if err := t.saveVault(); err != nil {
		fmt.Fprintf(out, "error: %v\r\n", err)
		return
	}
	fmt.Fprintln(out, "ok")
}

func (t *terminal) cmdGet(args []string) {
	if t.vault == nil {
		fmt.Fprintln(out, "no vault open")
		return
	}
	if len(args) < 2 {
		fmt.Fprintln(out, "usage: get <key>")
		return
	}
	key := args[1]

	val, ok := t.vault.Get(key)
	if !ok {
		fmt.Fprintf(out, "key not found: %s\r\n", key)
		return
	}
	fmt.Fprintf(out, "%s\r\n", string(val))
}

func (t *terminal) cmdDelete(args []string) {
	if t.vault == nil {
		fmt.Fprintln(out, "no vault open")
		return
	}
	if len(args) < 2 {
		fmt.Fprintln(out, "usage: delete <key>")
		return
	}
	key := args[1]

	t.vault.Delete(key)
	if err := t.saveVault(); err != nil {
		fmt.Fprintf(out, "error: %v\r\n", err)
		return
	}
	fmt.Fprintf(out, "deleted: %s\r\n", key)
}

func (t *terminal) cmdList() {
	if t.vault == nil {
		fmt.Fprintln(out, "no vault open")
		return
	}

	entries := t.vault.List()
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
		fmt.Fprintf(out, "%s  %s  %s  %s\r\n", ts, typeName, k, string(e.Value))
	}
}

func (t *terminal) cmdRaw() {
	if t.vault == nil {
		fmt.Fprintln(out, "no vault open")
		return
	}

	for i, e := range t.vault.Entries {
		ts := time.UnixMilli(int64(e.Timestamp)).Format("2006-01-02 15:04:05.000")
		typeName := fmt.Sprintf("%s(%d)", typeToString(e.Type), e.Type)
		value := "-"
		if e.Type != vault.TypeClear {
			if e.Type == vault.TypeBlob {
				value = fmt.Sprintf("[blob %d bytes]", len(e.Value))
			} else {
				value = string(e.Value)
			}
		}
		fmt.Fprintf(out, "#%d  %s  %-10s  %s  %s\r\n", i+1, ts, typeName, e.Key, value)
	}
}

func typeToString(t uint8) string {
	switch t {
	case vault.TypeClear:
		return "clear"
	case vault.TypeText:
		return "text"
	case vault.TypeBlob:
		return "blob"
	default:
		return "unknown"
	}
}

func (t *terminal) cmdPasswd() {
	if t.vault == nil {
		fmt.Fprintln(out, "no vault open")
		return
	}

	oldPassword := readPassword("Old password: ")
	newPassword1 := readPassword("New password: ")
	newPassword2 := readPassword("Confirm new password: ")

	if newPassword1 != newPassword2 {
		fmt.Fprintln(out, "passwords do not match")
		return
	}

	data, err := t.vault.ChangePassword(oldPassword, newPassword1)
	if err != nil {
		fmt.Fprintf(out, "error: %v\r\n", err)
		return
	}

	if err := atomicWriteFile(t.path, data); err != nil {
		fmt.Fprintf(out, "error: %v\r\n", err)
		return
	}
	fmt.Fprintln(out, "password changed")
}

func (t *terminal) cmdCompact() {
	if t.vault == nil {
		fmt.Fprintln(out, "no vault open")
		return
	}

	t.vault.Compact()
	if err := t.saveVault(); err != nil {
		fmt.Fprintf(out, "error: %v\r\n", err)
		return
	}
	fmt.Fprintln(out, "vault compacted")
}

func (t *terminal) cmdHelp() {
	fmt.Fprintln(out, `Commands:
  create              Create a new vault
  open                Open a vault file
  set <key> <value>   Set an entry
  get <key>           Get an entry value
  delete <key>        Delete an entry
  list                List all entries
  raw                 Show all raw entries (including deleted)
  passwd              Change password
  compact             Compact vault (remove deleted entries)
  help                Show this help
  quit / exit         Exit`)
}

func deleteLastRune(s string) string {
	runes := []rune(s)
	if len(runes) > 0 {
		return string(runes[:len(runes)-1])
	}
	return s
}

func utf8SeqLen(b byte) int {
	switch {
	case b >= 0xF0:
		return 3
	case b >= 0xE0:
		return 2
	case b >= 0xC0:
		return 1
	default:
		return 0
	}
}

func readPassword(prompt string) string {
	fmt.Fprint(out, prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprint(out, "\r\n")
	if err != nil {
		return ""
	}
	return string(password)
}

func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}
