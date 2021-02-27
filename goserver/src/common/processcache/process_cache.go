package processcache

/*
#include "stdlib.h"
#include "process_cache.h"
*/
import "C"

import (
	"time"
	"unsafe"
)

type ProcessCache struct {
	cache unsafe.Pointer
}

func (this *ProcessCache) Set(key string, data []byte, expireUnixTime int64) {
	keydata := []byte(key)
	keylen := C.int(len(keydata))
	datalen := len(data)
	if datalen == 0 {
		C.CProcessCacheSet(this.cache, unsafe.Pointer(&keydata[0]), keylen, unsafe.Pointer(C.NULL), C.int(0), C.longlong(expireUnixTime))
	} else {
		C.CProcessCacheSet(this.cache, unsafe.Pointer(&keydata[0]), keylen, unsafe.Pointer(&data[0]), C.int(datalen), C.longlong(expireUnixTime))
	}
}

func (this *ProcessCache) Get(key string) (data []byte, found bool) {
	keydata := []byte(key)
	keylen := C.int(len(keydata))
	datalen := C.int(0)
	cdata := unsafe.Pointer(C.NULL)
	cfound := C.CProcessCacheGet(this.cache, unsafe.Pointer(&keydata[0]), keylen, &cdata, &datalen)
	if cfound == 0 {
		return nil, false
	} else {
		found = true
		data = C.GoBytes(cdata, datalen)
		C.free(cdata)
		return
	}
}

func NewProcessCache(bucketCount uint32, cleanDuration time.Duration) *ProcessCache {
	interval := cleanDuration / time.Duration(bucketCount)
	if interval < 10*time.Millisecond {
		panic("ProcessCache: cleanDuration too small")
	}

	cache := &ProcessCache{
		cache: C.CProcessCacheNew(C.int(bucketCount)),
	}

	go func(cache unsafe.Pointer, duration time.Duration) {
		for {
			time.Sleep(interval)
			msgbuf := make([]byte, 256)
			msglen := C.CProcessCacheClean(cache, unsafe.Pointer(&msgbuf[0]))
			msgbuf = msgbuf[:msglen]
			//tlog.Info(string(msgbuf))
		}
	}(cache.cache, interval)

	return cache
}
