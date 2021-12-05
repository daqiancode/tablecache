package tablecache

import (
	"context"
	"reflect"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type RedisGorm struct {
	redisClient      *redis.Client
	db               *gorm.DB
	ttl              time.Duration
	marshaller       Marshaller
	cachePrefix      string
	FactorySingleRef func() interface{}
	FactoryListRef   func() interface{}
	idField          string
	cacheUtil        *CacheUtil
	redisCtx         context.Context
	idType           reflect.Kind
}

func NewRedisGorm(redisClient *redis.Client, db *gorm.DB, ttl time.Duration, idField, cachePrefix string, factorySingleRef, factoryListRef func() interface{}) *RedisGorm {
	r := &RedisGorm{
		redisClient:      redisClient,
		db:               db,
		ttl:              ttl,
		marshaller:       &JSONMarshaller{},
		cachePrefix:      cachePrefix,
		idField:          idField,
		cacheUtil:        &CacheUtil{},
		FactorySingleRef: factorySingleRef,
		FactoryListRef:   factoryListRef,
		redisCtx:         context.Background(),
	}
	r.checkFields(idField)
	r.setIDType()
	return r
}

func (s *RedisGorm) GetRedis() *redis.Client {
	return s.redisClient
}

func (s *RedisGorm) GetDB() *gorm.DB {
	return s.db
}

func (s *RedisGorm) setIDType() {
	f := reflect.Indirect(reflect.ValueOf(s.FactorySingleRef()))
	s.idType = f.FieldByName(s.idField).Kind()
}

func (s *RedisGorm) IsIDInteger() bool {
	return s.idType == reflect.Int || s.idType == reflect.Int8 || s.idType == reflect.Int16 || s.idType == reflect.Int32 || s.idType == reflect.Int64 ||
		s.idType == reflect.Uint || s.idType == reflect.Uint8 || s.idType == reflect.Uint16 || s.idType == reflect.Uint32 || s.idType == reflect.Uint64

}
func (s *RedisGorm) SetMarshaller(marshaller Marshaller) {
	s.marshaller = marshaller
}

func (s *RedisGorm) SetRedisCtx(redisCtx context.Context) {
	s.redisCtx = redisCtx
}

func (s *RedisGorm) GetTTL() time.Duration {
	return s.ttl
}
func (s *RedisGorm) GetIDField(value interface{}) string {
	return s.idField
}
func (s *RedisGorm) GetID(value interface{}) interface{} {
	return s.cacheUtil.GetFieldValue(value, s.idField)
}

func (s *RedisGorm) checkFields(fields ...string) {
	t := reflect.TypeOf(s.FactorySingleRef())
	structFields := GetStructFields(t)
	m := make(map[string]bool, len(structFields))
	for _, v := range structFields {
		m[v] = true
	}
	structName := t.Elem().PkgPath() + "." + t.Elem().Name()
	for _, field := range fields {
		if !m[field] {
			panic("field " + field + " no in struct " + structName)
		}
	}
}
