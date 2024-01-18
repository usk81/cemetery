package toki

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"time"
)

type Timestamp struct {
	Toki
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return t.MarshalText()
}

func (t Timestamp) MarshalText() ([]byte, error) {
	buf := new(bytes.Buffer)
	i := t.Time.Unix()
	if err := binary.Write(buf, binary.BigEndian, i); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	s := string(data)

	if s == "null" {
		return nil
	}

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err

	}
	t.Time = time.Unix(i, 0)
	return nil
}

func (t *Timestamp) UnmarshalText(data []byte) error {
	s := string(data)

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err

	}
	t.Time = time.Unix(i, 0)
	return nil
}

func NowTimeStamp() Timestamp {
	ts := Timestamp{
		Toki{
			layout: LayoutTimestamp,
			Time:   time.Now(),
		},
	}
	return ts
}
