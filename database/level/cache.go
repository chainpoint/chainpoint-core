package level

import (
	"encoding/json"
	"github.com/go-redis/redis"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

type KVStore struct {
	RedisClient *redis.Client
	LevelDb     dbm.DB
	Logger      log.Logger
}

func NewKVStore(db *dbm.DB, logger log.Logger) *KVStore {
	return &KVStore{
		LevelDb: *db,
		Logger:  logger,
	}
}

func (cache *KVStore) Get(key string) (string, error) {
	bArr, err := cache.LevelDb.Get([]byte(key))
	return string(bArr), err
}

func (cache *KVStore) GetArray(key string) ([]string, error) {
	bArr, err := cache.LevelDb.Get([]byte(key))
	if err != nil {
		return []string{}, err
	}
	if bArr == nil {
		return []string{}, nil
	}
	var arr []string
	err = json.Unmarshal(bArr, &arr)
	if err != nil {
		return []string{}, err
	}
	return arr, nil
}

func (cache *KVStore) Append(key string, value string) error {
	results, err := cache.GetArray(key)
	if err != nil {
		return err
	}
	results = append(results, value)
	bArr, _ := json.Marshal(results)
	err = cache.LevelDb.Set([]byte(key), bArr)
	if err != nil {
		return err
	}
	return nil
}

func (cache *KVStore) SetArray(key string, values []string) error {
	bArr, _ := json.Marshal(values)
	err := cache.LevelDb.Set([]byte(key), bArr)
	if err != nil {
		return err
	}
	return nil
}

func (cache *KVStore) Set(key string, value string) error {
	err := cache.LevelDb.Set([]byte(key), []byte(value))
	if err != nil {
		return err
	}
	return nil
}

func (cache *KVStore) Del(key string, value string) error {
	if value == "" {
		return cache.LevelDb.Delete([]byte(key))
	}
	results, err := cache.GetArray(key)
	if err != nil {
		return err
	}
	for i, v := range results {
		if v == value {
			results = append(results[:i], results[i+1:]...)
			break
		}
	}
	bArr, _ := json.Marshal(results)
	err = cache.LevelDb.Set([]byte(key), bArr)
	if err != nil {
		return err
	}
	return nil
}
