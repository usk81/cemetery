package toki

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"time"
)

type (
	Toki struct {
		layout string
		time.Time
	}

	// A Month specifies a month of the year (January = 1, ...).
	Month = time.Month

	// A Weekday specifies a day of the week (Sunday = 0, ...).
	Weekday = time.Weekday

	// A Location maps time instants to the zone in use at that time.
	// Typically, the Location represents the collection of time offsets
	// in use in a geographical area. For many Locations the time offset varies
	// depending on whether daylight savings time is in use at the time instant.
	Location = time.Location
)

const (
	// Months
	January   = time.January
	February  = time.February
	March     = time.March
	April     = time.April
	May       = time.May
	June      = time.June
	July      = time.July
	August    = time.August
	September = time.September
	October   = time.October
	November  = time.November
	December  = time.December

	// Weekdays
	Sunday    = time.Sunday
	Monday    = time.Monday
	Tuesday   = time.Tuesday
	Wednesday = time.Wednesday
	Thursday  = time.Thursday
	Friday    = time.Friday
	Saturday  = time.Saturday

	// layouts
	RFC3339              = time.RFC3339
	LayoutTimestamp      = "timestamp"
	LayoutTimestampMilli = "timestamp_milli"
	LayoutTimestampNano  = "timestamp_nano"
)

// daysBefore[m] counts the number of days in a non-leap year
// before month m begins. There is an entry for m=12, counting
// the number of days before January of next year (365).
var daysBefore = [...]int32{
	0,
	31,
	31 + 28,
	31 + 28 + 31,
	31 + 28 + 31 + 30,
	31 + 28 + 31 + 30 + 31,
	31 + 28 + 31 + 30 + 31 + 30,
	31 + 28 + 31 + 30 + 31 + 30 + 31,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31 + 30,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31 + 30 + 31,
}

var (
	// UTC represents Universal Coordinated Time (UTC).
	UTC = time.UTC
)

func setLayout(layouts ...string) string {
	layout := RFC3339
	if len(layouts) >= 1 && layouts[0] != "" {
		layout = layouts[0]
	}
	return layout
}

func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func New(layouts ...string) Toki {
	return Toki{layout: setLayout(layouts...)}
}

func Now(layouts ...string) Toki {
	return Toki{layout: setLayout(layouts...), Time: time.Now()}
}

func Unix(sec int64, nsec int64, layouts ...string) Toki {
	return Toki{layout: setLayout(layouts...), Time: time.Unix(sec, nsec)}
}

func UnixMilli(msec int64, layouts ...string) Toki {
	return Toki{layout: setLayout(layouts...), Time: time.UnixMilli(msec)}
}

func UnixMicro(usec int64, layouts ...string) Toki {
	return Toki{layout: setLayout(layouts...), Time: time.UnixMicro(usec)}
}

func Date(year int, month Month, day, hour, min, sec, nsec int, loc *time.Location, layouts ...string) Toki {
	return Toki{layout: setLayout(layouts...), Time: time.Date(year, month, day, hour, min, sec, nsec, loc)}
}

func DaysIn(m Month, year int) int {
	if m == February && isLeap(year) {
		return 29
	}
	return int(daysBefore[m] - daysBefore[m-1])
}

func FixedZone(name string, offset int) *Location {
	return time.FixedZone(name, offset)
}

func Parse(layout, value string, layouts ...string) (Toki, error) {
	t, err := time.Parse(layout, value)
	return Toki{layout: setLayout(layouts...), Time: t}, err
}

func (t Toki) Add(d time.Duration) Toki {
	t.Time = t.Time.Add(d)
	return t
}

func (t Toki) AddDate(years int, months int, days int) Toki {
	t.Time = t.Time.AddDate(years, months, days)
	return t
}

func (t Toki) After(u Toki) bool {
	return t.Time.After(u.ToTime())
}

func (t Toki) AppendFormat(b []byte, layout string) []byte {
	return t.Time.AppendFormat(b, layout)
}

func (t Toki) Before(u Toki) bool {
	return t.Time.Before(u.ToTime())
}

func (t Toki) Clock() (hour, min, sec int) {
	return t.Time.Clock()
}

func (t Toki) Date() (year int, month Month, day int) {
	return t.Time.Date()
}

func (t Toki) Day() int {
	return t.Time.Day()
}

func (t Toki) Equal(u Toki) bool {
	return t.Time.Equal(u.ToTime())
}

func (t Toki) Format(layout string) string {
	return t.Time.Format(layout)
}

func (t Toki) GetLayout() string {
	if t.layout != "" {
		return t.layout
	}
	return RFC3339
}

func (t Toki) GoString() string {
	return t.Time.GoString()
}

func (t *Toki) GobDecode(data []byte) error {
	return t.Time.GobDecode(data)
}

func (t Toki) GobEncode() ([]byte, error) {
	return t.Time.GobEncode()
}

func (t Toki) Hour() int {
	return t.Time.Hour()
}

func (t Toki) ISOWeek() (year, week int) {
	return t.Time.ISOWeek()
}

func (t Toki) In(loc *time.Location) Toki {
	_ = t.Time.In(loc)

	return t
}

func (t Toki) IsDST() bool {
	return t.Time.IsDST()
}

func (t Toki) IsZero() bool {
	return t.Time.IsZero()
}

func (t Toki) Local() Toki {
	_ = t.Time.Local()
	return t
}

func (t Toki) Location() *time.Location {
	return t.Time.Location()
}

func (t Toki) MarshalBinary() ([]byte, error) {
	return t.Time.MarshalBinary()
}

func (t Toki) MarshalJSON() ([]byte, error) {
	if t.GetLayout() == RFC3339 {
		return t.Time.MarshalJSON()
	}
	buf := new(bytes.Buffer)
	var err error
	switch t.GetLayout() {
	case LayoutTimestamp:
		i := t.Time.Unix()
		err = binary.Write(buf, binary.BigEndian, i)
	case LayoutTimestampMilli:
		i := t.Time.UnixMilli()
		err = binary.Write(buf, binary.BigEndian, i)
	case LayoutTimestampNano:
		i := t.Time.UnixNano()
		err = binary.Write(buf, binary.BigEndian, i)
	default:
		s := t.Time.Format(t.GetLayout())
		_, err = buf.WriteString(`"` + s + `"`)
	}

	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (t Toki) MarshalText() ([]byte, error) {
	if t.GetLayout() == RFC3339 {
		return t.Time.MarshalText()
	}
	buf := new(bytes.Buffer)
	var err error
	switch t.GetLayout() {
	case LayoutTimestamp:
		i := t.Time.Unix()
		err = binary.Write(buf, binary.BigEndian, i)
	case LayoutTimestampMilli:
		i := t.Time.UnixMilli()
		err = binary.Write(buf, binary.BigEndian, i)
	case LayoutTimestampNano:
		i := t.Time.UnixNano()
		err = binary.Write(buf, binary.BigEndian, i)
	default:
		s := t.Time.Format(t.GetLayout())
		_, err = buf.WriteString(s)
	}

	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (t Toki) Minute() int {
	return t.Time.Minute()
}

func (t Toki) Month() Month {
	return t.Time.Month()
}

func (t Toki) Nanosecond() int {
	return t.Time.Nanosecond()
}

func (t Toki) Round(d time.Duration) Toki {
	v := t.Time.Round(d)
	return Toki{
		layout: t.GetLayout(),
		Time:   v,
	}
}

func (t Toki) Second() int {
	return t.Time.Second()
}

func (t Toki) String() string {
	return t.Time.String()
}

func (t Toki) Sub(u Toki) time.Duration {
	return t.Time.Sub(u.ToTime())
}

func (t Toki) ToTime() time.Time {
	return t.Time
}

func (t Toki) Truncate(d time.Duration) Toki {
	t.Time = t.Time.Truncate(d)
	return t
}

func (t Toki) UTC() Toki {
	t.Time = t.Time.UTC()
	return t
}

func (t Toki) Unix() int64 {
	return t.Time.Unix()
}

func (t Toki) UnixMicro() int64 {
	return t.Time.UnixMicro()
}

func (t Toki) UnixMilli() int64 {
	return t.Time.UnixMilli()
}

func (t Toki) UnixNano() int64 {
	return t.Time.UnixNano()
}

func (t *Toki) UnmarshalBinary(data []byte) error {
	return t.Time.UnmarshalBinary(data)
}

func (t *Toki) UnmarshalJSON(data []byte) error {
	fmt.Printf("UnmarshalJSON: GetLayout: %s\n", t.GetLayout())

	if t.GetLayout() == RFC3339 {
		return t.Time.UnmarshalJSON(data)
	}

	s := string(data)
	if s == "null" {
		return nil
	}

	var err error
	switch t.GetLayout() {
	case LayoutTimestamp:
		var i int64
		if i, err = strconv.ParseInt(s, 10, 64); err == nil {
			t.Time = time.Unix(i, 0)
		}
	case LayoutTimestampMilli:
		var i int64
		if i, err = strconv.ParseInt(s, 10, 64); err == nil {
			t.Time = time.UnixMilli(i)
		}
	case LayoutTimestampNano:
		var i int64
		if i, err = strconv.ParseInt(s, 10, 64); err == nil {
			t.Time = time.Unix(0, i)
		}
	default:
		if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
			return errors.New("Time.UnmarshalJSON: input is not a JSON string")
		}
		data = data[len(`"`) : len(data)-len(`"`)]
		s = string(data)
		t.Time, err = time.Parse(t.GetLayout(), s)
	}

	if err != nil {
		if e := t.Time.UnmarshalJSON(data); e == nil {
			return nil
		}
	}

	return err
}

func (t *Toki) UnmarshalText(data []byte) error {
	if t.GetLayout() == RFC3339 {
		return t.Time.UnmarshalText(data)
	}

	s := string(data)

	var err error
	switch t.GetLayout() {
	case LayoutTimestamp:
		var i int64
		if i, err = strconv.ParseInt(s, 10, 64); err != nil {
			t.Time = time.Unix(i, 0)
		}
	case LayoutTimestampMilli:
		var i int64
		if i, err = strconv.ParseInt(s, 10, 64); err != nil {
			t.Time = time.UnixMilli(i)
		}
	case LayoutTimestampNano:
		var i int64
		if i, err = strconv.ParseInt(s, 10, 64); err != nil {
			t.Time = time.Unix(0, i)
		}
	default:
		t.Time, err = time.Parse(t.GetLayout(), s)
	}

	if err != nil {
		if e := t.Time.UnmarshalText(data); e == nil {
			return nil
		}
	}

	return err
}

func (t Toki) Weekday() Weekday {
	return t.Time.Weekday()
}

func (t Toki) Year() int {
	return t.Time.Year()
}

func (t Toki) YearDay() int {
	return t.Time.YearDay()
}

func (t Toki) Zone() (name string, offset int) {
	return t.Time.Zone()
}

func (t Toki) ZoneBounds() (start, end Toki) {
	st, ed := t.Time.ZoneBounds()
	start = Toki{
		layout: t.GetLayout(),
		Time:   st,
	}
	end = Toki{
		layout: t.GetLayout(),
		Time:   ed,
	}
	return
}
