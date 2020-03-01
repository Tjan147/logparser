package logparser

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	BSPrefix = "direct_"

	BSLogItemCount = 6
)

func RegisterBSPrefix() {
	if err := RegisterPrefixClassifier(BSPrefix, ParseBenchStoreItem); err != nil {
		panic(err)
	}
}

var _ Item = (*benchStoreItem)(nil)

type benchStoreItem struct {
	backendType   string
	method        string
	existingCount int
	count         int
	cost          time.Duration
}

func ParseBenchStoreItem(lineNum int, lineText string) (Item, error) {
	parts := strings.Split(lineText, ",")

	if len(parts) != BSLogItemCount {
		return nil, fmt.Errorf("[%d] malformed item: %s", lineNum, lineText)
	}

	bType := strings.TrimSpace(parts[0])
	bMethod := strings.TrimSpace(parts[1])

	bExisting, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return nil, fmt.Errorf("[%d] error parse existing(%s): %s", lineNum, parts[2], err.Error())
	}

	bCount, err := strconv.Atoi(strings.TrimSpace(parts[4]))
	if err != nil {
		return nil, fmt.Errorf("[%d] error parse count(%s): %s", lineNum, parts[4], err.Error())
	}

	bCost, err := time.ParseDuration(strings.TrimSpace(parts[5]))
	if err != nil {
		return nil, fmt.Errorf("[%d] error parse cost(%s): %s", lineNum, parts[55555], err.Error())
	}

	return &benchStoreItem{
		backendType:   bType,
		method:        bMethod,
		existingCount: bExisting,
		count:         bCount,
		cost:          bCost,
	}, nil
}

func (i *benchStoreItem) Data() string {
	return fmt.Sprintf("%s, %s, %d, %s, %d, %dms", i.backendType, i.method, i.existingCount, i.method, i.count, i.cost.Milliseconds())
}

func (i benchStoreItem) Header() []string {
	return []string{"backend", "method", "existing", "count", "cost"}
}

func (i *benchStoreItem) Format() []string {
	return []string{i.backendType, i.method, strconv.Itoa(i.existingCount), strconv.Itoa(i.count), strconv.FormatInt(i.cost.Milliseconds(), 10)}
}

func (i benchStoreItem) Stamp() time.Time {
	return time.Time{}
}

func (i *benchStoreItem) Class() string {
	return i.method
}

func (i benchStoreItem) Level() ItemLevel {
	return LevelInfo
}
