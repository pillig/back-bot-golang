package loot

import (
	"back-bot/backs"
	"testing"
)

func testBack(path string) backs.Back {
	back, _ := backs.GetBack(path)
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
				Loot:       map[backs.Back]int{testBack("a"): 1, testBack("b"): 2},
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
				Loot:       map[backs.Back]int{testBack("a"): 1, testBack("b"): 2},
			},
			wantErr: false,
		},
		{
			record:         []string{"bigback", "0", "a", "1", "b", "0", "c", "419"},
			expectedUserID: "bigback",
			expectedState: UserLootState{
				Greenbacks: 0,
				Loot:       map[backs.Back]int{testBack("a"): 1, testBack("c"): 419},
			},
			wantErr: false,
		},
		{
			record:         []string{"bigback", "0", "a", "1", "b", ":()", "c", "419"},
			expectedUserID: "bigback",
			expectedState: UserLootState{
				Greenbacks: 0,
				Loot:       map[backs.Back]int{testBack("a"): 1, testBack("c"): 419},
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
				Loot: map[backs.Back]int{
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
