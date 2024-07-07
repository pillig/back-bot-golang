package loot

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"slices"
	"strconv"
	"time"

	"back-bot/backs/model"
)

// TODO: move to models
type UserID string

// TODO: move to models?
type UserLootState struct {
	Loot       map[model.Back]int
	Greenbacks int
}

// LootItem is the tuple of (Back, Count), representing a (k, v) pair from the Loot map
type LootItem struct {
	model.Back
	Count int
}

// LootByRarity partitions Loot by rarity, sorted by count
func (u UserLootState) LootByRarity() map[model.Rarity][]LootItem {
	out := make(map[model.Rarity][]LootItem)

	for back, count := range u.Loot {
		out[back.Rarity()] = append(out[back.Rarity()], LootItem{Back: back, Count: count})
	}
	for k, backs := range out {
		sortLootItemsByCount(backs)
		out[k] = backs
	}

	return out
}

func (u UserLootState) RarityPoints() int {
	var rarityPoints int
	for rarity, backs := range u.LootByRarity() {
		value := model.RarityLootValues[rarity]
		rarityPoints += value * len(backs)
	}
	return rarityPoints
}

// sort by path asc
// TODO: gotta be a better way
func sortLootItemsByPath(s []LootItem) {
	slices.SortFunc(s, func(a, b LootItem) int {
		switch true {
		case a.Path() < b.Path():
			return -1
		case a.Path() > b.Path():
			return 1
		default:
			return 0
		}
	})
}

// sort by count desc
// TODO: gotta be a better way
func sortLootItemsByCount(s []LootItem) {
	slices.SortFunc(s, func(a, b LootItem) int {
		switch true {
		case a.Count < b.Count:
			return 1
		case a.Count > b.Count:
			return -1
		default:
			return 0
		}
	})
}

func StateFromCSVRecord(record []string) (UserID, UserLootState, error) {
	state := UserLootState{
		Loot: make(map[model.Back]int),
	}

	// record guaranteed by caller to be len >= 2
	userID := UserID(record[0])

	if userID == "" {
		return "", state, fmt.Errorf("empty user ID in csv record: %v", record)
	}

	if greenbacks, err := strconv.Atoi(record[1]); err != nil {
		return "", state, fmt.Errorf("invalid greenbacks value in csv record. userID: %v greenbacks: %v err: %w", userID, record[1], err)
	} else {
		state.Greenbacks = greenbacks
	}

	for record = record[2:]; len(record) >= 2; record = record[2:] {
		lootPath, countString := record[0], record[1]

		back, err := model.GetBack(lootPath)
		if err != nil {
			continue
		}
		lootCount, err := strconv.Atoi(countString)
		if err != nil {
			continue
		}
		if lootCount < 1 {
			continue
		}

		state.Loot[back] = lootCount
	}

	if len(record) > 0 {
		// TODO: structured log here
		fmt.Printf("WARNING: corrupted user loot record discovered. userID: %v remainder of record: %v\n", userID, record)
	}

	return userID, state, nil
}

func CSVRecordFromState(userID UserID, userState UserLootState) []string {
	var record []string

	record = append(record, string(userID), strconv.Itoa(userState.Greenbacks))

	var lootItems []LootItem
	for back, count := range userState.Loot {
		if count < 1 {
			continue
		}
		lootItems = append(lootItems, LootItem{Back: back, Count: count})
	}

	sortLootItemsByPath(lootItems)

	for _, lootItem := range lootItems {
		record = append(record, lootItem.Path(), strconv.Itoa(lootItem.Count))
	}

	return record
}

type LootBag interface {
	GetState(userID UserID) UserLootState
	AddLoot(userID UserID, loot model.Back)
	RemoveLoot(userID UserID, loot model.Back) bool
	// TODO: IMPL!
	// AddGreenbacks(userID UserID, gb int)
	// SubtractGreenbacks(userID UserID, gb int)
	Rollback(userID UserID)
}

type FlushPolicy interface {
	NotifyFlush()
	ShouldFlush() bool
}

type stalenessFlushPolicy struct {
	flushThreshold time.Duration
	lastFlushed    time.Time
}

func (sfp *stalenessFlushPolicy) NotifyFlush() {
	sfp.lastFlushed = time.Now()
}

func (sfp *stalenessFlushPolicy) ShouldFlush() bool {
	return time.Since(sfp.lastFlushed) > sfp.flushThreshold
}

type csvLootBag struct {
	file        *os.File
	userStates  map[UserID]UserLootState
	flushPolicy FlushPolicy
}

var _ LootBag = new(csvLootBag) // *csvLootBag implements LootBag

func NewCsvLootBag(datapath string) (*csvLootBag, error) {
	file, err := os.OpenFile(datapath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open csv loot bag data file: %w", err)
	}

	// CSV format: "userID","<greenbacks int>","<back-1-path>","<back-1-count>",...,"<back-n-path>","<back-n-count>"
	reader := csv.NewReader(file)
	// allow variable number of fields per record
	reader.FieldsPerRecord = -1
	restoredData, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error while reading csv file. filepath: %v err: %w", datapath, err)
	}

	userStates := make(map[UserID]UserLootState)

	for _, record := range restoredData {
		if len(record) < 2 {
			continue
		}

		userID, restoredState, err := StateFromCSVRecord(record)
		if err != nil {
			// TODO: structured log
			fmt.Printf("error restoring loot state from csv record. record: %v\n", record)
			continue
		}

		userStates[userID] = restoredState
	}

	c := &csvLootBag{
		file:        file,
		userStates:  userStates,
		flushPolicy: new(stalenessFlushPolicy),
	}

	return c, nil
}

func (c *csvLootBag) GetState(userID UserID) UserLootState {
	defer c.maybeFlush()

	return c.userStates[userID]
}

func (c *csvLootBag) AddLoot(userID UserID, loot model.Back) {
	defer c.maybeFlush()

	state := c.userStates[userID]

	if state.Loot == nil {
		state.Loot = make(map[model.Back]int)
	}

	prevCount := state.Loot[loot]

	state.Loot[loot] = prevCount + 1
	c.userStates[userID] = state
}

func (c *csvLootBag) RemoveLoot(userID UserID, loot model.Back) bool {
	defer c.maybeFlush()

	state := c.userStates[userID]

	if state.Loot[loot] < 1 {
		return false
	}

	state.Loot[loot] -= 1

	if state.Loot[loot] < 1 {
		delete(state.Loot, loot)
	}

	// This is technically unnecessary since the previous operations
	// all take direct effect on the Loot map, but why not be defensive
	// against future quirks or changes to the logic?
	c.userStates[userID] = state

	return true
}

func (c *csvLootBag) Rollback(userID UserID) {
	defer c.maybeFlush()

	state := c.userStates[userID]

	state.Loot = make(map[model.Back]int)
	c.userStates[userID] = state
}

func (c *csvLootBag) Shutdown() error {
	defer c.file.Close()

	return c.flush()
}

func (c *csvLootBag) SetFlushPolicy(fp FlushPolicy) {
	if fp != nil {
		c.flushPolicy = fp
	}
}

func (c *csvLootBag) maybeFlush() {
	if !c.flushPolicy.ShouldFlush() {
		return
	}

	err := c.flush()
	if err != nil {
		// TODO: structured log
		fmt.Printf("errored while flushing csv loot state to disk. err: %v\n", err)
	}
}

func (c *csvLootBag) flush() error {
	var records [][]string

	for userID, userState := range c.userStates {
		record := CSVRecordFromState(userID, userState)
		records = append(records, record)
	}

	buf := new(bytes.Buffer)
	{
		w := csv.NewWriter(buf)
		err := w.WriteAll(records)
		if err != nil {
			return fmt.Errorf("failed to prepare csv buffer in lootBag.flush(). err: %w", err)
		}
	}

	// Notify the flush before we do filesystem interactions.
	// If FS actions are failing, they aren't really likely to succeed
	// if we try again soon after.
	c.flushPolicy.NotifyFlush()

	// Set the file's write head back to the top
	_, err := c.file.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("CRITICAL: could not reset file for flushing to csv. err: %w", err)
	}

	// Write the contents of the buffer to the file
	writtenBytes, err := buf.WriteTo(c.file)
	if err != nil {
		return fmt.Errorf("CRITICAL: error while flushing csv buffer to file. err: %w", err)
	}

	// Truncate the file to what was written
	err = c.file.Truncate(writtenBytes)
	if err != nil {
		return fmt.Errorf("error while truncating csv file to fit buffer. err: %w", err)
	}

	return nil
}
