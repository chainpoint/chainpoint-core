package cache

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis"
	dbm "github.com/tendermint/tm-db"
)

type Cache struct {
	RedisClient   *redis.Client
	LevelDb       dbm.DB
}

// create redis caching layer that gradually migrates to leveldb
func NewCache (redis *redis.Client, db *dbm.DB) *Cache {
	return &Cache{
		RedisClient: redis,
		LevelDb:     *db,
	}
}

func (cache *Cache) Get (key string) ([]string, error) {
	if cache.RedisClient != nil {
		results := cache.RedisClient.WithContext(context.Background()).SMembers(key)
		if results.Err() != nil {
			return []string{}, results.Err()
		}
		return results.Val(), nil
	} else {
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
}

func (cache *Cache) GetOne (key string) (string, error) {
	bArr, err := cache.LevelDb.Get([]byte(key))
	return string(bArr), err

}

func (cache *Cache) Add(key string, value string) error {
	if cache.RedisClient != nil {
		cache.RedisClient.WithContext(context.Background()).SAdd(key, value)
	}
	results, err := cache.Get(key)
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

func (cache *Cache) SetArray(key string, values []string) error {
	bArr, _ := json.Marshal(values)
	err := cache.LevelDb.Set([]byte(key), bArr)
	if err != nil {
		return err
	}
	return nil
}

func (cache *Cache) Set(key string, value string) error {
	err := cache.LevelDb.Set([]byte(key), []byte(value))
	if err != nil {
		return err
	}
	return nil
}

func (cache *Cache) Del(key string, value string) error {
	if cache.RedisClient != nil {
		cache.RedisClient.WithContext(context.Background()).SRem(key, value)
	}
	if value == "" {
		return cache.LevelDb.Delete([]byte(key))
	}
	results, err := cache.Get(key)
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