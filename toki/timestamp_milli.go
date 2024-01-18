package toki

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"time"
)

type TimestampMilli struct {
	time.Time
}

func (t TimestampMilli) MarshalJSON() ([]byte, error) {
	return t.MarshalText()
}

func (t TimestampMilli) MarshalText() ([]byte, error) {
	buf := new(bytes.Buffer)
	i := t.Time.UnixMilli()
	if err := binary.Write(buf, binary.BigEndian, i); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (t *TimestampMilli) UnmarshalJSON(data []byte) error {
	s := string(data)

	if s == "null" {
		return nil
	}

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err

	}
	t.Time = time.UnixMilli(i)
	return nil
}

func (t *TimestampMilli) UnmarshalText(data []byte) error {
	s := string(data)

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err

	}
	t.Time = time.UnixMilli(i)
	return nil
}

func NowTimeStampMilli() TimestampMilli {
	ts := TimestampMilli{
		Time: time.Now(),
	}
	return ts
}
