package secrets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
	if _, err := os.Stat(path); err == nil {
		return false, nil
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
	for _, line := range strings.Split(string(contents), "\n") {
		match := exportLine.FindStringSubmatch(line)
		if len(match) != 3 {
			continue
		}
		values[match[1]] = parseValue(match[2])
	}
	return values, nil
}

func Prompt(home string, fields []Field) error {
	values, err := Read(home)
	if err != nil {
		return err
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
	if err := ui.Inputs("Secrets", inputs); err != nil {
		return err
	}
	for _, entry := range entries {
		values[entry.key] = *entry.value
	}
	return Update(home, values)
}

func Update(home string, updates map[string]string) error {
	path := Path(home)
	contents, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	lines := []string{}
	if len(contents) > 0 {
		lines = strings.Split(strings.TrimSuffix(string(contents), "\n"), "\n")
	}
	seen := map[string]bool{}
	for index, line := range lines {
		match := exportLine.FindStringSubmatch(line)
		if len(match) != 3 {
			continue
		}
		if value, ok := updates[match[1]]; ok {
			lines[index] = "export " + match[1] + "=" + shellQuote(value)
			seen[match[1]] = true
		}
	}
	for key, value := range updates {
		if !seen[key] {
			lines = append(lines, "export "+key+"="+shellQuote(value))
		}
	}
	return writeAtomic(path, []byte(strings.Join(lines, "\n")+"\n"))
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
	return "\"" + strings.ReplaceAll(strings.ReplaceAll(value, "\\", "\\\\"), "\"", "\\\"") + "\""
}

func parseValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		if unquoted, err := strconv.Unquote(value); err == nil {
			return unquoted
		}
	}
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}
	return value
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
