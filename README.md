# tablecache
Table cache with redis and gorm


## Example
```go
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

type User struct {
	Base
	Name string `gorm:"type:varchar(100) not null;"`
}

var tables = []interface{}{
	&User{},
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

type User struct {
	Base
	Name string `gorm:"type:varchar(100) not null;"`
}

var tables = []interface{}{
	&User{},
}

func (s *TableCacheTest) SetupTest() {
	redisGorm := tablecache.NewRedisGorm(GetRedis(), GetMysql(), 3*time.Minute, "ID", "test",
		func() interface{} { return &User{} }, func() interface{} { return &([]User{}) })
	redisGorm.GetDB().AutoMigrate(tables)
	s.users = tablecache.NewTableCache(redisGorm, "User", [][]string{{"Name"}})
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

```