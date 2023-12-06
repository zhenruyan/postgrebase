package rediscache

import (
	"strings"

	"github.com/spf13/cast"
)

// CacheKey 缓存 key 拼接
func CacheKey(key string, params ...any) string {
	sb := strings.Builder{}
	sb.WriteString(key)
	for _, param := range params {
		sb.WriteString("#")
		switch param.(type) {
		case []int64:
			sb.WriteString(strings.Join(cast.ToStringSlice(param), "#"))
		default:
			sb.WriteString(cast.ToString(param))
		}
	}
	return sb.String()
}
