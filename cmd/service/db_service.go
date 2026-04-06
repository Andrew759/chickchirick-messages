package service

import (
	"chickchirick-messages/cmd/config/dto"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DBDecorator struct {
	Gorm   *gorm.DB
	Native *sql.DB
}

func InitORM(config dto.DataBaseConfigInterface) *DBDecorator {
	dsn := dsn(config)

	ORM, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("db connect failed: %w", err))
	}

	nativeDB, err := ORM.DB()
	if err != nil {
		panic(fmt.Errorf("error receiving the native interface: %w", err))
	}

	dbd := DBDecorator{
		Gorm:   ORM,
		Native: nativeDB,
	}

	return &dbd
}

func dsn(config dto.DataBaseConfigInterface) string {
	dsn := []string{
		"host=" + config.Host(),
		"user=" + config.User(),
		"password=" + config.Password(),
		"dbname=" + config.Name(),
		"port=" + strconv.Itoa(config.Port()),
	}
	if config.Timezone() != "" {
		dsn = append(dsn, "TimeZone="+config.Timezone())
	}

	return strings.Join(dsn, " ")
}

func (dbd DBDecorator) CloseDB() {
	err := dbd.Native.Close()
	if err != nil {
		panic(fmt.Errorf("db close error: %w", err))
	}
}

func (dbd DBDecorator) GDB() *gorm.DB {
	return dbd.Gorm
}

func (dbd DBDecorator) NativeDB() *sql.DB {
	return dbd.Native
}
