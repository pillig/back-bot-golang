package loot

import (
	"back-bot/backs/model"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
)

func testBack(path string) model.Back {
	back, _ := model.GetBack(path)
	return back
}

func TestStateFromCSVRecord(t *testing.T) {
	cases := []struct {
		record         []string
		expectedUserID UserID
		expectedState  UserLootState
		wantErr        bool
	}{
		{
			record:         []string{"bigback", "0", "a", "1", "b", "2"},
			expectedUserID: "bigback",
			expectedState: UserLootState{
				Greenbacks: 0,
				Loot:       map[model.Back]int{testBack("a"): 1, testBack("b"): 2},
			},
			wantErr: false,
		},
		{
			record:         []string{"bigback", "500"},
			expectedUserID: "bigback",
			expectedState: UserLootState{
				Greenbacks: 500,
				Loot:       nil,
			},
			wantErr: false,
		},
		{
			record:         []string{"", "500"},
			expectedUserID: "",
			wantErr:        true,
		},
		{
			record:         []string{"bigback", "im da joker"},
			expectedUserID: "",
			wantErr:        true,
		},
		{
			record:         []string{"bigback", "0", "a", "1", "b", "2", "c"},
			expectedUserID: "bigback",
			expectedState: UserLootState{
				Greenbacks: 0,
				Loot:       map[model.Back]int{testBack("a"): 1, testBack("b"): 2},
			},
			wantErr: false,
		},
		{
			record:         []string{"bigback", "0", "a", "1", "b", "0", "c", "419"},
			expectedUserID: "bigback",
			expectedState: UserLootState{
				Greenbacks: 0,
				Loot:       map[model.Back]int{testBack("a"): 1, testBack("c"): 419},
			},
			wantErr: false,
		},
		{
			record:         []string{"bigback", "0", "a", "1", "b", ":()", "c", "419"},
			expectedUserID: "bigback",
			expectedState: UserLootState{
				Greenbacks: 0,
				Loot:       map[model.Back]int{testBack("a"): 1, testBack("c"): 419},
			},
			wantErr: false,
		},
	}

	for _, c := range cases {
		actualUserID, actualState, err := StateFromCSVRecord(c.record)

		if actualUserID != c.expectedUserID {
			t.Fatalf("actual user ID (%v) does not match expected (%v)", actualUserID, c.expectedUserID)
		}

		if actualState.Greenbacks != c.expectedState.Greenbacks {
			t.Fatalf("actual state greenbacks (%v) does not match expected (%v)", actualState.Greenbacks, c.expectedState.Greenbacks)
		}

		mapLenEquals := len(actualState.Loot) == len(c.expectedState.Loot)
		if !mapLenEquals {
			t.Fatalf("actual state loot map (%v) is different length from expected (%v)", actualState.Loot, c.expectedState.Loot)
		}

		for k, v := range c.expectedState.Loot {
			vv, found := actualState.Loot[k]
			if !found || v != vv {
				t.Fatalf("mismatched entries between actual loot map (%v, %v) and expected (%v, %v)", k, vv, k, v)
			}
		}

		if (err != nil) != c.wantErr {
			t.Fatalf("wanted err? (%v) but got (%v)", c.wantErr, err)
		}
	}
}

func TestCSVRecordFromState(t *testing.T) {
	cases := []struct {
		userID         UserID
		state          UserLootState
		expectedRecord []string
	}{
		{
			userID:         "bigback",
			state:          UserLootState{},
			expectedRecord: []string{"bigback", "0"},
		},
		{
			userID: "bigback",
			state: UserLootState{
				Greenbacks: 419,
			},
			expectedRecord: []string{"bigback", "419"},
		},
		{
			userID: "bigback",
			state: UserLootState{
				Greenbacks: 419,
				Loot: map[model.Back]int{
					testBack("zzz"): 1,
					testBack("zza"): 5,
					testBack("aa"):  10,
					testBack("ab"):  3,
					testBack("ac"):  0,
				},
			},
			expectedRecord: []string{"bigback", "419", "aa", "10", "ab", "3", "zza", "5", "zzz", "1"},
		},
	}

	for _, c := range cases {
		actualRecord := CSVRecordFromState(c.userID, c.state)

		if len(actualRecord) != len(c.expectedRecord) {
			t.Fatalf("length of actual record (%v) does not match expected length (%v)", len(actualRecord), len(c.expectedRecord))
		}

		for i, v := range c.expectedRecord {
			if v != actualRecord[i] {
				t.Fatalf("actual entry (%v: %v) does not match expected (%v)", i, actualRecord[i], v)
			}
		}
	}
}

type testFlushPolicy bool

func (t testFlushPolicy) ShouldFlush() bool { return bool(t) }
func (t testFlushPolicy) NotifyFlush()      {}

func TestCsvLootBag(t *testing.T) {
	testfilepath := filepath.Join(".", "test_loot.csv")

	truncateTestFile := func() {
		file, err := os.OpenFile(testfilepath, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			t.Fatal(err)
		}
		file.Truncate(0)
		file.Close()
	}
	truncateTestFile()

	isTestFileEmpty := func() bool {
		fi, err := os.Stat(testfilepath)
		if err != nil {
			t.Fatal(err)
		}

		return fi.Size() == 0
	}

	getTestRecords := func() [][]string {
		file, err := os.Open(testfilepath)
		if err != nil {
			t.Fatal(err)
		}

		defer file.Close()

		csvR := csv.NewReader(file)
		// allow variable number of fields per record
		csvR.FieldsPerRecord = -1
		records, err := csvR.ReadAll()
		if err != nil {
			t.Fatal(err)
		}

		return records
	}

	// clean up test file after test suite
	defer func() {
		err := os.Remove(testfilepath)
		if err != nil {
			t.Fatal(err)
		}
	}()

	testback1 := testBack("back-one")
	testback2 := testBack("back-two")
	testback3 := testBack("back-three")

	t.Run("test simple in-memory happy path: add, get, remove, get, add, rollback", func(t *testing.T) {
		csvLB, err := NewCsvLootBag(testfilepath)
		if err != nil {
			t.Fatal(err)
		}

		csvLB.SetFlushPolicy(testFlushPolicy(false))

		csvLB.AddLoot("bigback", testback1)

		state := csvLB.GetState("bigback")
		if state.Loot[testback1] != 1 {
			t.Fatalf("expected %v count for back %v, got %v", 1, testback1, state.Loot[testback1])
		}

		csvLB.RemoveLoot("bigback", testback1)

		state = csvLB.GetState("bigback")
		if state.Loot[testback1] != 0 {
			t.Fatalf("expected %v count for back %v, got %v", 0, testback1, state.Loot[testback1])
		}

		csvLB.AddLoot("bigback", testback1)
		csvLB.AddLoot("bigback", testback2)
		csvLB.AddLoot("bigback", testback3)
		csvLB.AddLoot("bigback", testback2)

		state = csvLB.GetState("bigback")
		if state.Loot[testback1] != 1 {
			t.Fatalf("expected %v count for back %v, got %v", 1, testback1, state.Loot[testback1])
		}
		if state.Loot[testback2] != 2 {
			t.Fatalf("expected %v count for back %v, got %v", 2, testback2, state.Loot[testback2])
		}
		if state.Loot[testback3] != 1 {
			t.Fatalf("expected %v count for back %v, got %v", 1, testback3, state.Loot[testback3])
		}

		csvLB.Rollback("bigback")

		state = csvLB.GetState("bigback")
		if len(state.Loot) > 0 {
			t.Fatalf("expected state.Loot to be empty, got %v", state.Loot)
		}

	})

	t.Run("test simple flush scenario", func(t *testing.T) {
		truncateTestFile()

		csvLB, err := NewCsvLootBag(testfilepath)
		if err != nil {
			t.Fatal(err)
		}

		// get some in memory first to assert flush policy has intended effect
		csvLB.SetFlushPolicy(testFlushPolicy(false))

		csvLB.AddLoot("bigback", testback1)
		csvLB.AddLoot("bigback", testback2)
		csvLB.AddLoot("bigback", testback3)

		if !isTestFileEmpty() {
			t.Fatal("flushed records when flushPolicy should have blocked")
		}

		csvLB.SetFlushPolicy(testFlushPolicy(true))

		csvLB.AddLoot("bigback", testback1)

		if isTestFileEmpty() {
			t.Fatalf("test file unexpectedly empty")
		}

		if records := getTestRecords(); len(records) != 1 {
			t.Fatalf("expected 1 test record, got %v. records: %v", len(records), records)
		}

		csvLB.AddLoot("parkour", testback2)

		if isTestFileEmpty() {
			t.Fatalf("test file unexpectedly empty")
		}

		if records := getTestRecords(); len(records) != 2 {
			t.Fatalf("expected 2 test records, got %v. records: %v", len(records), records)
		}
	})

	t.Run("file persistence across lootbag incarnations", func(t *testing.T) {
		truncateTestFile()

		csvLB, err := NewCsvLootBag(testfilepath)
		if err != nil {
			t.Fatal(err)
		}

		csvLB.SetFlushPolicy(testFlushPolicy(true))

		csvLB.AddLoot("bigback", testback1)
		csvLB.AddLoot("parkour", testback2)

		if state := csvLB.GetState("bigback"); state.Loot[testback1] != 1 {
			t.Fatalf("failed to write bigback's testback1 add")
		}
		if state := csvLB.GetState("parkour"); state.Loot[testback2] != 1 {
			t.Fatalf("failed to write parkour's testback2 add")
		}

		// create a separate lootbag instance. it should return the same state
		// as the last one (although they should not be both live at the same time!)
		csvLB2, err := NewCsvLootBag(testfilepath)
		if err != nil {
			t.Fatal(err)
		}

		if state := csvLB2.GetState("bigback"); state.Loot[testback1] != 1 {
			t.Fatalf("failed to restore  bigback's testback1 state")
		}
		if state := csvLB2.GetState("parkour"); state.Loot[testback2] != 1 {
			t.Fatalf("failed to restore parkour's testback2 state")
		}
	})
}
