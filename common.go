package logparser

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"
)

// Item as the parser definition
type Item interface {
	Data() string
	Header() []string
	Format() []string

	Stamp() time.Time
	Class() string

	Level() ItemLevel
}

// ItemLevel as the enum def
type ItemLevel int8

// enum value
const (
	LevelNone ItemLevel = iota
	LevelDbg
	LevelInfo
	LevelWarn
	LevelErr
)

// Str as the pretty serialzation
func (il ItemLevel) Str() string {
	switch il {
	case LevelNone:
		return ""
	case LevelDbg:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelErr:
		return "error"
	default:
	}
	return "unknown"
}

// --------------- unknown item def ----------------- //

// ClassUnknown as the Item.Class() return type
const ClassUnknown = "unknown"

var _ Item = (*UnknownItem)(nil)

// UnknownItem as the unclassified item data holder
type UnknownItem struct {
	data string
}

func NewUnknownItem(d string) Item {
	return &UnknownItem{
		data: d,
	}
}

func (i *UnknownItem) Data() string {
	return i.data
}

func (i *UnknownItem) Header() []string {
	return []string{"data"}
}

func (i *UnknownItem) Format() []string {
	return []string{i.data}
}

func (i *UnknownItem) Stamp() time.Time {
	return time.Time{}
}

func (i *UnknownItem) Class() string {
	return ClassUnknown
}

func (i UnknownItem) Level() ItemLevel {
	return LevelNone
}

// --------------- parser ---------------- //

type ParseFunc = func(int, string) (Item, error)

var prefixClassifier = map[string]ParseFunc{}

func RegisterPrefixClassifier(prefix string, parser ParseFunc) error {
	if _, ok := prefixClassifier[prefix]; ok {
		return fmt.Errorf("prefix %s already taken", prefix)
	}
	prefixClassifier[prefix] = parser
	return nil
}

// -------------- filter --------------- //

type FilterFunc = func(Item) bool

var itemFilters = []FilterFunc{}

func RegisterItemFilter(filter FilterFunc) {
	itemFilters = append(itemFilters, filter)
}

// -------------- log parsing ---------------- //

type ParseResult = map[string][]Item

func ParseByLine(path string) (ParseResult, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0

	parsedLogs := ParseResult{}
	for scanner.Scan() {
		lineCount++
		lineText := scanner.Text()

		hit := false
		var logItem Item
		for prefix, parser := range prefixClassifier {
			if strings.HasPrefix(lineText, prefix) {
				hit = true

				logItem, err = parser(lineCount, lineText)
				if err != nil {
					return nil, lineCount, err
				}
				break
			}
		}
		if !hit {
			logItem = NewUnknownItem(lineText)
		}

		pass := true
		for _, f := range itemFilters {
			if f(logItem) {
				pass = false
				break
			}
		}

		if pass {
			holder, ok := parsedLogs[logItem.Class()]
			if !ok {
				holder = make([]Item, 0)
			}
			parsedLogs[logItem.Class()] = append(holder, logItem)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	return parsedLogs, lineCount, nil
}

// ----------------- utility ---------------- //

func SaveAsCSV(path string, content []Item) error {
	if info, err := os.Stat(path); !os.IsNotExist(err) {
		if !info.IsDir() {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	text := make([][]string, 0)
	text = append(text, content[0].Header())
	for _, item := range content {
		text = append(text, item.Format())
	}

	out := csv.NewWriter(file)
	out.WriteAll(text)
	if err := out.Error(); err != nil {
		return err
	}

	return nil
}
