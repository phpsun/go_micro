#ifndef PROCESS_CACHE__H__
#define PROCESS_CACHE__H__

#include <stdint.h>

void* CProcessCacheNew(int bucketCount);
void  CProcessCacheDestroy(void* cache);
void  CProcessCacheSet(void* cache, const void* key, int keyLen, const void* data, int dataLen, long long expireUnixTime);
int   CProcessCacheGet(void* cache, const void* key, int keyLen, void** data, int* dataLen);
int   CProcessCacheClean(void* cache, void* msgBuffer);

#endif
