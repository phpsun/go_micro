#ifndef PROCESS_CACHE__H__
#define PROCESS_CACHE__H__

#include <stdint.h>

void* CListCacheNew(int bucketCount);
void  CListCacheDestroy(void* cache);
void  CListCachePush(void* cache, const void* key, int keyLen, const void* data, int dataLen, long long expireUnixTime);
void  CListCacheRem(void* cache, const void* key, int keyLen, const void* data, int dataLen);
void  CListCacheTrim(void* cache, const void* key, int keyLen, int count);
int   CListCacheGet(void* cache, const void* key, int keyLen, void** data, int* dataLen);
int   CListCacheClean(void* cache, void* msgBuffer);

#endif
