package utils

import (
	"github.com/patrickmn/go-cache"
	"time"
)

var Ca *cache.Cache

func init() {
	Ca = cache.New(1*time.Hour, 2*time.Hour)
}

func SetCacheToken(token string) {
	Ca.Set("XToken", token, cache.DefaultExpiration)
}

func GetCacheToken() string {
	foo, found := Ca.Get("XToken")
	if found {
		return foo.(string)
	}
	return ""
}
