package idgen

import "time"

func nowUnixMilli() uint64 {
	return uint64(time.Now().UnixMilli())
}
