package tablecache

import (
	"errors"
	"reflect"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type FullTableCache struct {
	*RedisGorm
	key string
}

func NewFullTableCache(redisGorm *RedisGorm, key string) *FullTableCache {
	return &FullTableCache{
		RedisGorm: redisGorm,
		key:       key,
	}
}

// encode sliceRef -> {id: json_encoded(record)}
func (s *FullTableCache) encode(sliceRef interface{}) (map[string]interface{}, error) {
	vs := reflect.Indirect(reflect.ValueOf(sliceRef))
	n := vs.Len()
	r := make(map[string]interface{}, n)
	for i := 0; i < n; i++ {
		item := vs.Index(i)
		key := item.FieldByName(s.idField).Interface()
		bs, err := s.marshaller.Marshal(item.Interface())
		if err != nil {
			return nil, err
		}
		r[s.cacheUtil.Stringify(key)] = string(bs)
	}
	return r, nil
}

func (s *FullTableCache) Exist() (bool, error) {
	c, err := s.redisClient.Exists(s.redisCtx, s.key).Result()
	return c > 0, err
}

func (s *FullTableCache) ensureLoaded() error {
	ok, err := s.Exist()
	if err != nil {
		return err
	}
	if !ok {
		return s.Load()
	}
	return nil
}

func (s *FullTableCache) Load() error {
	records := s.FactoryListRef()
	tx := s.db.Find(records)
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return tx.Error
	}
	kvs, err := s.encode(records)
	if err != nil {
		return err
	}
	s.redisClient.HSet(s.redisCtx, s.key, kvs)
	if s.ttl > 0 {
		err := s.redisClient.Expire(s.redisCtx, s.key, s.ttl).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

//All return all record from cache. type is: *[]Table
func (s *FullTableCache) All() (interface{}, error) {
	records := s.FactoryListRef()
	err := s.ensureLoaded()
	if err != nil {
		return records, err
	}
	kvs, err := s.redisClient.HGetAll(s.redisCtx, s.key).Result()
	if err != nil {
		return records, err
	}
	if len(kvs) == 0 {
		return records, nil
	}
	jsonStr := "["
	for _, v := range kvs {
		jsonStr += v + ","
	}
	jsonStr = jsonStr[0:len(jsonStr)-1] + "]"
	err = s.marshaller.Unmarshal(records, jsonStr)
	return records, err
}

//Get record by id, type is *Table
func (s *FullTableCache) Get(id interface{}) (interface{}, error) {
	err := s.ensureLoaded()
	if err != nil {
		return nil, err
	}
	record := s.FactorySingleRef()
	jsonStr, err := s.redisClient.HGet(s.redisCtx, s.key, s.cacheUtil.Stringify(id)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return record, err
	}
	err = s.marshaller.Unmarshal(record, jsonStr)
	return record, err
}

func (s *FullTableCache) Create(valueRef interface{}) error {
	tx := s.db.Create(valueRef)
	if tx.Error != nil {
		return tx.Error
	}
	return s.set(valueRef)
}

func (s *FullTableCache) Save(valueRef interface{}) error {
	tx := s.db.Save(valueRef)
	if tx.Error != nil {
		return tx.Error
	}
	return s.set(valueRef)
}

func (s *FullTableCache) Update(valueRef interface{}, fields ...string) error {
	var tx *gorm.DB
	if len(fields) > 0 {
		tx = s.db.Model(valueRef).Select(fields).Updates(valueRef)
	} else {
		tx = s.db.Model(valueRef).Select("*").Updates(valueRef)
	}
	if tx.Error != nil {
		return tx.Error
	}
	return s.set(valueRef)
}
func (s *FullTableCache) set(valueRef interface{}) error {
	err := s.ensureLoaded()
	if err != nil {
		return err
	}
	id := s.cacheUtil.GetFieldValue(valueRef, s.idField)
	jsonStr, err := s.marshaller.Marshal(valueRef)
	if err != nil {
		return err
	}
	s.redisClient.HSet(s.redisCtx, s.key, s.cacheUtil.Stringify(id), jsonStr)
	return nil
}
func (s *FullTableCache) stringifyIDs(ids interface{}) []string {
	idsV := reflect.ValueOf(ids)
	n := idsV.Len()
	r := make([]string, n)
	for i := 0; i < n; i++ {
		r[i] = s.cacheUtil.Stringify(idsV.Index(i).Interface())
	}
	return r
}

//Delete by ids
func (s *FullTableCache) Delete(ids ...interface{}) error {
	if len(ids) == 0 {
		return nil
	}
	err := s.db.Delete(s.FactorySingleRef(), ids).Error
	if err != nil {
		return err
	}
	s.redisClient.HDel(s.redisCtx, s.key, s.stringifyIDs(ids)...)
	return nil
}
