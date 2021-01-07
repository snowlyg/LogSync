package utils

import (
	"github.com/patrickmn/go-cache"
	"time"
)

var ca *cache.Cache

func GetCache() *cache.Cache {
	if ca != nil {
		return ca
	}
	ca = cache.New(1*time.Hour, 2*time.Hour)
	return ca
}

func SetCacheToken(token string) {
	GetCache().Set("XToken", token, cache.DefaultExpiration)
}

func GetCacheToken() string {
	foo, found := GetCache().Get("XToken")
	if found {
		return foo.(string)
	}
	return ""
}
