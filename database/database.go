package database

import (
	"github.com/jinzhu/gorm"
	"go-notify/auth"
	"go-notify/model"
	"os"
	"path/filepath"
)

type Database struct {
	DB *gorm.DB
}

const (
	defaultUser     = "admin"
	defaultPass     = "admin"
	defaultStrength = 10
)

func (d *Database) Close() {
	d.DB.Close()
}

// sqlite3的目录创建
func createDirectory(dataPath string) {
	if _, err := os.Stat(filepath.Dir(dataPath)); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(dataPath), 0o777); err != nil {
			panic(err)
		}
	}
}

func prepareBlobColumn(db *gorm.DB) error {
	for _, target := range []struct {
		Table  interface{}
		Column string
	}{
		{model.Message{}, "extras"},
		{model.PluginConf{}, "config"},
		{model.PluginConf{}, "storage"},
	} {
		if err := db.Model(target.Table).ModifyColumn(target.Column, "").Error; err != nil {
			return err
		}
	}
	return nil
}

func NewDatabase(dataPath string) (*Database, error) {
	createDirectory(dataPath)
	db, err := gorm.Open("sqlite3", dataPath)
	if err != nil {
		panic("failed to connect database")
	}
	db.DB().SetMaxIdleConns(10)
	db.DB().SetMaxOpenConns(100)
	if err := db.AutoMigrate(new(model.User), new(model.Application), new(model.Message), new(model.Client), new(model.PluginConf)).Error; err != nil {
		return nil, err
	}

	if err := prepareBlobColumn(db); err != nil {
		return nil, err
	}

	userCount := 0
	db.Find(new(model.User)).Count(&userCount)
	if userCount == 0 {
		db.Create(&model.User{Name: defaultUser, Pass: auth.CreatePassword(defaultPass, defaultStrength), Admin: true})
	}

	return &Database{DB: db}, nil
}
