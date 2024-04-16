package config

import (
	"database/sql"
	"fmt"
	"github.com/didi/gendry/manager"
	"github.com/didi/gendry/scanner"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

var DB *sql.DB

const ( //don't add comma
	database = "advisor"
	user     = "root"
	password = "2001liweijia"
	host     = "127.0.0.1"
)

var IsConsistent = false

func InitDB() *sql.DB {
	var err error
	op := manager.New(database, user, password, host)
	DB, err = op.Set(
		manager.SetCharset("utf8"),
		manager.SetAllowCleartextPasswords(true),
		manager.SetInterpolateParams(true),
		manager.SetTimeout(1*time.Second),
		manager.SetReadTimeout(1*time.Second),
		//manager.SetParseTime(true), //开启后 数据库中的时间类型只能读取到go中的time.Time类型 不开启则只能读取到go中的[]byte和string类型
		//manager.SetLoc(url.QueryEscape("Asia/Shanghai")),
	).Port(3306).Open(true)
	if err != nil {
		fmt.Println(err.Error())
		panic("Initial mysql failed")
	}
	err = DB.Ping()
	if err != nil {
		panic("Failed to connect to database")
	}
	fmt.Println("Connection established")
	scanner.SetTagName("json")
	return DB
}
