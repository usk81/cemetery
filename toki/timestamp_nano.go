package toki

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"time"
)

type TimestampNano struct {
	time.Time
}

func (t TimestampNano) MarshalJSON() ([]byte, error) {
	return t.MarshalText()
}

func (t TimestampNano) MarshalText() ([]byte, error) {
	buf := new(bytes.Buffer)
	i := t.Time.UnixNano()
	if err := binary.Write(buf, binary.BigEndian, i); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (t *TimestampNano) UnmarshalJSON(data []byte) error {
	s := string(data)

	if s == "null" {
		return nil
	}

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err

	}
	t.Time = time.Unix(0, i)
	return nil
}

func (t *TimestampNano) UnmarshalText(data []byte) error {
	s := string(data)

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err

	}
	t.Time = time.Unix(0, i)
	return nil
}

func NowTimeStampNano() TimestampNano {
	ts := TimestampNano{
		Time: time.Now(),
	}
	return ts
}
