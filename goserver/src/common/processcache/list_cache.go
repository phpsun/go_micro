package processcache

/*
#include "stdlib.h"
#include "list_cache.h"
*/
import "C"

import (
	"bytes"
	"common/tlog"
	"fmt"
	"strings"
	"time"
	"unsafe"
)

const LIST_RECORD_SEPERATOR = "\x1E"

type ListCache struct {
	cache unsafe.Pointer
}

func (this *ListCache) Push(key string, vals []string, expireUnixTime int64) {
	keydata := []byte(key)
	keylen := C.int(len(keydata))
	count := len(vals)
	if count == 0 {
		C.CListCachePush(this.cache, unsafe.Pointer(&keydata[0]), keylen, unsafe.Pointer(C.NULL), C.int(0), C.longlong(expireUnixTime))
	} else {
		var buf bytes.Buffer
		for i := 0; i < count; i++ {
			buf.WriteString(vals[i])
			buf.WriteString(LIST_RECORD_SEPERATOR)
		}
		data := buf.Bytes()
		C.CListCachePush(this.cache, unsafe.Pointer(&keydata[0]), keylen, unsafe.Pointer(&data[0]), C.int(len(data)), C.longlong(expireUnixTime))
	}
}

func (this *ListCache) Get(key string) (vals []string, found bool) {
	keydata := []byte(key)
	keylen := C.int(len(keydata))
	datalen := C.int(0)
	cdata := unsafe.Pointer(C.NULL)
	cfound := C.CListCacheGet(this.cache, unsafe.Pointer(&keydata[0]), keylen, &cdata, &datalen)
	if cfound == 0 {
		return nil, false
	} else {
		if datalen > 0 {
			data := C.GoStringN((*C.char)(cdata), datalen)
			C.free(cdata)
			vals = strings.Split(data, LIST_RECORD_SEPERATOR)
			return vals[:len(vals)-1], true
		} else {
			return nil, true
		}
	}
}

func (this *ListCache) Rem(key string, val string) {
	if len(val) == 0 {
		return
	}
	keydata := []byte(key)
	keylen := C.int(len(keydata))
	data := []byte(val)
	C.CListCacheRem(this.cache, unsafe.Pointer(&keydata[0]), keylen, unsafe.Pointer(&data[0]), C.int(len(data)))
}

func (this *ListCache) Trim(key string, remainCount int) {
	keydata := []byte(key)
	keylen := C.int(len(keydata))
	C.CListCacheTrim(this.cache, unsafe.Pointer(&keydata[0]), keylen, C.int(remainCount))
}

func NewListCache(bucketCount uint32, cleanDuration time.Duration) *ListCache {
	interval := cleanDuration / time.Duration(bucketCount)
	if interval < 10*time.Millisecond {
		panic("ListCache: cleanDuration too small")
	}

	cache := &ListCache{
		cache: C.CListCacheNew(C.int(bucketCount)),
	}

	go func(cache unsafe.Pointer, duration time.Duration) {
		for {
			time.Sleep(interval)
			msgbuf := make([]byte, 256)
			msglen := C.CListCacheClean(cache, unsafe.Pointer(&msgbuf[0]))
			msgbuf = msgbuf[:msglen]
			tlog.Info(string(msgbuf))
		}
	}(cache.cache, interval)

	return cache
}

func TestListCache() {
	cache := NewListCache(5, 60*time.Minute)
	now := time.Now().Unix()
	key := "123"
	cache.Push(key, []string{"ab", "ff", "cd"}, now+10)
	cache.Rem(key, "abcdx")
	cache.Rem(key, "ff")
	cache.Push(key, []string{"ee", "vv"}, now+10)
	cache.Rem(key, "ee")
	cache.Rem(key, "cd")
	cache.Trim(key, 1)
	fmt.Println(cache.Get(key))
}
