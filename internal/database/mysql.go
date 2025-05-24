package database

import (
	"database/sql"
	"gorm.io/gorm"
	"github.com/meta-node-blockchain/mining-service/internal/config"
	"github.com/meta-node-blockchain/mining-service/internal/model"
	"gorm.io/driver/mysql"
	log "github.com/sirupsen/logrus"
	"time"	
	"fmt"
)

var(
	MySQLDB *gorm.DB
)

func StartMySQL(config *config.AppConfig) {
	db,err := sql.Open("mysql",config.MYSQL_URL)
	if err != nil {
		log.Fatal("Connect DB mysql Error",err)
	}
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:db,
	}),&gorm.Config{})
	if err != nil {
		log.Fatal("Connect DB Gorm Error")
	}
	gormdb, err := gormDB.DB()
	if err != nil {
		log.Fatal("Connect DB Gorm Error", err)
	}
	migrations := []interface{}{
		&model.OtpVerification{},
	}
	gormDB.AutoMigrate(migrations...)
	// gormDB.Migrator().
	gormdb.SetMaxIdleConns(10)
	gormdb.SetMaxOpenConns(100)
	gormdb.SetConnMaxLifetime(time.Minute * 5)

	MySQLDB = gormDB

	fmt.Print("MYSQL DATABASE CONNECTED!\n")
}
func GetMySqlConn() *gorm.DB {
	return MySQLDB
}