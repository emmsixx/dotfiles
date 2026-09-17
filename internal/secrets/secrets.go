package secrets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/emmsixx/dotfiles/internal/ui"
)

type Field struct {
	Key    string
	Secret bool
}

var DefaultFields = []Field{
	{Key: "GH_WORK_USER"},
	{Key: "GH_PERSONAL_USER"},
	{Key: "OPENROUTER_API_KEY", Secret: true},
	{Key: "FIRECRAWL_API_KEY", Secret: true},
	{Key: "PHOBOS_TOKEN", Secret: true},
}

var exportLine = regexp.MustCompile(`^\s*export\s+([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

func Path(home string) string { return filepath.Join(home, ".secrets") }

func Ensure(home, template string) (bool, error) {
	path := Path(home)
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("%s must be a regular private file", path)
		}
		return false, os.Chmod(path, 0o600)
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	contents, err := os.ReadFile(template)
	if err != nil {
		return false, fmt.Errorf("read secrets template: %w", err)
	}
	if err := writeAtomic(path, contents); err != nil {
		return false, err
	}
	return true, nil
}

func Read(home string) (map[string]string, error) {
	contents, err := os.ReadFile(Path(home))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	values := map[string]string{}
	for _, record := range exportRecords(string(contents)) {
		if record.key != "" && record.literal {
			values[record.key] = record.value
		}
	}
	return values, nil
}

func Prompt(home string, fields []Field) error {
	return prompt(home, fields, ui.Inputs)
}

func prompt(home string, fields []Field, input func(string, []ui.Input) error) error {
	values, err := Read(home)
	if err != nil {
		return err
	}
	contents, err := os.ReadFile(Path(home))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, record := range exportRecords(string(contents)) {
		for _, field := range fields {
			if record.key == field.Key && !record.literal {
				return fmt.Errorf("%s uses shell expressions; edit it manually before using the secrets editor", field.Key)
			}
		}
	}
	inputs := make([]ui.Input, 0, len(fields))
	entries := make([]struct {
		key   string
		value *string
	}, 0, len(fields))
	for _, field := range fields {
		value := values[field.Key]
		valuePtr := new(string)
		*valuePtr = value
		inputs = append(inputs, ui.Input{Title: field.Key, Value: valuePtr, Secret: field.Secret})
		entries = append(entries, struct {
			key   string
			value *string
		}{key: field.Key, value: valuePtr})
	}
	if err := input("Secrets", inputs); err != nil {
		return err
	}
	updates := map[string]string{}
	for _, entry := range entries {
		updates[entry.key] = *entry.value
	}
	return Update(home, updates)
}

func Update(home string, updates map[string]string) error {
	for key, value := range updates {
		if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(key) || strings.ContainsRune(value, 0) {
			return fmt.Errorf("invalid shell variable or NUL value for %q", key)
		}
	}
	path := Path(home)
	contents, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var output strings.Builder
	seen := map[string]bool{}
	for _, record := range exportRecords(string(contents)) {
		if value, ok := updates[record.key]; ok {
			output.WriteString("export " + record.key + "=" + shellQuote(value) + "\n")
			seen[record.key] = true
		} else {
			output.WriteString(record.raw)
		}
	}
	keys := make([]string, 0, len(updates))
	for key := range updates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !seen[key] {
			if output.Len() > 0 && !strings.HasSuffix(output.String(), "\n") {
				output.WriteByte('\n')
			}
			output.WriteString("export " + key + "=" + shellQuote(updates[key]) + "\n")
		}
	}
	return writeAtomic(path, []byte(output.String()))
}

func Status(home string, fields []Field) ([]string, error) {
	values, err := Read(home)
	if err != nil {
		return nil, err
	}
	status := make([]string, 0, len(fields))
	for _, field := range fields {
		value := values[field.Key]
		if value == "" {
			status = append(status, field.Key+": not set")
			continue
		}
		if field.Secret {
			status = append(status, field.Key+": "+mask(value))
		} else {
			status = append(status, field.Key+": set")
		}
	}
	return status, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

type exportRecord struct {
	raw, key, value string
	literal         bool
}

func exportRecords(contents string) []exportRecord {
	lines := strings.SplitAfter(contents, "\n")
	var records []exportRecord
	for i := 0; i < len(lines); i++ {
		raw := lines[i]
		match := exportLine.FindStringSubmatch(strings.TrimSuffix(raw, "\n"))
		if len(match) != 3 {
			records = append(records, exportRecord{raw: raw})
			continue
		}
		encoded := match[2]
		value, literal, incomplete := parseLiteral(encoded)
		for incomplete && i+1 < len(lines) {
			i++
			raw += lines[i]
			encoded += "\n" + strings.TrimSuffix(lines[i], "\n")
			value, literal, incomplete = parseLiteral(encoded)
		}
		records = append(records, exportRecord{raw: raw, key: match[1], value: value, literal: literal && !incomplete})
	}
	return records
}

// Parse only literal shell words. Never evaluate expansions or commands.
func parseLiteral(encoded string) (string, bool, bool) {
	var out strings.Builder
	var quote byte
	for i := 0; i < len(encoded); i++ {
		c := encoded[i]
		if quote == '\'' {
			if c == '\'' {
				quote = 0
			} else {
				out.WriteByte(c)
			}
			continue
		}
		if c == '\\' {
			if i+1 == len(encoded) {
				return "", false, true
			}
			next := encoded[i+1]
			if quote == '"' && !strings.ContainsRune("$`\"\\\n", rune(next)) {
				out.WriteByte(c)
				continue
			}
			i++
			if next != '\n' {
				out.WriteByte(next)
			}
			continue
		}
		if c == '$' || c == '`' {
			return "", false, false
		}
		if quote == '"' {
			if c == '"' {
				quote = 0
			} else {
				out.WriteByte(c)
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		if c == ' ' || c == '\t' || c == '\r' {
			rest := strings.TrimSpace(encoded[i:])
			return out.String(), rest == "" || strings.HasPrefix(rest, "#"), false
		}
		if strings.ContainsRune(";|&<>()", rune(c)) {
			return "", false, false
		}
		out.WriteByte(c)
	}
	return out.String(), quote == 0, quote != 0
}

func mask(value string) string {
	if len(value) <= 4 {
		return "••••"
	}
	return value[:2] + "••••" + value[len(value)-2:]
}

func writeAtomic(path string, contents []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".secrets.*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(contents); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
