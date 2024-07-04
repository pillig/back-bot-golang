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

func CSVRecordFromState(userID UserID, userState UserLootState) ([]string, error) {
	var record []string

	record = append(record, string(userID), strconv.Itoa(userState.Greenbacks))

	type lootItem struct {
		path  string
		count string
	}

	var lootItems []lootItem
	for loot, count := range userState.Loot {
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

	return record, nil
}

type LootBag interface {
	// TODO: Will GetState ever return an error?
	GetState(userID UserID) (UserLootState, error)
	AddLoot(userID UserID, loot backs.Back) error
	RedeemLoot(userID UserID, loot backs.Back) error
	SellLoot(userID UserID, loot backs.Back) (int, error)
	Rollback(userID UserID) error
}

type csvLootBag struct {
	file       *os.File
	userStates map[UserID]UserLootState
	pulse      <-chan time.Time
}

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

	return &csvLootBag{
		file:       file,
		userStates: userStates,
	}, nil
}

func (c *csvLootBag) GetState(userID UserID) (UserLootState, error) {
	return c.userStates[userID], nil
}

func (c *csvLootBag) AddLoot(userID UserID, loot backs.Back) error         {}
func (c *csvLootBag) RedeemLoot(userID UserID, loot backs.Back) error      {}
func (c *csvLootBag) SellLoot(userID UserID, loot backs.Back) (int, error) {}
func (c *csvLootBag) Rollback(userID UserID) error                         {}

func (c *csvLootBag) Shutdown() error {
	defer c.file.Close()

	return c.flush()
}

func (c *csvLootBag) flush() error {
	var records [][]string

	for userID, userState := range c.userStates {
		record, err := CSVRecordFromState(userID, userState)
		if err != nil {
			// TODO: structured log
			fmt.Printf("error serializing user state to csv record. userID: %v userState: %v err: %v\n", userID, userState, err)
		}

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

	return nil
}

func (c *csvLootBag) poll(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			fmt.Println("csv loot bag poller shutting down")
			return

		case <-c.pulse:
			err := c.flush()
			if err != nil {
				fmt.Println(fmt.Errorf("error flushing in csv loot poller: %w", err))
			}
		}
	}
}

func (c *csvLootBag) tick() {
	c.pulse = time.After(5 * time.Second)
}
