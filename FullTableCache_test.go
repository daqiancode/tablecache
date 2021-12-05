package tablecache_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/daqiancode/tablecache"
	"github.com/stretchr/testify/suite"
)

type FullTableCacheTest struct {
	suite.Suite
	full *tablecache.FullTableCache
}

func (s *FullTableCacheTest) SetupTest() {
	rg := tablecache.NewRedisGorm(GetRedis(), GetMysql(), 3*time.Second, "ID", "test",
		func() interface{} { return &User{} }, func() interface{} { return &([]User{}) })
	s.full = tablecache.NewFullTableCache(rg, "test")
}

func (s *FullTableCacheTest) TestAll() {

	users, err := s.full.All()
	s.Nil(err)
	all := *(users.(*[]User))
	fmt.Println(all)

	u, err := s.full.Get(13)
	s.Nil(err)
	fmt.Println(u)
	u1 := u.(*User)
	u1.Name = "tom"
	s.full.Update(u, "Name")
	s.full.Delete(12, 11)
	s.full.Delete(12)
}

func TestFullTableCacheTest(t *testing.T) {
	suite.Run(t, new(FullTableCacheTest))
}
