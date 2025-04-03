package store

import "time"

type ValueType int

const (
	TypeString ValueType = iota
	TypeList
	TypeSet
)

type Entry struct {
	Type      ValueType
	StringVal string
	ListVal   []string
	SetVal    map[string]struct{}
	ExpiresAt int64
}

func (e *Entry) IsExpired() bool {
	return e.ExpiresAt > 0 && time.Now().UnixMilli() > e.ExpiresAt
}
