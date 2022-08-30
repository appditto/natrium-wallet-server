package database

import (
	"fmt"

	"github.com/appditto/natrium-wallet-server/models/dbmodels"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	Host     string
	Port     string
	Password string
	User     string
	DBName   string
	SSLMode  string
}

func NewConnection(config *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return db, err
	}
	return db, nil
}

func DropAndCreateTables(db *gorm.DB) error {
	err := db.Migrator().DropTable(&dbmodels.FcmToken{})
	if err != nil {
		return err
	}
	err = db.Migrator().CreateTable(&dbmodels.FcmToken{})
	return err
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&dbmodels.FcmToken{})
}
