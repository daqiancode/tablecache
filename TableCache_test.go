package tablecache_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/daqiancode/tablecache"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type TableCacheTest struct {
	suite.Suite
	users *tablecache.TableCache
}

func GetMysql() *gorm.DB {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: false,       // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,       // Disable color
		},
	)
	// refer https://github.com/go-sql-driver/mysql#dsn-data-source-name for details
	dsn := "root:123456@tcp(127.0.0.1:3306)/cache?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: newLogger})
	if err != nil {
		panic(err)
	}
	return db
}

func GetRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
	})
}

type Base struct {
	ID        uint64    `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"type:datetime not null;"`
	UpdatedAt time.Time `gorm:"type:datetime not null;"`
	// DeletedAt sql.NullTime `gorm:"index"`
}

func (s Base) GetID() uint64 {
	return s.ID
}

type User struct {
	Base
	Name string `gorm:"type:varchar(100) not null;"`
}

// var tables = []interface{}{
// 	&User{},
// }

func (s *TableCacheTest) SetupTest() {
	rg := tablecache.NewRedisGorm(GetRedis(), GetMysql(), 3*time.Minute, "ID", "test",
		func() interface{} { return &User{} }, func() interface{} { return &([]User{}) })
	s.users = tablecache.NewTableCache(rg, "User", [][]string{{"Name"}})
}

func (s *TableCacheTest) TestGet() {

	u, err := s.users.Get(14)
	s.Nil(err)
	fmt.Println(u.(*User))
	u, err = s.users.Get(12)
	s.Nil(err)
	fmt.Println(u, u == nil)

}

func (s *TableCacheTest) TestList() {
	u, err := s.users.List([]uint64{15, 123})
	s.Nil(err)
	fmt.Println(*u.(*[]User))
	u, err = s.users.List([]uint64{12, 120})
	s.Nil(err)
	fmt.Println(*u.(*[]User))
}

func (s *TableCacheTest) TestGetBy() {
	u, err := s.users.GetBy("ID", 14)
	s.Nil(err)
	fmt.Println(u)
	u, err = s.users.GetBy("ID", 100)
	s.Nil(err)
	fmt.Println(u)
}

func (s *TableCacheTest) TestListBy() {
	u, err := s.users.ListBy("ID", 16)
	s.Nil(err)
	fmt.Println(u)
	u, err = s.users.ListBy("ID", 12)
	s.Nil(err)
	fmt.Println(u)
	u, err = s.users.ListBy("ID", 12)
	s.Nil(err)
	fmt.Println(u)
}

func (s *TableCacheTest) TestCreate() {
	u := User{Name: "haha"}
	err := s.users.Create(&u)
	s.Nil(err)
	fmt.Println(u)
	nu, err := s.users.Get(u.ID)
	s.Nil(err)
	newUser := nu.(*User)
	s.Equal(newUser.ID, u.ID)
	fmt.Println(newUser)
}
func (s *TableCacheTest) TestCreateMany() {
	u := User{Name: "haha"}
	us := []User{u}
	err := s.users.CreateMany(&us)
	s.Nil(err)
	fmt.Println(us)
}

func (s *TableCacheTest) TestDelete() {
	err := s.users.Delete(25)
	s.Nil(err)
	err = s.users.Delete(25, 26)
	s.Nil(err)
	fmt.Println(s.users.Get(25))
}

func TestTableCacheTest(t *testing.T) {
	suite.Run(t, new(TableCacheTest))
}
