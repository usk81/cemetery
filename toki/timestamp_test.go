package toki

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

var timestampJsonTests = []struct {
	time time.Time
	json string
}{
	{time.Date(9999, 4, 12, 23, 20, 50, 520*1e6, UTC), `253379575250`},
	{time.Date(1996, 12, 19, 16, 39, 57, 0, local()), `851042397`},
	{time.Date(0, 1, 1, 0, 0, 0, 1, FixedZone("", 1*60)), `-62167219260`},
	{time.Date(2020, 1, 1, 0, 0, 0, 0, FixedZone("", 23*60*60+59*60)), `1577750460`},
}

var timestampMilliJsonTests = []struct {
	time time.Time
	json string
}{
	{time.Date(9999, 4, 12, 23, 20, 50, 520*1e6, UTC), `253379575250520`},
	{time.Date(1996, 12, 19, 16, 39, 57, 0, local()), `851042397000`},
	{time.Date(0, 1, 1, 0, 0, 0, 1, FixedZone("", 1*60)), `-62167219260000`},
	{time.Date(2020, 1, 1, 0, 0, 0, 0, FixedZone("", 23*60*60+59*60)), `1577750460000`},
}

var timestampNanoJsonTests = []struct {
	time time.Time
	json string
}{
	{time.Date(9999, 4, 12, 23, 20, 50, 520*1e6, UTC), `-4874841781413722624`}, // FIXME: overflow
	{time.Date(1996, 12, 19, 16, 39, 57, 0, local()), `851042397000000000`},
	{time.Date(0, 1, 1, 0, 0, 0, 1, FixedZone("", 1*60)), `-6826987038871345151`},
	{time.Date(2020, 1, 1, 0, 0, 0, 0, FixedZone("", 23*60*60+59*60)), `1577750460000000000`},
}

func TestTimestampJSON(t *testing.T) {
	for _, tt := range timestampJsonTests {
		jsonTime := Timestamp{}
		u := tt.time.Unix()

		j := &jsonTime

		fmt.Printf("debug : %s\n", j.GetLayout())

		var jsonBytes []byte
		var err error
		if jsonBytes, err = json.Marshal(u); err != nil {
			t.Errorf("%v json.Marshal error = %v, want nil", tt.time, err)
		} else if string(jsonBytes) != tt.json {
			t.Errorf("%v JSON = %#q, want %#q", tt.time, string(jsonBytes), tt.json)
		} else if err = json.Unmarshal(jsonBytes, &jsonTime); err != nil {
			t.Errorf("%v json.Unmarshal error = %v, want nil", tt.time, err)
		}
	}
}

func TestTimestampMilliJSON(t *testing.T) {
	for _, tt := range timestampMilliJsonTests {
		jsonTime := TimestampMilli{}
		u := tt.time.UnixMilli()

		var jsonBytes []byte
		var err error
		if jsonBytes, err = json.Marshal(u); err != nil {
			t.Errorf("%v json.Marshal error = %v, want nil", tt.time, err)
		} else if string(jsonBytes) != tt.json {
			t.Errorf("%v JSON = %#q, want %#q", tt.time, string(jsonBytes), tt.json)
		} else if err = json.Unmarshal(jsonBytes, &jsonTime); err != nil {
			t.Errorf("%v json.Unmarshal error = %v, want nil", tt.time, err)
		}
	}
}

func TestTimestampNanoJSON(t *testing.T) {
	for _, tt := range timestampNanoJsonTests {
		jsonTime := TimestampNano{}
		u := tt.time.UnixNano()

		var jsonBytes []byte
		var err error
		if jsonBytes, err = json.Marshal(u); err != nil {
			t.Errorf("%v json.Marshal error = %v, want nil", tt.time, err)
		} else if string(jsonBytes) != tt.json {
			t.Errorf("%v JSON = %#q, want %#q", tt.time, string(jsonBytes), tt.json)
		} else if err = json.Unmarshal(jsonBytes, &jsonTime); err != nil {
			t.Errorf("%v json.Unmarshal error = %v, want nil", tt.time, err)
		}
	}
}
