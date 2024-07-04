package loot

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"slices"
	"strconv"
	"time"

	"back-bot/backs"
)

type UserID string

type UserLootState struct {
	Loot       map[backs.Back]int
	Greenbacks int
}

func StateFromCSVRecord(record []string) (UserID, UserLootState, error) {
	state := UserLootState{
		Loot: make(map[backs.Back]int),
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

		back, err := backs.GetBack(lootPath)
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

	type lootItem struct {
		path  string
		count string
	}

	var lootItems []lootItem
	for loot, count := range userState.Loot {
		if count < 1 {
			continue
		}
		lootItems = append(lootItems, lootItem{path: loot.Path(), count: strconv.Itoa(count)})
	}

	slices.SortFunc(lootItems, func(a, b lootItem) int {
		switch true {
		case a.path < b.path:
			return -1
		case a.path > b.path:
			return 1
		default:
			return 0
		}
	})

	for _, lootItem := range lootItems {
		record = append(record, lootItem.path, lootItem.count)
	}

	return record
}

type LootBag interface {
	GetState(userID UserID) UserLootState
	AddLoot(userID UserID, loot backs.Back)
	RemoveLoot(userID UserID, loot backs.Back) bool
	Rollback(userID UserID)
}

type csvLootBag struct {
	file       *os.File
	userStates map[UserID]UserLootState
	lastSaved  time.Time
}

var _ LootBag = &csvLootBag{}

func NewCsvLootBag(datapath string) (*csvLootBag, error) {
	file, err := os.OpenFile(datapath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open csv loot bag data file: %w", err)
	}

	// CSV format: "userID","<greenbacks int>","<back-1-path>","<back-1-count>",...,"<back-n-path>","<back-n-count>"
	reader := csv.NewReader(file)
	restoredData, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error while reading csv file. filepath: %v err: %w", datapath, err)
	}

	userStates := make(map[UserID]UserLootState)

	for _, record := range restoredData {
		userID, restoredState, err := StateFromCSVRecord(record)
		if err != nil {
			// TODO: structured log
			fmt.Printf("error restoring loot state from csv record. record: %v\n", record)
			continue
		}

		userStates[userID] = restoredState
	}

	c := &csvLootBag{
		file:       file,
		userStates: userStates,
	}

	return c, nil
}

func (c *csvLootBag) GetState(userID UserID) UserLootState {
	defer c.maybeFlush()

	return c.userStates[userID]
}

func (c *csvLootBag) AddLoot(userID UserID, loot backs.Back) {
	defer c.maybeFlush()

	state := c.userStates[userID]

	if state.Loot == nil {
		state.Loot = make(map[backs.Back]int)
	}

	prevCount := state.Loot[loot]

	state.Loot[loot] = prevCount + 1
	c.userStates[userID] = state
}

func (c *csvLootBag) RemoveLoot(userID UserID, loot backs.Back) bool {
	defer c.maybeFlush()

	state := c.userStates[userID]
	if state.Loot[loot] < 1 {
		return false
	}

	state.Loot[loot] -= 1

	return true
}

func (c *csvLootBag) Rollback(userID UserID) {
	defer c.maybeFlush()

	state := c.userStates[userID]

	state.Loot = make(map[backs.Back]int)
	c.userStates[userID] = state
}

func (c *csvLootBag) Shutdown() error {
	defer c.file.Close()

	return c.flush()
}

func (c *csvLootBag) maybeFlush() {
	if time.Since(c.lastSaved) < 15*time.Second {
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

	c.lastSaved = time.Now()

	return nil
}
