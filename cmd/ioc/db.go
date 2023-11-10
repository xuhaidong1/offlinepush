package ioc

import (
	"github.com/xuhaidong1/offlinepush/config"
	"github.com/xuhaidong1/offlinepush/internal/repository/dao"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"sync"
)

var (
	db         *gorm.DB
	dbInitOnce sync.Once
)

func InitDB() *gorm.DB {
	dsn := config.StartConfig.MySQL.DSN
	var err error
	dbInitOnce.Do(func() {
		db, err = gorm.Open(mysql.Open(dsn))
		if err != nil {
			panic(err)
		}
		initTables(db)
	})
	return db
}

func initTables(db *gorm.DB) {
	err := dao.InitTable(db)
	if err != nil {
		panic(err)
	}
}
