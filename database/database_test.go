package database

import (
	"testing"
)

func TestDatabase(t *testing.T) {
	_, err := NewGormDatabase("data/go-notify.db")
	if err != nil {
		panic(err)
	}
}
