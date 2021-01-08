package utils

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"net/http"
	"time"
)

var ca *cache.Cache
var token string
var phpsess *http.Cookie

func SetSessionId(cookies []*http.Cookie) {
	for _, cookie := range cookies {
		if cookie.Name == "PHPSESSID" {
			GetCache().Set(fmt.Sprintf("PHPSESSIONID_%s", Config.Appid), cookie, cache.DefaultExpiration)
		}
	}
}

func GetSessionId() *http.Cookie {
	if phpsess != nil {
		return phpsess
	}
	foo, found := GetCache().Get(fmt.Sprintf("PHPSESSIONID_%s", Config.Appid))
	if found {
		phpsess = foo.(*http.Cookie)
	}
	return phpsess
}

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
		token = foo.(string)
	}
	return token
}
