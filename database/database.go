package database

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // enable the sqlite3 dialect.
	"go-notify/auth"
	"go-notify/model"
	"os"
	"path/filepath"
)

type GormDatabase struct {
	DB *gorm.DB
}

const (
	defaultUser     = "admin"
	defaultPass     = "admin"
	defaultStrength = 10
)

func (d *GormDatabase) Close() {
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

// 手动创建 app_users 表  sqlite3 or gorm bug?
func initAppUsersTable(db *gorm.DB) error {
	// 执行创建表的 SQL
	createTableSQL := `
	PRAGMA foreign_keys = ON;
	CREATE TABLE app_users (
		app_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,  -- 对应 autoCreateTime
		deleted_at DATETIME,
		PRIMARY KEY (app_id, user_id),
		FOREIGN KEY (app_id) REFERENCES applications(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`
	if err := db.Exec(createTableSQL).Error; err != nil {
		return err
	}
	return nil
}

func NewGormDatabase(dataPath string) (*GormDatabase, error) {
	createDirectory(dataPath)
	db, err := gorm.Open("sqlite3", dataPath)
	if err != nil {
		panic("failed to connect GormDatabase")
	}
	db.DB().SetMaxIdleConns(10)
	db.DB().SetMaxOpenConns(100)
	if err := initAppUsersTable(db); err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(new(model.User), new(model.Application), new(model.Message), new(model.Client), new(model.PluginConf)).Error; err != nil {
		return nil, err
	}

	userCount := 0
	db.Find(new(model.User)).Count(&userCount)
	if userCount == 0 {
		db.Create(&model.User{Name: defaultUser, Pass: auth.CreatePassword(defaultPass, defaultStrength), Admin: true})
	}

	return &GormDatabase{DB: db}, nil
}
