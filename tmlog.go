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
	currentHeight      = 0
	currentHeightStamp = time.Time{}
)

func SetCurrentHeight(h int) {
	currentHeight = h
}

func SetCurrentHeightStamp(s time.Time) {
	currentHeightStamp = s
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
}

func NewTmInfoApply(h, vtxn, itxn int, s time.Time) Item {
	return &TMInfoApply{
		height:       h,
		validTxNum:   vtxn,
		invalidTxNum: itxn,
		stamp:        s,
	}
}

// form like `NAME=INT_VALUE`
func fetchIntFromPair(pair string) (int, error) {
	parts := strings.Split(pair, "=")
	if len(parts) != 2 {
		return -1, fmt.Errorf("malformed pair: %s", pair)
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil {
		return -1, fmt.Errorf("error parse int: %s", err.Error())
	}
	return n, nil
}

func parseTailApply(stamp time.Time, tail string) (Item, error) {
	parts := strings.Split(tail, " ")
	if len(parts) != 4 {
		return nil, fmt.Errorf("malformed apply tail: %s", tail)
	}

	h, err := fetchIntFromPair(parts[1])
	if err != nil {
		return nil, fmt.Errorf("error parse apply height: %s", err.Error())
	}

	vtxs, err := fetchIntFromPair(parts[2])
	if err != nil {
		return nil, fmt.Errorf("error parse apply validTxs: %s", err.Error())
	}

	itxs, err := fetchIntFromPair(parts[3])
	if err != nil {
		return nil, fmt.Errorf("error parse apply invalidTxs: %s", err.Error())
	}

	return NewTmInfoApply(h, vtxs, itxs, stamp), nil
}

func (i *TMInfoApply) Data() string {
	return fmt.Sprintf("I[%s] %-32s module=state height=%d", i.stamp.Format(TMStampFmt), itemNameApply, i.height)
}

func (i TMInfoApply) Header() []string {
	return []string{"height", "stamp"}
}

func (i *TMInfoApply) Format() []string {
	return []string{strconv.Itoa(i.height), i.stamp.Format(time.RFC3339)}
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

func parseTailCommit(stamp time.Time, tail string) (Item, error) {
	parts := strings.Split(tail, " ")
	if len(parts) != 4 {
		return nil, fmt.Errorf("malformed commit tail: %s", tail)
	}

	h, err := fetchIntFromPair(parts[1])
	if err != nil {
		return nil, fmt.Errorf("error parse commit height: %s", err.Error())
	}

	txs, err := fetchIntFromPair(parts[2])
	if err != nil {
		return nil, fmt.Errorf("error parse commit txs: %s", err.Error())
	}

	hashParts := strings.Split(parts[3], "=")
	if len(hashParts) != 2 {
		return nil, fmt.Errorf("malformed commit appHash: %s", parts[3])
	}

	// update current height info
	SetCurrentHeight(h)
	SetCurrentHeightStamp(stamp)

	return NewTMInfoCommit(h, txs, hashParts[1], stamp), nil
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

func fetchDurationFromPair(pair string) (time.Duration, error) {
	parts := strings.Split(pair, "=")
	if len(parts) != 2 {
		return 0, fmt.Errorf("malformed pair: %s", pair)
	}
	dur, err := time.ParseDuration(parts[1])
	if err != nil {
		return 0, fmt.Errorf("error parse duration: %s", err.Error())
	}
	return dur, nil
}

func parseTailEndBlocker(stamp time.Time, tail string) (Item, error) {
	parts := strings.Split(tail, " ")
	if len(parts) != 4 {
		return nil, fmt.Errorf("malformed endblocker tail: %s", tail)
	}

	h, err := fetchIntFromPair(parts[1])
	if err != nil {
		return nil, fmt.Errorf("error parse endblocker height: %s", err.Error())
	}

	nameParts := strings.Split(parts[2], "=")
	if len(nameParts) != 2 {
		return nil, fmt.Errorf("malformed endblocker name: %s", parts[2])
	}

	c, err := fetchDurationFromPair(parts[3])
	if err != nil {
		return nil, fmt.Errorf("error parse endblocker cost: %s", err.Error())
	}

	return NewTMInfoEndBlocker(stamp, h, nameParts[1], c), nil
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

func parseTailHandler(stamp time.Time, tail string) (Item, error) {
	parts := strings.Split(tail, " ")
	if len(parts) != 4 {
		return nil, fmt.Errorf("malformed handler tail: %s", tail)
	}

	h, err := fetchIntFromPair(parts[1])
	if err != nil {
		return nil, fmt.Errorf("error parse handler height: %s", err.Error())
	}

	typeParts := strings.Split(parts[2], "=")
	if len(typeParts) != 2 {
		return nil, fmt.Errorf("malformed handler type: %s", parts[2])
	}

	c, err := fetchDurationFromPair(parts[3])
	if err != nil {
		return nil, fmt.Errorf("error parse handler cost: %s", err.Error())
	}

	return NewTMInfoHandler(stamp, h, typeParts[1], c), nil
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

func parseTailQuerier(stamp time.Time, tail string) (Item, error) {
	parts := strings.Split(tail, " ")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed querier tail: %s", tail)
	}

	pathParts := strings.Split(parts[1], "=")
	if len(pathParts) != 2 {
		return nil, fmt.Errorf("malformed querier path: %s", parts[1])
	}
	path := strings.TrimRight(strings.TrimLeft(pathParts[1], "["), "]")

	c, err := fetchDurationFromPair(parts[2])
	if err != nil {
		return nil, fmt.Errorf("error parse querier time: %s", err.Error())
	}

	return NewTMInfoQuerier(stamp, currentHeight, path, c), nil
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

// UnknownItem
var _ Item = (*TMInfoIgnore)(nil)

type TMInfoIgnore struct {
	stamp  time.Time
	height int
	head   string
	tail   string
}

func NewTMInfoIgnore(s time.Time, h int, head, tail string) Item {
	return &TMInfoIgnore{
		stamp:  s,
		height: h,
		head:   head,
		tail:   tail,
	}
}

func (i *TMInfoIgnore) Data() string {
	return fmt.Sprintf("I[%s] %-32s module=%s", i.stamp.Format(TMStampFmt), i.head, i.tail)
}

func (i TMInfoIgnore) Header() []string {
	return []string{"height", "head", "tail"}
}

func (i *TMInfoIgnore) Format() []string {
	return []string{strconv.Itoa(i.height), i.head, i.tail}
}

func (i *TMInfoIgnore) Stamp() time.Time {
	return i.stamp
}

func (i TMInfoIgnore) Class() string {
	return "tmIgnore"
}

func (i TMInfoIgnore) Level() ItemLevel {
	return LevelInfo
}

// ------------------------- //

type parseTailFunc = func(time.Time, string) (Item, error)

func ParseTMInfo(lineNum int, lineText string) (Item, error) {
	parts := strings.Split(lineText, TMItemSep)
	if len(parts) != 2 {
		return nil, fmt.Errorf("%d: malformed tm item: %s", lineNum, lineText)
	}

	// parse head
	headParts := strings.Split(parts[0], "]")

	stamp, err := parseTMStamp(headParts[0])
	if err != nil {
		return nil, fmt.Errorf("%d: error parse stamp: %s", lineNum, err.Error())
	}

	var tailParser parseTailFunc
	name := strings.TrimSpace(headParts[1])
	switch name {
	case itemNameApply:
		tailParser = parseTailApply
	case itemNameCommit:
		tailParser = parseTailCommit
	case itemNameEndBlocker:
		tailParser = parseTailEndBlocker
	case itemNameHandler:
		tailParser = parseTailHandler
	case itemNameQuerier:
		tailParser = parseTailQuerier
	default:
	}

	var ret Item
	if tailParser != nil {
		if ret, err = tailParser(stamp, parts[1]); err != nil {
			return nil, fmt.Errorf("%d: error parse tail: %s", lineNum, err.Error())
		}
	} else {
		ret = NewTMInfoIgnore(stamp, currentHeight, name, parts[1])
	}

	return ret, nil
}
