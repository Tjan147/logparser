package logparser

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	TMPrefixInfo = "I["
	TMPrefixErr  = "E["

	TMItemSep = "module="

	TMStampTrim = "IE["
	TMStampFmt  = "2006-01-02|15:04:05.000"
)

var (
	currentHeight = 0
	currentStamp  = time.Time{}
)

func SetCurrentHeight(h int) {
	currentHeight = h
}

func SetCurrentStamp(s time.Time) {
	currentStamp = s
}

// ------------- register -------------- //

func RegisterTMPrefix() {
	if err := RegisterPrefixClassifier(TMPrefixErr, ParseTMErr); err != nil {
		panic(err)
	}
	if err := RegisterPrefixClassifier(TMPrefixInfo, ParseTMInfo); err != nil {
		panic(err)
	}
}

func splitTMItem(lineText string) (time.Time, string, string, error) {
	parts := strings.Split(lineText, TMItemSep)
	if len(parts) != 2 {
		return time.Time{}, "", "", fmt.Errorf("malformed item: %s", lineText)
	}

	// parse head
	headParts := strings.Split(parts[0], "]")
	if len(parts) != 2 {
		return time.Time{}, "", "", fmt.Errorf("malformed head: %s", parts[0])
	}

	stamp, err := parseTMStamp(headParts[0])
	if err != nil {
		return time.Time{}, "", "", fmt.Errorf("error parse timestamp(%s): %s", headParts[0], err.Error())
	}
	name := strings.TrimSpace(headParts[1])

	return stamp, name, parts[1], nil
}

func parseTMStamp(lineHead string) (time.Time, error) {
	return time.Parse(TMStampFmt, strings.TrimLeft(lineHead, TMStampTrim))
}

// ------------- error item -------------- //

var _ Item = (*TMItemErr)(nil)

type TMItemErr struct {
	line   int
	stamp  time.Time
	height int
	name   string
	module string
	info   string
}

func NewTMItemErr(t time.Time, l, h int, n, m, i string) Item {
	return &TMItemErr{
		line:   l,
		stamp:  t,
		height: h,
		name:   n,
		module: m,
		info:   i,
	}
}

func (e *TMItemErr) Data() string {
	return fmt.Sprintf("E[%s] %-32s module=%s err=\"%s\"", e.stamp.Format(TMStampFmt), e.name, e.module, e.info)
}

func (e TMItemErr) Header() []string {
	return []string{"line", "height", "head", "module", "detail"}
}

func (e *TMItemErr) Format() []string {
	return []string{strconv.Itoa(e.line), strconv.Itoa(e.height), e.name, e.module, e.info}
}

func (e *TMItemErr) Stamp() time.Time {
	return e.stamp
}

func (e TMItemErr) Class() string {
	return "tmErr"
}

func (e TMItemErr) Level() ItemLevel {
	return LevelErr
}

func ParseTMErr(lineNum int, lineText string) (Item, error) {
	stamp, name, tail, err := splitTMItem(lineText)
	if err != nil {
		return nil, fmt.Errorf("[%d] error split err log: %s", lineNum, err.Error())
	}

	// parse tail
	tailParts := strings.Split(tail, "err=")
	if len(tailParts) < 2 {
		return nil, fmt.Errorf("[%d] malformed err log tail: %s", lineNum, tail)
	}

	module := strings.TrimSpace(tailParts[0])
	detail := strings.TrimSuffix(strings.TrimPrefix(tailParts[1], "\""), "\"")

	return NewTMItemErr(stamp, lineNum, currentHeight, name, module, detail), nil
}

// ------------- info item ---------------- //

// ApplyItem
const itemNameApply = "Executed block"

var _ Item = (*TMInfoApply)(nil)

type TMInfoApply struct {
	height       int
	validTxNum   int
	invalidTxNum int
	stamp        time.Time
	cost         time.Duration
}

func NewTmInfoApply(h, vtxn, itxn int, s time.Time, c time.Duration) Item {
	return &TMInfoApply{
		height:       h,
		validTxNum:   vtxn,
		invalidTxNum: itxn,
		stamp:        s,
		cost:         c,
	}
}

func (i *TMInfoApply) Data() string {
	return fmt.Sprintf("I[%s] %-32s module=state height=%d", i.stamp.Format(TMStampFmt), itemNameApply, i.height)
}

func (i TMInfoApply) Header() []string {
	return []string{"height", "stamp", "cost"}
}

func (i *TMInfoApply) Format() []string {
	asMS := strconv.FormatInt(i.cost.Milliseconds(), 10)
	return []string{strconv.Itoa(i.height), i.stamp.Format(time.RFC3339), asMS}
}

func (i *TMInfoApply) Stamp() time.Time {
	return i.stamp
}

func (i TMInfoApply) Class() string {
	return "tmApply"
}

func (i TMInfoApply) Level() ItemLevel {
	return LevelInfo
}

// CommitItem
const itemNameCommit = "Committed state"

var _ Item = (*TMInfoCommit)(nil)

type TMInfoCommit struct {
	height  int
	txNum   int
	appHash string
	stamp   time.Time
}

func NewTMInfoCommit(h, tn int, hash string, s time.Time) Item {
	return &TMInfoCommit{
		height:  h,
		txNum:   tn,
		appHash: hash,
		stamp:   s,
	}
}

func (i *TMInfoCommit) Data() string {
	return fmt.Sprintf("I[%s] %-32s module=state height=%d txs=%d hash=%s", i.stamp.Format(TMStampFmt), itemNameCommit, i.height, i.txNum, i.appHash)
}

func (i TMInfoCommit) Header() []string {
	return []string{"height", "stamp", "txs", "hash"}
}

func (i *TMInfoCommit) Format() []string {
	return []string{strconv.Itoa(i.height), i.stamp.Format(time.RFC3339), strconv.Itoa(i.txNum), i.appHash}
}

func (i *TMInfoCommit) Stamp() time.Time {
	return i.stamp
}

func (i TMInfoCommit) Class() string {
	return "tmCommit"
}

func (i TMInfoCommit) Level() ItemLevel {
	return LevelInfo
}

// EndBlockerItem
const itemNameEndBlocker = "EndBlocker Time"

var _ Item = (*TMInfoEndBlocker)(nil)

type TMInfoEndBlocker struct {
	stamp  time.Time
	height int
	module string
	cost   time.Duration
}

func NewTMInfoEndBlocker(s time.Time, h int, m string, c time.Duration) Item {
	return &TMInfoEndBlocker{
		stamp:  s,
		height: h,
		module: m,
		cost:   c,
	}
}

func (i *TMInfoEndBlocker) Data() string {
	asMS := strconv.FormatInt(i.cost.Milliseconds(), 10)
	return fmt.Sprintf("I[%s] %-32s module=main height=%d name=%s cost=%s", i.stamp.Format(TMStampFmt), itemNameEndBlocker, i.height, i.module, asMS)
}

func (i TMInfoEndBlocker) Header() []string {
	return []string{"height", "module", "cost"}
}

func (i *TMInfoEndBlocker) Format() []string {
	asMS := strconv.FormatInt(i.cost.Milliseconds(), 10)
	return []string{strconv.Itoa(i.height), i.module, asMS}
}

func (i *TMInfoEndBlocker) Stamp() time.Time {
	return i.stamp
}

func (i TMInfoEndBlocker) Class() string {
	return "tmEndBlocker"
}

func (i TMInfoEndBlocker) Level() ItemLevel {
	return LevelInfo
}

// HandlerItem
const itemNameHandler = "Deliver Time"

var _ Item = (*TMInfoHandler)(nil)

type TMInfoHandler struct {
	stamp  time.Time
	height int
	txType string
	cost   time.Duration
}

func NewTMInfoHandler(s time.Time, h int, t string, c time.Duration) Item {
	return &TMInfoHandler{
		stamp:  s,
		height: h,
		txType: t,
		cost:   c,
	}
}

func (i *TMInfoHandler) Data() string {
	asMS := strconv.FormatInt(i.cost.Milliseconds(), 10)
	return fmt.Sprintf("I[%s] %-32s module=main height=%d name=%s cost=%s", i.stamp.Format(TMStampFmt), itemNameHandler, i.height, i.txType, asMS)
}

func (i TMInfoHandler) Header() []string {
	return []string{"height", "type", "cost"}
}

func (i *TMInfoHandler) Format() []string {
	asMS := strconv.FormatInt(i.cost.Milliseconds(), 10)
	return []string{strconv.Itoa(i.height), i.txType, asMS}
}

func (i *TMInfoHandler) Stamp() time.Time {
	return i.stamp
}

func (i TMInfoHandler) Class() string {
	return "tmHandler"
}

func (i TMInfoHandler) Level() ItemLevel {
	return LevelInfo
}

// QuerierItem
const itemNameQuerier = "Query Time"

var _ Item = (*TMInfoQuerier)(nil)

type TMInfoQuerier struct {
	stamp  time.Time
	height int
	path   string
	cost   time.Duration
}

func NewTMInfoQuerier(s time.Time, h int, p string, c time.Duration) Item {
	return &TMInfoQuerier{
		stamp:  s,
		height: h,
		path:   p,
		cost:   c,
	}
}

func (i *TMInfoQuerier) Data() string {
	asMS := strconv.FormatInt(i.cost.Milliseconds(), 10)
	return fmt.Sprintf("I[%s] %-32s module=main path=[%s] cost=%s", i.stamp.Format(TMStampFmt), itemNameQuerier, i.path, asMS)
}

func (i TMInfoQuerier) Header() []string {
	return []string{"height", "path", "cost"}
}

func (i *TMInfoQuerier) Format() []string {
	asMS := strconv.FormatInt(i.cost.Milliseconds(), 10)
	return []string{strconv.Itoa(i.height), i.path, asMS}
}

func (i *TMInfoQuerier) Stamp() time.Time {
	return i.stamp
}

func (i TMInfoQuerier) Class() string {
	return "tmQuerier"
}

func (i TMInfoQuerier) Level() ItemLevel {
	return LevelInfo
}

// ------------------------- //

func ParseTMInfo(lineNum int, lineText string) (Item, error) {
	parts := strings.Split(lineText, TMItemSep)

	// parse head
	headParts := strings.Split(parts[0], "]")

	stamp, err := parseTMStamp(headParts[0])
	if err != nil {
		return nil, fmt.Errorf("%d: %s", lineNum, err.Error())
	}
	name := strings.TrimSpace(headParts[1])

	switch name {

	}
}
