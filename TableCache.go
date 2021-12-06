package tablecache

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type TableCache struct {
	*RedisGorm
	structName string
	Indexes    [][]string
}

func NewTableCache(redisGorm *RedisGorm, structName string, indexes [][]string) *TableCache {

	r := &TableCache{
		RedisGorm:  redisGorm,
		structName: structName,
		Indexes:    indexes,
	}
	if len(indexes) > 0 {
		for _, v := range indexes {
			r.checkFields(v...)
		}
	}
	return r
}

func (s *TableCache) AddIndex(index []string) {
	s.checkFields(index...)
	s.Indexes = append(s.Indexes, index)
}

func (s *TableCache) GetMaxID() (uint64, error) {
	key := s.getMaxRedisKey()
	valueStr, err := s.redisClient.Get(s.redisCtx, key).Result()
	if err == nil {
		return strconv.ParseUint(valueStr, 10, 64)
	}
	if err != nil && err != redis.Nil {
		return 0, err
	}
	var r uint64
	m := s.FactorySingleRef()
	err = s.db.Model(m).Select("max(" + s.idField + ") as maxID").Scan(&r).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	err = s.redisClient.Set(s.redisCtx, key, r, s.ttl).Err()
	return r, err
}

func (s *TableCache) getMaxRedisKey() string {
	return s.cachePrefix + "/" + s.structName + "/__maxID__"
}
func (s *TableCache) getIDRedisKey(id interface{}) string {
	return s.cachePrefix + "/" + s.structName + "/" + s.cacheUtil.MakeKey(s.idField, id)
}

func (s *TableCache) getIndexRedisKey(index map[string]interface{}) string {
	return s.cachePrefix + "/" + s.structName + "/index/" + s.cacheUtil.MakeKeyWithMap(index)
}

func (s *TableCache) cacheGetByID(id interface{}) (interface{}, bool, error) {
	redisKey := s.getIDRedisKey(id)
	jsonStr, err := s.redisClient.Get(s.redisCtx, redisKey).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if jsonStr == NullStr {
		return nil, true, nil
	}
	r := s.FactorySingleRef()
	err = s.marshaller.Unmarshal(r, jsonStr)
	if err != nil {
		return nil, true, err
	}
	return r, true, nil
}

// func (s *TableCache) cacheSetByID(value interface{}, id interface{}) error {
// 	if id == 0 {
// 		id = s.GetID(value).(uint64)
// 	} else {
// 		redisKey := s.getIDRedisKey(id)
// 		if s.cacheUtil.GetFieldValue(value, s.idField) != id {
// 			s.redisClient.Set(s.redisCtx, redisKey, NullStr, s.ttl)
// 			return nil
// 		}
// 	}
// 	redisKey := s.getIDRedisKey(id)
// 	jsonStr, err := s.marshaller.Marshal(value)
// 	if err != nil {
// 		return err
// 	}
// 	return s.redisClient.Set(s.redisCtx, redisKey, jsonStr, s.ttl).Err()
// }

func (s *TableCache) cacheGet(valueRef interface{}, key string) (interface{}, bool, error) {
	jsonStr, err := s.redisClient.Get(s.redisCtx, key).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if jsonStr == NullStr {
		return nil, true, nil
	}
	r := s.FactorySingleRef()
	err = s.marshaller.Unmarshal(r, jsonStr)
	if err != nil {
		return nil, true, err
	}
	return r, true, nil
}

func (s *TableCache) cacheSet(value interface{}, key string) error {

	jsonStr, err := s.marshaller.Marshal(value)
	if err != nil {
		return err
	}
	return s.redisClient.Set(s.redisCtx, key, jsonStr, s.ttl).Err()
}

// return []string
func (s *TableCache) cacheMGet(keys []string) ([]interface{}, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	// var r map[string][]byte
	return s.redisClient.MGet(s.redisCtx, keys...).Result()

}

func (s *TableCache) cacheGetInt(key string) (uint64, bool, error) {
	r, err := s.redisClient.Get(s.redisCtx, key).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	v, err := strconv.ParseUint(r, 10, 64)
	return v, true, err
}
func (s *TableCache) cacheSetInt(key string, value uint64) error {
	return s.redisClient.Set(s.redisCtx, key, strconv.FormatUint(value, 10), s.ttl).Err()
}

func (s *TableCache) cacheGetInts(key string) ([]uint64, bool, error) {
	r, err := s.redisClient.Get(s.redisCtx, key).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var values []uint64
	err = s.marshaller.Unmarshal(&values, r)
	return values, true, err
}
func (s *TableCache) cacheSetInts(key string, values []uint64) error {
	vlauesStr, err := s.marshaller.Marshal(values)
	if err != nil {
		return err
	}
	return s.redisClient.Set(s.redisCtx, key, vlauesStr, s.ttl).Err()
}

func (s *TableCache) pick(obj interface{}, keys []string) map[string]interface{} {
	r := make(map[string]interface{}, len(keys))
	for _, k := range keys {
		r[k] = s.cacheUtil.GetFieldValue(obj, k)
	}
	return r
}

func (s *TableCache) Get(id interface{}) (interface{}, error) {
	if s.IsIDInteger() {
		idInt, err := strconv.ParseUint(fmt.Sprintf("%d", id), 10, 64)
		if err != nil {
			return nil, err
		}
		maxID, err := s.GetMaxID()
		if err != nil {
			return nil, err
		}
		if idInt > maxID || idInt <= 0 {
			return nil, nil
		}
	}
	r := s.FactorySingleRef()
	key := s.getIDRedisKey(id)
	r, ok, err := s.cacheGet(r, key)
	if err != nil {
		return nil, err
	}
	if ok {
		return r, err
	}
	r1 := s.FactorySingleRef()
	tx := s.db.Take(r1, id)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		err = s.cacheSet(nil, key)
		return nil, err
	}
	if tx.Error != nil {
		return nil, tx.Error
	}
	err = s.cacheSet(r1, key)
	return r1, err
}

func (s *TableCache) hasNilInSlices(values []interface{}) bool {
	for _, v := range values {
		if v == nil {
			return true
		}
	}
	return false
}
func (s *TableCache) List(ids interface{}) (interface{}, error) {
	if ids == nil {
		return s.FactoryListRef(), nil
	}
	idsV := reflect.Indirect(reflect.ValueOf(ids))
	n := idsV.Len()
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		keys[i] = s.getIDRedisKey(idsV.Index(i).Interface())
	}
	strs, err := s.cacheMGet(keys)
	if err != nil {
		return nil, err
	}

	// no nil value in redis,
	if !s.hasNilInSlices(strs) {
		r := s.FactoryListRef()
		jsonStr := "[ "
		for _, v := range strs {
			if v == nil || NullStr == v {
				// jsonStr += "null,"
				continue
			}
			jsonStr += v.(string) + ","
		}
		jsonStr = jsonStr[0:len(jsonStr)-1] + "]"
		err = s.marshaller.Unmarshal(r, jsonStr)
		return r, err
	}
	// fetch from db and store into redis
	r1 := s.FactoryListRef()
	tx := s.db.Where(ids).Find(r1)
	if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return r1, tx.Error
	}
	var key string

	// var id uint64
	var goodIds []interface{}
	rV := reflect.Indirect(reflect.ValueOf(r1))
	m := rV.Len()
	for i := 0; i < m; i++ {
		ele := rV.Index(i)
		id := ele.FieldByName(s.idField).Interface()
		err = s.cacheSet(ele.Interface(), s.getIDRedisKey(id))
		if err != nil {
			return r1, err
		}
		goodIds = append(goodIds, id)
	}
	badIds := SubStrs(ToStringSlice(ids), ToStringSlice(goodIds))
	for _, v := range badIds {
		key = s.getIDRedisKey(v)
		s.cacheSet(nil, key)
		// err := s.redisClient.Set(s.redisCtx, key, NullStr, s.ttl).Err()
		if err != nil {
			return r1, err
		}
	}
	return r1, nil
}

//GetBy index ,index:eg. uid,1
func (s *TableCache) GetBy(index ...interface{}) (interface{}, error) {
	return s.GetByMap(argsToMap(index...))
}

// redis key eg. projectusers/pid/2/uid/1 -> id
func (s *TableCache) GetByMap(index map[string]interface{}) (interface{}, error) {
	key := s.getIndexRedisKey(index)
	id, ok, err := s.cacheGetInt(key)
	if err != nil {
		return nil, err
	}
	if ok { //hit
		if id == 0 {
			return nil, nil
		}
		return s.Get(id)
	}

	// miss
	r1 := s.FactorySingleRef()
	err = s.db.Where(index).Take(r1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = s.cacheSetInt(key, 0)
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	err = s.cacheSetInt(key, s.GetID(r1).(uint64))
	return r1, err
}

//ListBy index ,index:eg. uid,1
func (s *TableCache) ListBy(index ...interface{}) (interface{}, error) {
	return s.ListByMap(argsToMap(index...))
}

// redis key eg. projectusers/pid/2/uid/1 -> [id1,id2]
func (s *TableCache) ListByMap(index map[string]interface{}) (interface{}, error) {
	key := s.getIndexRedisKey(index)
	var ids []uint64
	ids, ok, err := s.cacheGetInts(key)
	fmt.Println(key, ids, ok, err)
	if err != nil {
		return s.FactorySingleRef, err
	}
	if ok { //hit
		return s.List(ids)
	}

	// miss
	r1 := s.FactoryListRef()
	err = s.db.Where(index).Find(r1).Error
	if err != nil {
		return nil, err
	}

	r1V := reflect.Indirect(reflect.ValueOf(r1))
	n := r1V.Len()
	newIds := make([]uint64, n)
	for i := 0; i < n; i++ {
		newIds[i] = r1V.Index(i).FieldByName(s.idField).Uint()
	}
	err = s.cacheSetInts(key, newIds)
	return r1, err
}

func (s *TableCache) Create(valueRef interface{}) error {
	tx := s.db.Create(valueRef)
	if tx.Error != nil {
		return tx.Error
	}
	return s.ClearCache(valueRef)

}

func (s *TableCache) CreateMany(sliceRef interface{}) error {
	err := s.db.CreateInBatches(sliceRef, 200).Error
	if err != nil {
		return err
	}
	sr := reflect.Indirect(reflect.ValueOf(sliceRef))
	n := sr.Len()
	var objs []interface{}
	for i := 0; i < n; i++ {
		objs = append(objs, sr.Index(i).Interface())
	}
	return s.ClearCache(objs...)
}

func (s *TableCache) Save(valueRef interface{}) error {
	tx := s.db.Save(valueRef)
	if tx.Error != nil {
		return tx.Error
	}
	return s.ClearCache(valueRef)
}

//Delete by id , eg. Delete(1,2)
func (s *TableCache) Delete(ids ...interface{}) error {
	var args interface{} = ids
	if len(ids) == 1 && isSlice(ids[0]) {
		args = ids[0]
	}
	var old []map[string]interface{}
	m := s.FactorySingleRef()
	err := s.db.Model(m).Where(args).Find(&old).Error
	if err != nil {
		return err
	}

	err = s.db.Delete(m, args).Error
	if err != nil {
		return err
	}
	return s.ClearCacheWithMaps(old...)
}
func (s *TableCache) Update(resultRef interface{}, fields ...string) error {
	v1 := make(map[string]interface{})
	m := s.FactorySingleRef()
	err := s.db.Model(m).Take(&v1, s.GetID(resultRef)).Error
	if err != nil {
		return err
	}
	m1 := s.FactorySingleRef()
	var tx *gorm.DB
	if len(fields) > 0 {
		tx = s.db.Model(m1).Select(fields).Updates(resultRef)
	} else {
		tx = s.db.Model(m1).Select("*").Updates(resultRef)
	}
	if tx.Error != nil {
		return tx.Error
	}
	v2 := make(map[string]interface{})
	err = s.db.Model(m1).Take(&v2, s.GetID(resultRef)).Error
	if err != nil {
		return err
	}
	return s.ClearCacheWithMaps(v1, v2)
}

func (s *TableCache) ClearCache(objs ...interface{}) error {
	if len(objs) == 0 {
		return nil
	}
	var args interface{} = objs
	if len(objs) == 1 && isSlice(objs[0]) {
		args = objs[0]
	}
	objsV := reflect.Indirect(reflect.ValueOf(args))
	n := objsV.Len()
	m := n*(len(s.Indexes)+1) + 1
	var keySet map[string]bool = make(map[string]bool, m)
	keySet[s.getMaxRedisKey()] = true
	for i := 0; i < n; i++ {
		v := reflect.Indirect(objsV.Index(i)).Interface()
		id := s.GetID(v).(uint64)
		keySet[s.getIDRedisKey(id)] = true
		for _, pairs := range s.Indexes {
			d := s.pick(v, pairs)
			keySet[s.getIndexRedisKey(d)] = true
		}
	}
	rkeys := make([]string, m)
	i := 0
	for k := range keySet {
		rkeys[i] = k
		i++
	}
	return s.redisClient.Del(s.redisCtx, rkeys...).Err()
}

func (s *TableCache) ClearCacheWithMaps(objs ...map[string]interface{}) error {
	if len(objs) == 0 {
		return nil
	}
	n := len(objs)
	m := n*(len(s.Indexes)+1) + 1
	var keySet map[string]bool = make(map[string]bool, m)
	keySet[s.getMaxRedisKey()] = true
	for i := 0; i < n; i++ {
		v := objs[i]
		keySet[s.cacheUtil.MakeKeyWithMap(pickFromMap(v, s.idField))] = true
		for _, pairs := range s.Indexes {
			m := pickFromMap(v, pairs...)
			keySet[s.getIndexRedisKey(m)] = true
		}
	}

	rkeys := make([]string, m)
	i := 0
	for k := range keySet {
		rkeys[i] = k
		i++
	}
	return s.redisClient.Del(s.redisCtx, rkeys...).Err()
}

func (s *TableCache) DeleteUint64s(ids []uint64) error {
	args := make([]interface{}, len(ids))
	for i, v := range ids {
		args[i] = v
	}
	return s.Delete(ids)
}

func (s *TableCache) DeleteStrs(ids []string) error {
	args := make([]interface{}, len(ids))
	for i, v := range ids {
		args[i] = v
	}
	return s.Delete(ids)
}
