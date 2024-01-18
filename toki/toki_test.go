package toki

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// parsedTime is the struct representing a parsed time value.
type parsedTime struct {
	Year                 int
	Month                Month
	Day                  int
	Hour, Minute, Second int // 15:04:05 is 15, 4, 5.
	Nanosecond           int // Fractional second.
	Weekday              time.Weekday
	ZoneOffset           int    // seconds east of UTC, e.g. -7*60*60 for -0700
	Zone                 string // e.g., "MST"
}

type TimeTest struct {
	seconds int64
	golden  parsedTime
}

type YearDayTest struct {
	year, month, day int
	yday             int
}

type ISOWeekTest struct {
	year       int // year
	month, day int // month and day
	yex        int // expected year
	wex        int // expected week
}

const (
	// The time routines provide no way to get absolute time
	// (seconds since zero), but we need it to compute the right
	// answer for bizarre roundings like "to the nearest 3 ns".
	// Compute as t - year1 = (t - 1970) + (1970 - 2001) + (2001 - 1).
	// t - 1970 is returned by Unix and Nanosecond.
	// 1970 - 2001 is -(31*365+8)*86400 = -978307200 seconds.
	// 2001 - 1 is 2000*365.2425*86400 = 63113904000 seconds.
	unixToZero = -978307200 + 63113904000

	minDuration time.Duration = -1 << 63
	maxDuration time.Duration = 1<<63 - 1

	testdataRFC3339UTC = "2020-08-22T11:27:43.123456789Z"

	secondsPerMinute = 60
	secondsPerHour   = 60 * secondsPerMinute
	secondsPerDay    = 24 * secondsPerHour

	unixToInternal int64 = (1969*365 + 1969/4 - 1969/100 + 1969/400) * secondsPerDay
)

var utctests = []TimeTest{
	{
		seconds: 0,
		golden: parsedTime{
			Year:       1970,
			Month:      January,
			Day:        1,
			Hour:       0,
			Minute:     0,
			Second:     0,
			Nanosecond: 0,
			Weekday:    time.Thursday,
			ZoneOffset: 0,
			Zone:       "UTC",
		},
	},
	{
		seconds: 1221681866,
		golden: parsedTime{
			Year:       2008,
			Month:      time.September,
			Day:        17,
			Hour:       20,
			Minute:     4,
			Second:     26,
			Nanosecond: 0,
			Weekday:    time.Wednesday,
			ZoneOffset: 0,
			Zone:       "UTC",
		},
	},
	{
		seconds: -1221681866,
		golden: parsedTime{
			Year:       1931,
			Month:      April,
			Day:        16,
			Hour:       3,
			Minute:     55,
			Second:     34,
			Nanosecond: 0,
			Weekday:    time.Thursday,
			ZoneOffset: 0,
			Zone:       "UTC",
		},
	},
	{
		seconds: -11644473600,
		golden: parsedTime{
			Year:       1601,
			Month:      January,
			Day:        1,
			Hour:       0,
			Minute:     0,
			Second:     0,
			Nanosecond: 0,
			Weekday:    time.Monday,
			ZoneOffset: 0,
			Zone:       "UTC",
		},
	},
	{
		seconds: 599529660,
		golden: parsedTime{
			Year:       1988,
			Month:      December,
			Day:        31,
			Hour:       0,
			Minute:     1,
			Second:     0,
			Nanosecond: 0,
			Weekday:    time.Saturday,
			ZoneOffset: 0,
			Zone:       "UTC",
		},
	},
	{
		seconds: 978220860,
		golden: parsedTime{
			Year:       2000,
			Month:      December,
			Day:        31,
			Hour:       0,
			Minute:     1,
			Second:     0,
			Nanosecond: 0,
			Weekday:    time.Sunday,
			ZoneOffset: 0,
			Zone:       "UTC",
		},
	},
}

var nanoutctests = []TimeTest{
	{
		seconds: 0,
		golden: parsedTime{
			Year:       1970,
			Month:      January,
			Day:        1,
			Hour:       0,
			Minute:     0,
			Second:     0,
			Nanosecond: 1e8,
			Weekday:    Thursday,
			ZoneOffset: 0,
			Zone:       "UTC",
		},
	},
	{
		seconds: 1221681866,
		golden: parsedTime{
			Year:       2008,
			Month:      September,
			Day:        17,
			Hour:       20,
			Minute:     4,
			Second:     26,
			Nanosecond: 2e8,
			Weekday:    Wednesday,
			ZoneOffset: 0,
			Zone:       "UTC",
		},
	},
}

var localtests = []TimeTest{
	{0, parsedTime{1969, December, 31, 16, 0, 0, 0, Wednesday, -8 * 60 * 60, "PST"}},
	{1221681866, parsedTime{2008, September, 17, 13, 4, 26, 0, Wednesday, -7 * 60 * 60, "PDT"}},
	{2159200800, parsedTime{2038, June, 3, 11, 0, 0, 0, Thursday, -7 * 60 * 60, "PDT"}},
	{2152173599, parsedTime{2038, March, 14, 1, 59, 59, 0, Sunday, -8 * 60 * 60, "PST"}},
	{2152173600, parsedTime{2038, March, 14, 3, 0, 0, 0, Sunday, -7 * 60 * 60, "PDT"}},
	{2152173601, parsedTime{2038, March, 14, 3, 0, 1, 0, Sunday, -7 * 60 * 60, "PDT"}},
	{2172733199, parsedTime{2038, November, 7, 1, 59, 59, 0, Sunday, -7 * 60 * 60, "PDT"}},
	{2172733200, parsedTime{2038, November, 7, 1, 0, 0, 0, Sunday, -8 * 60 * 60, "PST"}},
	{2172733201, parsedTime{2038, November, 7, 1, 0, 1, 0, Sunday, -8 * 60 * 60, "PST"}},
}

var nanolocaltests = []TimeTest{
	{0, parsedTime{1969, December, 31, 16, 0, 0, 1e8, Wednesday, -8 * 60 * 60, "PST"}},
	{1221681866, parsedTime{2008, September, 17, 13, 4, 26, 3e8, Wednesday, -7 * 60 * 60, "PDT"}},
}

var daysInTests = []struct {
	year, month, di int
}{
	{2011, 1, 31},  // January, first month, 31 days
	{2011, 2, 28},  // February, non-leap year, 28 days
	{2012, 2, 29},  // February, leap year, 29 days
	{2011, 6, 30},  // June, 30 days
	{2011, 12, 31}, // December, last month, 31 days
}

var isoWeekTests = []ISOWeekTest{
	{1981, 1, 1, 1981, 1},
	{1982, 1, 1, 1981, 53},
	{1983, 1, 1, 1982, 52},
	{1984, 1, 1, 1983, 52},
	{1985, 1, 1, 1985, 1},
	{1986, 1, 1, 1986, 1},
	{1987, 1, 1, 1987, 1},
	{1988, 1, 1, 1987, 53},
	{1989, 1, 1, 1988, 52},
	{1990, 1, 1, 1990, 1},
	{1991, 1, 1, 1991, 1},
	{1992, 1, 1, 1992, 1},
	{1993, 1, 1, 1992, 53},
	{1994, 1, 1, 1993, 52},
	{1995, 1, 2, 1995, 1},
	{1996, 1, 1, 1996, 1},
	{1996, 1, 7, 1996, 1},
	{1996, 1, 8, 1996, 2},
	{1997, 1, 1, 1997, 1},
	{1998, 1, 1, 1998, 1},
	{1999, 1, 1, 1998, 53},
	{2000, 1, 1, 1999, 52},
	{2001, 1, 1, 2001, 1},
	{2002, 1, 1, 2002, 1},
	{2003, 1, 1, 2003, 1},
	{2004, 1, 1, 2004, 1},
	{2005, 1, 1, 2004, 53},
	{2006, 1, 1, 2005, 52},
	{2007, 1, 1, 2007, 1},
	{2008, 1, 1, 2008, 1},
	{2009, 1, 1, 2009, 1},
	{2010, 1, 1, 2009, 53},
	{2010, 1, 1, 2009, 53},
	{2011, 1, 1, 2010, 52},
	{2011, 1, 2, 2010, 52},
	{2011, 1, 3, 2011, 1},
	{2011, 1, 4, 2011, 1},
	{2011, 1, 5, 2011, 1},
	{2011, 1, 6, 2011, 1},
	{2011, 1, 7, 2011, 1},
	{2011, 1, 8, 2011, 1},
	{2011, 1, 9, 2011, 1},
	{2011, 1, 10, 2011, 2},
	{2011, 1, 11, 2011, 2},
	{2011, 6, 12, 2011, 23},
	{2011, 6, 13, 2011, 24},
	{2011, 12, 25, 2011, 51},
	{2011, 12, 26, 2011, 52},
	{2011, 12, 27, 2011, 52},
	{2011, 12, 28, 2011, 52},
	{2011, 12, 29, 2011, 52},
	{2011, 12, 30, 2011, 52},
	{2011, 12, 31, 2011, 52},
	{1995, 1, 1, 1994, 52},
	{2012, 1, 1, 2011, 52},
	{2012, 1, 2, 2012, 1},
	{2012, 1, 8, 2012, 1},
	{2012, 1, 9, 2012, 2},
	{2012, 12, 23, 2012, 51},
	{2012, 12, 24, 2012, 52},
	{2012, 12, 30, 2012, 52},
	{2012, 12, 31, 2013, 1},
	{2013, 1, 1, 2013, 1},
	{2013, 1, 6, 2013, 1},
	{2013, 1, 7, 2013, 2},
	{2013, 12, 22, 2013, 51},
	{2013, 12, 23, 2013, 52},
	{2013, 12, 29, 2013, 52},
	{2013, 12, 30, 2014, 1},
	{2014, 1, 1, 2014, 1},
	{2014, 1, 5, 2014, 1},
	{2014, 1, 6, 2014, 2},
	{2015, 1, 1, 2015, 1},
	{2016, 1, 1, 2015, 53},
	{2017, 1, 1, 2016, 52},
	{2018, 1, 1, 2018, 1},
	{2019, 1, 1, 2019, 1},
	{2020, 1, 1, 2020, 1},
	{2021, 1, 1, 2020, 53},
	{2022, 1, 1, 2021, 52},
	{2023, 1, 1, 2022, 52},
	{2024, 1, 1, 2024, 1},
	{2025, 1, 1, 2025, 1},
	{2026, 1, 1, 2026, 1},
	{2027, 1, 1, 2026, 53},
	{2028, 1, 1, 2027, 52},
	{2029, 1, 1, 2029, 1},
	{2030, 1, 1, 2030, 1},
	{2031, 1, 1, 2031, 1},
	{2032, 1, 1, 2032, 1},
	{2033, 1, 1, 2032, 53},
	{2034, 1, 1, 2033, 52},
	{2035, 1, 1, 2035, 1},
	{2036, 1, 1, 2036, 1},
	{2037, 1, 1, 2037, 1},
	{2038, 1, 1, 2037, 53},
	{2039, 1, 1, 2038, 52},
	{2040, 1, 1, 2039, 52},
}

// Test YearDay in several different scenarios
// and corner cases
var yearDayTests = []YearDayTest{
	// Non-leap-year tests
	{2007, 1, 1, 1},
	{2007, 1, 15, 15},
	{2007, 2, 1, 32},
	{2007, 2, 15, 46},
	{2007, 3, 1, 60},
	{2007, 3, 15, 74},
	{2007, 4, 1, 91},
	{2007, 12, 31, 365},

	// Leap-year tests
	{2008, 1, 1, 1},
	{2008, 1, 15, 15},
	{2008, 2, 1, 32},
	{2008, 2, 15, 46},
	{2008, 3, 1, 61},
	{2008, 3, 15, 75},
	{2008, 4, 1, 92},
	{2008, 12, 31, 366},

	// Looks like leap-year (but isn't) tests
	{1900, 1, 1, 1},
	{1900, 1, 15, 15},
	{1900, 2, 1, 32},
	{1900, 2, 15, 46},
	{1900, 3, 1, 60},
	{1900, 3, 15, 74},
	{1900, 4, 1, 91},
	{1900, 12, 31, 365},

	// Year one tests (non-leap)
	{1, 1, 1, 1},
	{1, 1, 15, 15},
	{1, 2, 1, 32},
	{1, 2, 15, 46},
	{1, 3, 1, 60},
	{1, 3, 15, 74},
	{1, 4, 1, 91},
	{1, 12, 31, 365},

	// Year minus one tests (non-leap)
	{-1, 1, 1, 1},
	{-1, 1, 15, 15},
	{-1, 2, 1, 32},
	{-1, 2, 15, 46},
	{-1, 3, 1, 60},
	{-1, 3, 15, 74},
	{-1, 4, 1, 91},
	{-1, 12, 31, 365},

	// 400 BC tests (leap-year)
	{-400, 1, 1, 1},
	{-400, 1, 15, 15},
	{-400, 2, 1, 32},
	{-400, 2, 15, 46},
	{-400, 3, 1, 61},
	{-400, 3, 15, 75},
	{-400, 4, 1, 92},
	{-400, 12, 31, 366},

	// Special Cases

	// Gregorian calendar change (no effect)
	{1582, 10, 4, 277},
	{1582, 10, 15, 288},
}

// Check to see if YearDay is location sensitive
var yearDayLocations = []*Location{
	FixedZone("UTC-8", -8*60*60),
	FixedZone("UTC-4", -4*60*60),
	UTC,
	FixedZone("UTC+4", 4*60*60),
	FixedZone("UTC+8", 8*60*60),
}

// Several ways of getting from
// Fri Nov 18 7:56:35 PST 2011
// to
// Thu Mar 19 7:56:35 PST 2016
var addDateTests = []struct {
	years, months, days int
}{
	{4, 4, 1},
	{3, 16, 1},
	{3, 15, 30},
	{5, -6, -18 - 30 - 12},
}

var gobTests = []Toki{
	Date(0, 1, 2, 3, 4, 5, 6, UTC),
	Date(7, 8, 9, 10, 11, 12, 13, FixedZone("", 0)),
	// FIXME:
	// Decoded time = 2598017997-09-08 03:48:16.985229328 +0900 +0900, want 2598017997-09-08 03:48:16.985229328 +0900 JST
	// Unix(81985467080890095, 0x76543210), // Time.sec: 0x0123456789ABCDEF
	{}, // nil location
	Date(1, 2, 3, 4, 5, 6, 7, FixedZone("", 32767*60)),
	Date(1, 2, 3, 4, 5, 6, 7, FixedZone("", -32768*60)),
}

var truncateRoundTests = []struct {
	t Toki
	d time.Duration
}{
	{Date(-1, January, 1, 12, 15, 30, 5e8, UTC), 3},
	{Date(-1, January, 1, 12, 15, 31, 5e8, UTC), 3},
	{Date(2012, January, 1, 12, 15, 30, 5e8, UTC), time.Second},
	{Date(2012, January, 1, 12, 15, 31, 5e8, UTC), time.Second},
	{Unix(-19012425939, 649146258), 7435029458905025217}, // 5.8*d rounds to 6*d, but .8*d+.8*d < 0 < d
}

var invalidEncodingTests = []struct {
	bytes []byte
	want  string
}{
	{[]byte{}, "Time.UnmarshalBinary: no data"},
	{[]byte{0, 2, 3}, "Time.UnmarshalBinary: unsupported version"},
	{[]byte{1, 2, 3}, "Time.UnmarshalBinary: invalid length"},
}

var notEncodableTimes = []struct {
	time Toki
	want string
}{
	{Date(0, 1, 2, 3, 4, 5, 6, FixedZone("", -1*60)), "Time.MarshalBinary: unexpected zone offset"},
	{Date(0, 1, 2, 3, 4, 5, 6, FixedZone("", -32769*60)), "Time.MarshalBinary: unexpected zone offset"},
	{Date(0, 1, 2, 3, 4, 5, 6, FixedZone("", 32768*60)), "Time.MarshalBinary: unexpected zone offset"},
}

var jsonTests = []struct {
	time Toki
	json string
}{
	{Date(9999, 4, 12, 23, 20, 50, 520*1e6, UTC), `"9999-04-12T23:20:50.52Z"`},
	{Date(1996, 12, 19, 16, 39, 57, 0, local()), `"1996-12-19T16:39:57-08:00"`},
	{Date(0, 1, 1, 0, 0, 0, 1, FixedZone("", 1*60)), `"0000-01-01T00:00:00.000000001+00:01"`},
	{Date(2020, 1, 1, 0, 0, 0, 0, FixedZone("", 23*60*60+59*60)), `"2020-01-01T00:00:00+23:59"`},
}

var subTests = []struct {
	t Toki
	u Toki
	d time.Duration
}{
	{New(), New(), time.Duration(0)},
	{Date(2009, 11, 23, 0, 0, 0, 1, UTC), Date(2009, 11, 23, 0, 0, 0, 0, UTC), time.Duration(1)},
	{Date(2009, 11, 23, 0, 0, 0, 0, UTC), Date(2009, 11, 24, 0, 0, 0, 0, UTC), -24 * time.Hour},
	{Date(2009, 11, 24, 0, 0, 0, 0, UTC), Date(2009, 11, 23, 0, 0, 0, 0, UTC), 24 * time.Hour},
	{Date(-2009, 11, 24, 0, 0, 0, 0, UTC), Date(-2009, 11, 23, 0, 0, 0, 0, UTC), 24 * time.Hour},
	{New(), Date(2109, 11, 23, 0, 0, 0, 0, UTC), time.Duration(minDuration)},
	{Date(2109, 11, 23, 0, 0, 0, 0, UTC), New(), time.Duration(maxDuration)},
	{New(), Date(-2109, 11, 23, 0, 0, 0, 0, UTC), time.Duration(maxDuration)},
	{Date(-2109, 11, 23, 0, 0, 0, 0, UTC), New(), time.Duration(minDuration)},
	{Date(2290, 1, 1, 0, 0, 0, 0, UTC), Date(2000, 1, 1, 0, 0, 0, 0, UTC), 290*365*24*time.Hour + 71*24*time.Hour},
	{Date(2300, 1, 1, 0, 0, 0, 0, UTC), Date(2000, 1, 1, 0, 0, 0, 0, UTC), time.Duration(maxDuration)},
	{Date(2000, 1, 1, 0, 0, 0, 0, UTC), Date(2290, 1, 1, 0, 0, 0, 0, UTC), -290*365*24*time.Hour - 71*24*time.Hour},
	{Date(2000, 1, 1, 0, 0, 0, 0, UTC), Date(2300, 1, 1, 0, 0, 0, 0, UTC), time.Duration(minDuration)},
	{Date(2311, 11, 26, 02, 16, 47, 63535996, UTC), Date(2019, 8, 16, 2, 29, 30, 268436582, UTC), 9223372036795099414},
}

var defaultLocTests = []struct {
	name string
	f    func(t1, t2 Toki) bool
}{
	{"After", func(t1, t2 Toki) bool { return t1.After(t2) == t2.After(t1) }},
	{"Before", func(t1, t2 Toki) bool { return t1.Before(t2) == t2.Before(t1) }},
	{"Equal", func(t1, t2 Toki) bool { return t1.Equal(t2) == t2.Equal(t1) }},
	{"Compare", func(t1, t2 Toki) bool { return t1.Compare(t2) == t2.Compare(t1) }},

	{"IsZero", func(t1, t2 Toki) bool { return t1.IsZero() == t2.IsZero() }},
	{"Date", func(t1, t2 Toki) bool {
		a1, b1, c1 := t1.Date()
		a2, b2, c2 := t2.Date()
		return a1 == a2 && b1 == b2 && c1 == c2
	}},
	{"Year", func(t1, t2 Toki) bool { return t1.Year() == t2.Year() }},
	{"Month", func(t1, t2 Toki) bool { return t1.Month() == t2.Month() }},
	{"Day", func(t1, t2 Toki) bool { return t1.Day() == t2.Day() }},
	{"Weekday", func(t1, t2 Toki) bool { return t1.Weekday() == t2.Weekday() }},
	{"ISOWeek", func(t1, t2 Toki) bool {
		a1, b1 := t1.ISOWeek()
		a2, b2 := t2.ISOWeek()
		return a1 == a2 && b1 == b2
	}},
	{"Clock", func(t1, t2 Toki) bool {
		a1, b1, c1 := t1.Clock()
		a2, b2, c2 := t2.Clock()
		return a1 == a2 && b1 == b2 && c1 == c2
	}},
	{"Hour", func(t1, t2 Toki) bool { return t1.Hour() == t2.Hour() }},
	{"Minute", func(t1, t2 Toki) bool { return t1.Minute() == t2.Minute() }},
	{"Second", func(t1, t2 Toki) bool { return t1.Second() == t2.Second() }},
	{"Nanosecond", func(t1, t2 Toki) bool { return t1.Hour() == t2.Hour() }},
	{"YearDay", func(t1, t2 Toki) bool { return t1.YearDay() == t2.YearDay() }},

	// Using Equal since Add don't modify loc using "==" will cause a fail
	{"Add", func(t1, t2 Toki) bool { return t1.Add(time.Hour).Equal(t2.Add(time.Hour)) }},
	{"Sub", func(t1, t2 Toki) bool { return t1.Sub(t2) == t2.Sub(t1) }},

	//Original caus for this test case bug 15852
	{"AddDate", func(t1, t2 Toki) bool { return t1.AddDate(1991, 9, 3) == t2.AddDate(1991, 9, 3) }},

	{"UTC", func(t1, t2 Toki) bool { return t1.UTC() == t2.UTC() }},
	{"Local", func(t1, t2 Toki) bool { return t1.Local() == t2.Local() }},
	{"In", func(t1, t2 Toki) bool { return t1.In(UTC) == t2.In(UTC) }},

	{"Local", func(t1, t2 Toki) bool { return t1.Local() == t2.Local() }},
	{"Zone", func(t1, t2 Toki) bool {
		a1, b1 := t1.Zone()
		a2, b2 := t2.Zone()
		return a1 == a2 && b1 == b2
	}},

	{"Unix", func(t1, t2 Toki) bool { return t1.Unix() == t2.Unix() }},
	{"UnixNano", func(t1, t2 Toki) bool { return t1.UnixNano() == t2.UnixNano() }},
	{"UnixMilli", func(t1, t2 Toki) bool { return t1.UnixMilli() == t2.UnixMilli() }},
	{"UnixMicro", func(t1, t2 Toki) bool { return t1.UnixMicro() == t2.UnixMicro() }},

	{"MarshalBinary", func(t1, t2 Toki) bool {
		a1, b1 := t1.MarshalBinary()
		a2, b2 := t2.MarshalBinary()
		return bytes.Equal(a1, a2) && b1 == b2
	}},
	{"GobEncode", func(t1, t2 Toki) bool {
		a1, b1 := t1.GobEncode()
		a2, b2 := t2.GobEncode()
		return bytes.Equal(a1, a2) && b1 == b2
	}},
	{"MarshalJSON", func(t1, t2 Toki) bool {
		a1, b1 := t1.MarshalJSON()
		a2, b2 := t2.MarshalJSON()
		return bytes.Equal(a1, a2) && b1 == b2
	}},
	{"MarshalText", func(t1, t2 Toki) bool {
		a1, b1 := t1.MarshalText()
		a2, b2 := t2.MarshalText()
		return bytes.Equal(a1, a2) && b1 == b2
	}},

	{"Truncate", func(t1, t2 Toki) bool { return t1.Truncate(time.Hour).Equal(t2.Truncate(time.Hour)) }},
	{"Round", func(t1, t2 Toki) bool { return t1.Round(time.Hour).Equal(t2.Round(time.Hour)) }},

	{"== Time{}", func(t1, t2 Toki) bool { return (t1 == New()) == (t2 == New()) }},
}

func dateTests() []struct {
	year, month, day, hour, min, sec, nsec int
	z                                      *Location
	unix                                   int64
} {
	return []struct {
		year, month, day, hour, min, sec, nsec int
		z                                      *Location
		unix                                   int64
	}{
		{2011, 11, 6, 1, 0, 0, 0, time.Local, 1320566400},   // 1:00:00 PDT
		{2011, 11, 6, 1, 59, 59, 0, time.Local, 1320569999}, // 1:59:59 PDT
		{2011, 11, 6, 2, 0, 0, 0, time.Local, 1320573600},   // 2:00:00 PST

		{2011, 3, 13, 1, 0, 0, 0, time.Local, 1300006800},   // 1:00:00 PST
		{2011, 3, 13, 1, 59, 59, 0, time.Local, 1300010399}, // 1:59:59 PST
		{2011, 3, 13, 3, 0, 0, 0, time.Local, 1300010400},   // 3:00:00 PDT
		{2011, 3, 13, 2, 30, 0, 0, time.Local, 1300008600},  // 2:30:00 PDT ≡ 1:30 PST
		{2012, 12, 24, 0, 0, 0, 0, time.Local, 1356336000},  // Leap year

		// Many names for Fri Nov 18 7:56:35 PST 2011
		{2011, 11, 18, 7, 56, 35, 0, time.Local, 1321631795},                 // Nov 18 7:56:35
		{2011, 11, 19, -17, 56, 35, 0, time.Local, 1321631795},               // Nov 19 -17:56:35
		{2011, 11, 17, 31, 56, 35, 0, time.Local, 1321631795},                // Nov 17 31:56:35
		{2011, 11, 18, 6, 116, 35, 0, time.Local, 1321631795},                // Nov 18 6:116:35
		{2011, 10, 49, 7, 56, 35, 0, time.Local, 1321631795},                 // Oct 49 7:56:35
		{2011, 11, 18, 7, 55, 95, 0, time.Local, 1321631795},                 // Nov 18 7:55:95
		{2011, 11, 18, 7, 56, 34, 1e9, time.Local, 1321631795},               // Nov 18 7:56:34 + 10⁹ns
		{2011, 12, -12, 7, 56, 35, 0, time.Local, 1321631795},                // Dec -21 7:56:35
		{2012, 1, -43, 7, 56, 35, 0, time.Local, 1321631795},                 // Jan -52 7:56:35 2012
		{2012, int(January - 2), 18, 7, 56, 35, 0, time.Local, 1321631795},   // (Jan-2) 18 7:56:35 2012
		{2010, int(December + 11), 18, 7, 56, 35, 0, time.Local, 1321631795}, // (Dec+11) 18 7:56:35 2010
	}
}

func local() *time.Location {
	ForceUSPacificForTesting()
	return time.Local
}

func same(t Toki, u *parsedTime) bool {
	// Check aggregates.
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	name, offset := t.Zone()
	if year != u.Year || month != u.Month || day != u.Day ||
		hour != u.Hour || min != u.Minute || sec != u.Second ||
		name != u.Zone || offset != u.ZoneOffset {
		return false
	}
	// Check individual entries.
	return t.Year() == u.Year &&
		t.Month() == u.Month &&
		t.Day() == u.Day &&
		t.Hour() == u.Hour &&
		t.Minute() == u.Minute &&
		t.Second() == u.Second &&
		t.Nanosecond() == u.Nanosecond &&
		t.Weekday() == u.Weekday
}

func equalTimeAndZone(a, b Toki) bool {
	aname, aoffset := a.Zone()
	bname, boffset := b.Zone()
	return a.Equal(b) && aoffset == boffset && aname == bname
}

// abs returns the absolute time stored in t, as seconds and nanoseconds.
func abs(t Toki) (sec, nsec int64) {
	unix := t.Unix()
	nano := t.Nanosecond()
	return unix + unixToZero, int64(nano)
}

// absString returns abs as a decimal string.
func absString(t Toki) string {
	sec, nsec := abs(t)
	if sec < 0 {
		sec = -sec
		nsec = -nsec
		if nsec < 0 {
			nsec += 1e9
			sec--
		}
		return fmt.Sprintf("-%d%09d", sec, nsec)
	}
	return fmt.Sprintf("%d%09d", sec, nsec)
}

func TestSecondsToUTC(t *testing.T) {
	for _, tt := range utctests {
		sec := tt.seconds
		golden := &tt.golden
		tm := Unix(sec, 0).UTC()
		newsec := tm.Unix()
		if newsec != sec {
			t.Errorf("SecondsToUTC(%d).Seconds() = %d", sec, newsec)
		}
		if !same(tm, golden) {
			t.Errorf("SecondsToUTC(%d):  // %#v", sec, tm)
			t.Errorf("  want=%+v", *golden)
			t.Errorf("  have=%v", tm.Format(RFC3339+" MST"))
		}
	}
}

func TestNanosecondsToUTC(t *testing.T) {
	for _, tt := range nanoutctests {
		golden := &tt.golden
		nsec := tt.seconds*1e9 + int64(golden.Nanosecond)
		tm := Unix(0, nsec).UTC()
		newnsec := tm.Unix()*1e9 + int64(tm.Nanosecond())
		if newnsec != nsec {
			t.Errorf("NanosecondsToUTC(%d).Nanoseconds() = %d", nsec, newnsec)
		}
		if !same(tm, golden) {
			t.Errorf("NanosecondsToUTC(%d):", nsec)
			t.Errorf("  want=%+v", *golden)
			t.Errorf("  have=%+v", tm.Format(RFC3339+" MST"))
		}
	}
}

func TestSecondsToLocalTime(t *testing.T) {
	for _, test := range localtests {
		sec := test.seconds
		golden := &test.golden
		tm := Unix(sec, 0)
		newsec := tm.Unix()
		if newsec != sec {
			t.Errorf("SecondsToLocalTime(%d).Seconds() = %d", sec, newsec)
		}
		if !same(tm, golden) {
			t.Errorf("SecondsToLocalTime(%d):", sec)
			t.Errorf("  want=%+v", *golden)
			t.Errorf("  have=%+v", tm.Format(RFC3339+" MST"))
		}
	}
}

func TestNanosecondsToLocalTime(t *testing.T) {
	for _, test := range nanolocaltests {
		golden := &test.golden
		nsec := test.seconds*1e9 + int64(golden.Nanosecond)
		tm := Unix(0, nsec)
		newnsec := tm.Unix()*1e9 + int64(tm.Nanosecond())
		if newnsec != nsec {
			t.Errorf("NanosecondsToLocalTime(%d).Seconds() = %d", nsec, newnsec)
		}
		if !same(tm, golden) {
			t.Errorf("NanosecondsToLocalTime(%d):", nsec)
			t.Errorf("  want=%+v", *golden)
			t.Errorf("  have=%+v", tm.Format(RFC3339+" MST"))
		}
	}
}

func TestSecondsToUTCAndBack(t *testing.T) {
	f := func(sec int64) bool { return Unix(sec, 0).UTC().Unix() == sec }
	f32 := func(sec int32) bool { return f(int64(sec)) }
	cfg := &quick.Config{MaxCount: 10000}

	// Try a reasonable date first, then the huge ones.
	if err := quick.Check(f32, cfg); err != nil {
		t.Fatal(err)
	}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatal(err)
	}
}

func TestNanosecondsToUTCAndBack(t *testing.T) {
	f := func(nsec int64) bool {
		t := Unix(0, nsec).UTC()
		ns := t.Unix()*1e9 + int64(t.Nanosecond())
		return ns == nsec
	}
	f32 := func(nsec int32) bool { return f(int64(nsec)) }
	cfg := &quick.Config{MaxCount: 10000}

	// Try a small date first, then the large ones. (The span is only a few hundred years
	// for nanoseconds in an int64.)
	if err := quick.Check(f32, cfg); err != nil {
		t.Fatal(err)
	}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatal(err)
	}
}

func TestUnixMilli(t *testing.T) {
	f := func(msec int64) bool {
		t := UnixMilli(msec)
		return t.UnixMilli() == msec
	}
	cfg := &quick.Config{MaxCount: 10000}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatal(err)
	}
}

func TestUnixMicro(t *testing.T) {
	f := func(usec int64) bool {
		t := UnixMicro(usec)
		return t.UnixMicro() == usec
	}
	cfg := &quick.Config{MaxCount: 10000}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatal(err)
	}
}

func TestTruncateRound(t *testing.T) {
	var (
		bsec  = new(big.Int)
		bnsec = new(big.Int)
		bd    = new(big.Int)
		bt    = new(big.Int)
		br    = new(big.Int)
		bq    = new(big.Int)
		b1e9  = new(big.Int)
	)

	b1e9.SetInt64(1e9)

	testOne := func(ti, tns, di int64) bool {
		t.Helper()

		t0 := Unix(ti, int64(tns)).UTC()
		d := time.Duration(di)
		if d < 0 {
			d = -d
		}
		if d <= 0 {
			d = 1
		}

		// Compute bt = absolute nanoseconds.
		sec, nsec := abs(t0)
		bsec.SetInt64(sec)
		bnsec.SetInt64(nsec)
		bt.Mul(bsec, b1e9)
		bt.Add(bt, bnsec)

		// Compute quotient and remainder mod d.
		bd.SetInt64(int64(d))
		bq.DivMod(bt, bd, br)

		// To truncate, subtract remainder.
		// br is < d, so it fits in an int64.
		r := br.Int64()
		t1 := t0.Add(-time.Duration(r))

		// Check that time.Truncate works.
		if trunc := t0.Truncate(d); trunc != t1 {
			t.Errorf("Time.Truncate(%s, %s) = %s, want %s\n"+
				"%v trunc %v =\n%v want\n%v",
				t0.Format(time.RFC3339Nano), d, trunc, t1.Format(time.RFC3339Nano),
				absString(t0), int64(d), absString(trunc), absString(t1))
			return false
		}

		// To round, add d back if remainder r > d/2 or r == exactly d/2.
		// The commented out code would round half to even instead of up,
		// but that makes it time-zone dependent, which is a bit strange.
		if r > int64(d)/2 || r+r == int64(d) /*&& bq.Bit(0) == 1*/ {
			t1 = t1.Add(time.Duration(d))
		}

		// Check that time.Round works.
		if rnd := t0.Round(d); rnd != t1 {
			t.Errorf("Time.Round(%s, %s) = %s, want %s\n"+
				"%v round %v =\n%v want\n%v",
				t0.Format(time.RFC3339Nano), d, rnd, t1.Format(time.RFC3339Nano),
				absString(t0), int64(d), absString(rnd), absString(t1))
			return false
		}
		return true
	}

	// manual test cases
	for _, tt := range truncateRoundTests {
		testOne(tt.t.Unix(), int64(tt.t.Nanosecond()), int64(tt.d))
	}

	// exhaustive near 0
	for i := 0; i < 100; i++ {
		for j := 1; j < 100; j++ {
			testOne(unixToZero, int64(i), int64(j))
			testOne(unixToZero, -int64(i), int64(j))
			if t.Failed() {
				return
			}
		}
	}

	if t.Failed() {
		return
	}

	// randomly generated test cases
	cfg := &quick.Config{MaxCount: 100000}
	if testing.Short() {
		cfg.MaxCount = 1000
	}

	// divisors of Second
	f1 := func(ti int64, tns int32, logdi int32) bool {
		d := time.Duration(1)
		a, b := uint(logdi%9), (logdi>>16)%9
		d <<= a
		for i := 0; i < int(b); i++ {
			d *= 5
		}

		// Make room for unix ↔ internal conversion.
		// We don't care about behavior too close to ± 2^63 Unix seconds.
		// It is full of wraparounds but will never happen in a reasonable program.
		// (Or maybe not? See go.dev/issue/20678. In any event, they're not handled today.)
		ti >>= 1

		return testOne(ti, int64(tns), int64(d))
	}
	quick.Check(f1, cfg)

	// multiples of Second
	f2 := func(ti int64, tns int32, di int32) bool {
		d := time.Duration(di) * time.Second
		if d < 0 {
			d = -d
		}
		ti >>= 1 // see comment in f1
		return testOne(ti, int64(tns), int64(d))
	}
	quick.Check(f2, cfg)

	// halfway cases
	f3 := func(tns, di int64) bool {
		di &= 0xfffffffe
		if di == 0 {
			di = 2
		}
		tns -= tns % di
		if tns < 0 {
			tns += di / 2
		} else {
			tns -= di / 2
		}
		return testOne(0, tns, di)
	}
	quick.Check(f3, cfg)

	// full generality
	f4 := func(ti int64, tns int32, di int64) bool {
		ti >>= 1 // see comment in f1
		return testOne(ti, int64(tns), di)
	}
	quick.Check(f4, cfg)
}

func TestISOWeek(t *testing.T) {
	// Selected dates and corner cases
	for _, wt := range isoWeekTests {
		dt := Date(wt.year, Month(wt.month), wt.day, 0, 0, 0, 0, UTC)
		y, w := dt.ISOWeek()
		if w != wt.wex || y != wt.yex {
			t.Errorf("got %d/%d; expected %d/%d for %d-%02d-%02d",
				y, w, wt.yex, wt.wex, wt.year, wt.month, wt.day)
		}
	}

	// The only real invariant: Jan 04 is in week 1
	for year := 1950; year < 2100; year++ {
		if y, w := Date(year, January, 4, 0, 0, 0, 0, UTC).ISOWeek(); y != year || w != 1 {
			t.Errorf("got %d/%d; expected %d/1 for Jan 04", y, w, year)
		}
	}
}

func TestYearDay(t *testing.T) {
	for i, loc := range yearDayLocations {
		for _, ydt := range yearDayTests {
			dt := Date(ydt.year, Month(ydt.month), ydt.day, 0, 0, 0, 0, loc)
			yday := dt.YearDay()
			if yday != ydt.yday {
				t.Errorf("Date(%d-%02d-%02d in %v).YearDay() = %d, want %d",
					ydt.year, ydt.month, ydt.day, loc, yday, ydt.yday)
				continue
			}

			if ydt.year < 0 || ydt.year > 9999 {
				continue
			}
			f := fmt.Sprintf("%04d-%02d-%02d %03d %+.2d00",
				ydt.year, ydt.month, ydt.day, ydt.yday, (i-2)*4)
			dt1, err := Parse("2006-01-02 002 -0700", f)
			if err != nil {
				t.Errorf(`Parse("2006-01-02 002 -0700", %q): %v`, f, err)
				continue
			}
			if !dt1.Equal(dt) {
				t.Errorf(`Parse("2006-01-02 002 -0700", %q) = %v, want %v`, f, dt1, dt)
			}
		}
	}
}

func TestDate(t *testing.T) {
	for _, tt := range dateTests() {
		time := Date(tt.year, Month(tt.month), tt.day, tt.hour, tt.min, tt.sec, tt.nsec, tt.z)
		want := Unix(tt.unix, 0)
		if !time.Equal(want) {
			t.Errorf("Date(%d, %d, %d, %d, %d, %d, %d, %s) = %v, want %v",
				tt.year, tt.month, tt.day, tt.hour, tt.min, tt.sec, tt.nsec, tt.z,
				time, want)
		}
	}
}

func TestAddDate(t *testing.T) {
	t0 := Date(2011, 11, 18, 7, 56, 35, 0, UTC)
	t1 := Date(2016, 3, 19, 7, 56, 35, 0, UTC)
	for _, at := range addDateTests {
		time := t0.AddDate(at.years, at.months, at.days)
		if !time.Equal(t1) {
			t.Errorf("AddDate(%d, %d, %d) = %v, want %v",
				at.years, at.months, at.days,
				time, t1)
		}
	}
}

func TestDaysIn(t *testing.T) {
	// The daysIn function is not exported.
	// Test the daysIn function via the `var DaysIn = daysIn`
	// statement in the internal_test.go file.
	for _, tt := range daysInTests {
		di := DaysIn(Month(tt.month), tt.year)
		if di != tt.di {
			t.Errorf("got %d; expected %d for %d-%02d",
				di, tt.di, tt.year, tt.month)
		}
	}
}

func TestAddToExactSecond(t *testing.T) {
	// Add an amount to the current time to round it up to the next exact second.
	// This test checks that the nsec field still lies within the range [0, 999999999].
	t1 := Now()
	t2 := t1.Add(time.Second - time.Duration(t1.Nanosecond()))
	sec := (t1.Second() + 1) % 60
	if t2.Second() != sec || t2.Nanosecond() != 0 {
		t.Errorf("sec = %d, nsec = %d, want sec = %d, nsec = 0", t2.Second(), t2.Nanosecond(), sec)
	}
}

func TestTimeGob(t *testing.T) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	dec := gob.NewDecoder(&b)
	for _, tt := range gobTests {
		gobtt := New()
		if err := enc.Encode(&tt); err != nil {
			t.Errorf("%v gob Encode error = %q, want nil", tt, err)
		} else if err := dec.Decode(&gobtt); err != nil {
			t.Errorf("%v gob Decode error = %q, want nil", tt, err)
		} else if !equalTimeAndZone(gobtt, tt) {
			t.Errorf("Decoded time = %v, want %v", gobtt, tt)
		}
		b.Reset()
	}
}

func TestInvalidTimeGob(t *testing.T) {
	for _, tt := range invalidEncodingTests {
		ignored := New()
		err := ignored.GobDecode(tt.bytes)
		if err == nil || err.Error() != tt.want {
			t.Errorf("time.GobDecode(%#v) error = %v, want %v", tt.bytes, err, tt.want)
		}
		err = ignored.UnmarshalBinary(tt.bytes)
		if err == nil || err.Error() != tt.want {
			t.Errorf("time.UnmarshalBinary(%#v) error = %v, want %v", tt.bytes, err, tt.want)
		}
	}
}

func TestNotGobEncodableTime(t *testing.T) {
	for _, tt := range notEncodableTimes {
		_, err := tt.time.GobEncode()
		if err == nil || err.Error() != tt.want {
			t.Errorf("%v GobEncode error = %v, want %v", tt.time, err, tt.want)
		}
		_, err = tt.time.MarshalBinary()
		if err == nil || err.Error() != tt.want {
			t.Errorf("%v MarshalBinary error = %v, want %v", tt.time, err, tt.want)
		}
	}
}

func TestTimeJSON(t *testing.T) {
	for _, tt := range jsonTests {
		jsonTime := New()

		if jsonBytes, err := json.Marshal(tt.time); err != nil {
			t.Errorf("%v json.Marshal error = %v, want nil", tt.time, err)
		} else if string(jsonBytes) != tt.json {
			t.Errorf("%v JSON = %#q, want %#q", tt.time, string(jsonBytes), tt.json)
		} else if err = json.Unmarshal(jsonBytes, &jsonTime); err != nil {
			t.Errorf("%v json.Unmarshal error = %v, want nil", tt.time, err)
		} else if !equalTimeAndZone(jsonTime, tt.time) {
			t.Errorf("Unmarshaled time = %v, want %v", jsonTime, tt.time)
		}
	}
}

func TestUnmarshalInvalidTimes(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{`{}`, "Time.UnmarshalJSON: input is not a JSON string"},
		{`[]`, "Time.UnmarshalJSON: input is not a JSON string"},
		{`"2000-01-01T1:12:34Z"`, `<nil>`},
		{`"2000-01-01T00:00:00,000Z"`, `<nil>`},
		{`"2000-01-01T00:00:00+24:00"`, `<nil>`},
		{`"2000-01-01T00:00:00+00:60"`, `<nil>`},
		{`"2000-01-01T00:00:00+123:45"`, `parsing time "2000-01-01T00:00:00+123:45" as "2006-01-02T15:04:05Z07:00": cannot parse "+123:45" as "Z07:00"`},
	}

	for _, tt := range tests {
		ts := New()

		want := tt.want
		err := json.Unmarshal([]byte(tt.in), &ts)
		if fmt.Sprint(err) != want {
			t.Errorf("Time.UnmarshalJSON(%s) = %v, want %v", tt.in, err, want)
		}

		if strings.HasPrefix(tt.in, `"`) && strings.HasSuffix(tt.in, `"`) {
			err = ts.UnmarshalText([]byte(strings.Trim(tt.in, `"`)))
			if fmt.Sprint(err) != want {
				t.Errorf("Time.UnmarshalText(%s) = %v, want %v", tt.in, err, want)
			}
		}
	}
}

func TestMarshalInvalidTimes(t *testing.T) {
	tests := []struct {
		time Toki
		want string
	}{
		{Date(10000, 1, 1, 0, 0, 0, 0, UTC), "Time.MarshalJSON: year outside of range [0,9999]"},
		{Date(-998, 1, 1, 0, 0, 0, 0, UTC).Add(-time.Second), "Time.MarshalJSON: year outside of range [0,9999]"},
		{Date(0, 1, 1, 0, 0, 0, 0, UTC).Add(-time.Nanosecond), "Time.MarshalJSON: year outside of range [0,9999]"},
		{Date(2020, 1, 1, 0, 0, 0, 0, FixedZone("", 24*60*60)), "Time.MarshalJSON: timezone hour outside of range [0,23]"},
		{Date(2020, 1, 1, 0, 0, 0, 0, FixedZone("", 123*60*60)), "Time.MarshalJSON: timezone hour outside of range [0,23]"},
	}

	for _, tt := range tests {
		want := tt.want
		b, err := tt.time.MarshalJSON()
		switch {
		case b != nil:
			t.Errorf("(%v).MarshalText() = %q, want nil", tt.time, b)
		case err == nil || err.Error() != want:
			t.Errorf("(%v).MarshalJSON() error = %v, want %v", tt.time, err, want)
		}

		want = strings.ReplaceAll(tt.want, "JSON", "Text")
		b, err = tt.time.MarshalText()
		switch {
		case b != nil:
			t.Errorf("(%v).MarshalText() = %q, want nil", tt.time, b)
		case err == nil || err.Error() != want:
			t.Errorf("(%v).MarshalText() error = %v, want %v", tt.time, err, want)
		}
	}
}

func TestSub(t *testing.T) {
	for i, st := range subTests {
		got := st.t.Sub(st.u)
		if got != st.d {
			t.Errorf("#%d: Sub(%v, %v): got %v; want %v", i, st.t, st.u, got, st.d)
		}
	}
}

func TestDefaultLoc(t *testing.T) {
	// Verify that all of Time's methods behave identically if loc is set to
	// nil or UTC.
	for _, tt := range defaultLocTests {
		t1 := New()
		t2 := New().UTC()
		if !tt.f(t1, t2) {
			t.Errorf("Toki{} and Toki{}.UTC() behave differently for %s", tt.name)
		}
	}
}

func TestMarshalBinaryZeroTime(t *testing.T) {
	t0 := New()
	enc, err := t0.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	t1 := Now() // not zero
	if err := t1.UnmarshalBinary(enc); err != nil {
		t.Fatal(err)
	}
	if t1 != t0 {
		t.Errorf("t0=%#v\nt1=%#v\nwant identical structures", t0, t1)
	}
}

func TestMarshalBinaryVersion2(t *testing.T) {
	t0, err := Parse(RFC3339, "1880-01-01T00:00:00Z")
	if err != nil {
		t.Errorf("Failed to parse time, error = %v", err)
	}
	loc, err := time.LoadLocation("US/Eastern")
	if err != nil {
		t.Errorf("Failed to load location, error = %v", err)
	}
	t1 := t0.In(loc)
	b, err := t1.MarshalBinary()
	if err != nil {
		t.Errorf("Failed to Marshal, error = %v", err)
	}

	t2 := New()
	err = t2.UnmarshalBinary(b)
	if err != nil {
		t.Errorf("Failed to Unmarshal, error = %v", err)
	}

	if !(t0.Equal(t1) && t1.Equal(t2)) {
		if !t0.Equal(t1) {
			t.Errorf("The result t1: %+v after Marshal is not matched original t0: %+v", t1, t0)
		}
		if !t1.Equal(t2) {
			t.Errorf("The result t2: %+v after Unmarshal is not matched original t1: %+v", t2, t1)
		}
	}
}

func TestUnmarshalTextAllocations(t *testing.T) {
	in := []byte(testdataRFC3339UTC) // short enough to be stack allocated
	if allocs := testing.AllocsPerRun(100, func() {
		tt := New()
		tt.UnmarshalText(in)
	}); allocs != 0 {
		t.Errorf("got %v allocs, want 0 allocs", allocs)
	}
}

// Issue 17720: Zero value of time.Month fails to print
func TestZeroMonthString(t *testing.T) {
	if got, want := Month(0).String(), "%!Month(0)"; got != want {
		t.Errorf("zero month = %q; want %q", got, want)
	}
}

// Issue 24692: Out of range weekday panics
func TestWeekdayString(t *testing.T) {
	if got, want := Weekday(Tuesday).String(), "Tuesday"; got != want {
		t.Errorf("Tuesday weekday = %q; want %q", got, want)
	}
	if got, want := Weekday(14).String(), "%!Weekday(14)"; got != want {
		t.Errorf("14th weekday = %q; want %q", got, want)
	}
}

func TestTimeIsDST(t *testing.T) {
	tzWithDST, err := time.LoadLocation("Australia/Sydney")
	if err != nil {
		t.Fatalf("could not load tz 'Australia/Sydney': %v", err)
	}
	tzWithoutDST, err := time.LoadLocation("Australia/Brisbane")
	if err != nil {
		t.Fatalf("could not load tz 'Australia/Brisbane': %v", err)
	}
	tzFixed := FixedZone("FIXED_TIME", 12345)

	tests := [...]struct {
		time Toki
		want bool
	}{
		0: {Date(2009, 1, 1, 12, 0, 0, 0, UTC), false},
		1: {Date(2009, 6, 1, 12, 0, 0, 0, UTC), false},
		2: {Date(2009, 1, 1, 12, 0, 0, 0, tzWithDST), true},
		3: {Date(2009, 6, 1, 12, 0, 0, 0, tzWithDST), false},
		4: {Date(2009, 1, 1, 12, 0, 0, 0, tzWithoutDST), false},
		5: {Date(2009, 6, 1, 12, 0, 0, 0, tzWithoutDST), false},
		6: {Date(2009, 1, 1, 12, 0, 0, 0, tzFixed), false},
		7: {Date(2009, 6, 1, 12, 0, 0, 0, tzFixed), false},
	}

	for i, tt := range tests {
		got := tt.time.IsDST()
		if got != tt.want {
			t.Errorf("#%d:: (%#v).IsDST()=%t, want %t", i, tt.time.Format(RFC3339), got, tt.want)
		}
	}
}

func TestTimeAddSecOverflow(t *testing.T) {
	// Test it with positive delta.
	var maxInt64 int64 = 1<<63 - 1
	timeExt := maxInt64 - unixToInternal - 50
	notMonoTime := Unix(timeExt, 0)
	for i := int64(0); i < 100; i++ {
		sec := notMonoTime.Unix()
		notMonoTime = notMonoTime.Add(time.Duration(i * 1e9))
		if newSec := notMonoTime.Unix(); newSec != sec+i && newSec+unixToInternal != maxInt64 {
			t.Fatalf("time ext: %d overflows with positive delta, overflow threshold: %d", newSec, maxInt64)
		}
	}
}

// Issue 49284: time: ParseInLocation incorrectly because of Daylight Saving Time
func TestTimeWithZoneTransition(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}

	tests := [...]struct {
		give Toki
		want Toki
	}{
		// 14 Apr 1991 - Daylight Saving Time Started
		// When time of "Asia/Shanghai" was about to reach
		// Sunday, 14 April 1991, 02:00:00 clocks were turned forward 1 hour to
		// Sunday, 14 April 1991, 03:00:00 local daylight time instead.
		// The UTC time was 13 April 1991, 18:00:00
		0: {Date(1991, April, 13, 17, 50, 0, 0, loc), Date(1991, April, 13, 9, 50, 0, 0, UTC)},
		1: {Date(1991, April, 13, 18, 0, 0, 0, loc), Date(1991, April, 13, 10, 0, 0, 0, UTC)},
		2: {Date(1991, April, 14, 1, 50, 0, 0, loc), Date(1991, April, 13, 17, 50, 0, 0, UTC)},
		3: {Date(1991, April, 14, 3, 0, 0, 0, loc), Date(1991, April, 13, 18, 0, 0, 0, UTC)},

		// 15 Sep 1991 - Daylight Saving Time Ended
		// When local daylight time of "Asia/Shanghai" was about to reach
		// Sunday, 15 September 1991, 02:00:00 clocks were turned backward 1 hour to
		// Sunday, 15 September 1991, 01:00:00 local standard time instead.
		// The UTC time was 14 September 1991, 17:00:00
		4: {Date(1991, September, 14, 16, 50, 0, 0, loc), Date(1991, September, 14, 7, 50, 0, 0, UTC)},
		5: {Date(1991, September, 14, 17, 0, 0, 0, loc), Date(1991, September, 14, 8, 0, 0, 0, UTC)},
		6: {Date(1991, September, 15, 0, 50, 0, 0, loc), Date(1991, September, 14, 15, 50, 0, 0, UTC)},
		7: {Date(1991, September, 15, 2, 00, 0, 0, loc), Date(1991, September, 14, 18, 00, 0, 0, UTC)},
	}

	for i, tt := range tests {
		if !tt.give.Equal(tt.want) {
			t.Errorf("#%d:: %#v is not equal to %#v", i, tt.give.Format(RFC3339), tt.want.Format(RFC3339))
		}
	}
}

func TestZoneBounds(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}

	// The ZoneBounds of a UTC location would just return two zero Time.
	for _, test := range utctests {
		sec := test.seconds
		golden := &test.golden
		tm := Unix(sec, 0).UTC()
		start, end := tm.ZoneBounds()
		if !(start.IsZero() && end.IsZero()) {
			t.Errorf("ZoneBounds of %+v expects two zero Time, got:\n  start=%v\n  end=%v", *golden, start, end)
		}
	}

	// If the zone begins at the beginning of time, start will be returned as a zero Time.
	// Use math.MinInt32 to avoid overflow of int arguments on 32-bit systems.
	beginTime := Date(math.MinInt32, January, 1, 0, 0, 0, 0, loc)
	start, end := beginTime.ZoneBounds()
	if !start.IsZero() || end.IsZero() {
		t.Errorf("ZoneBounds of %v expects start is zero Time, got:\n  start=%v\n  end=%v", beginTime, start, end)
	}

	// If the zone goes on forever, end will be returned as a zero Time.
	// Use math.MaxInt32 to avoid overflow of int arguments on 32-bit systems.
	foreverTime := Date(math.MaxInt32, January, 1, 0, 0, 0, 0, loc)
	start, end = foreverTime.ZoneBounds()
	if start.IsZero() || !end.IsZero() {
		t.Errorf("ZoneBounds of %v expects end is zero Time, got:\n  start=%v\n  end=%v", foreverTime, start, end)
	}

	// Check some real-world cases to make sure we're getting the right bounds.
	boundOne := Date(1990, September, 16, 1, 0, 0, 0, loc)
	boundTwo := Date(1991, April, 14, 3, 0, 0, 0, loc)
	boundThree := Date(1991, September, 15, 1, 0, 0, 0, loc)
	makeLocalTime := func(sec int64) Toki { return Unix(sec, 0) }
	realTests := [...]struct {
		giveTime  Toki
		wantStart Toki
		wantEnd   Toki
	}{
		// The ZoneBounds of "Asia/Shanghai" Daylight Saving Time
		0: {Date(1991, April, 13, 17, 50, 0, 0, loc), boundOne, boundTwo},
		1: {Date(1991, April, 13, 18, 0, 0, 0, loc), boundOne, boundTwo},
		2: {Date(1991, April, 14, 1, 50, 0, 0, loc), boundOne, boundTwo},
		3: {boundTwo, boundTwo, boundThree},
		4: {Date(1991, September, 14, 16, 50, 0, 0, loc), boundTwo, boundThree},
		5: {Date(1991, September, 14, 17, 0, 0, 0, loc), boundTwo, boundThree},
		6: {Date(1991, September, 15, 0, 50, 0, 0, loc), boundTwo, boundThree},

		// The ZoneBounds of a "Asia/Shanghai" after the last transition (Standard Time)
		7:  {boundThree, boundThree, New()},
		8:  {Date(1991, December, 15, 1, 50, 0, 0, loc), boundThree, New()},
		9:  {Date(1992, April, 13, 17, 50, 0, 0, loc), boundThree, New()},
		10: {Date(1992, April, 13, 18, 0, 0, 0, loc), boundThree, New()},
		11: {Date(1992, April, 14, 1, 50, 0, 0, loc), boundThree, New()},
		12: {Date(1992, September, 14, 16, 50, 0, 0, loc), boundThree, New()},
		13: {Date(1992, September, 14, 17, 0, 0, 0, loc), boundThree, New()},
		14: {Date(1992, September, 15, 0, 50, 0, 0, loc), boundThree, New()},

		// The ZoneBounds of a local time would return two local Time.
		// Note: We preloaded "America/Los_Angeles" as time.Local for testing
		15: {makeLocalTime(0), makeLocalTime(-5756400), makeLocalTime(9972000)},
		16: {makeLocalTime(1221681866), makeLocalTime(1205056800), makeLocalTime(1225616400)},
		17: {makeLocalTime(2152173599), makeLocalTime(2145916800), makeLocalTime(2152173600)},
		18: {makeLocalTime(2152173600), makeLocalTime(2152173600), makeLocalTime(2172733200)},
		19: {makeLocalTime(2152173601), makeLocalTime(2152173600), makeLocalTime(2172733200)},
		20: {makeLocalTime(2159200800), makeLocalTime(2152173600), makeLocalTime(2172733200)},
		21: {makeLocalTime(2172733199), makeLocalTime(2152173600), makeLocalTime(2172733200)},
		22: {makeLocalTime(2172733200), makeLocalTime(2172733200), makeLocalTime(2177452800)},
	}
	for i, tt := range realTests {
		start, end := tt.giveTime.ZoneBounds()
		if !start.Equal(tt.wantStart) || !end.Equal(tt.wantEnd) {
			t.Errorf("#%d:: ZoneBounds of %v expects right bounds:\n  got start=%v\n  want start=%v\n  got end=%v\n  want end=%v",
				i, tt.giveTime, start, tt.wantStart, end, tt.wantEnd)
		}
	}
}
