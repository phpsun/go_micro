#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <pthread.h>
#include <inttypes.h>
#include "tommyhashdyn.h"
#include "allocator.h"

uint32_t MyCRC32(const void* data, int datalen);
void     MyMd5(const void* data, int datalen, unsigned char digest[16]);
//void     MySha1(const void* data, int datalen, unsigned char digest[20]);

#define HASH_INIT_VAL 321
#define KEY_LEN       16
#define RECORD_SEPRATOR '\x1E'
typedef int BOOL;

typedef struct ListItemHashNode
{
	tommy_hashdyn_node node;
	unsigned char  key[KEY_LEN];
	unsigned char* val;
	int            valLen;
	time_t         expireTime;
} ListItemHashNode;

typedef struct StorageListCleanParam
{
	struct MyAllocator* lalloc;
	tommy_hashdyn* map;
	time_t now;
	int    deleteCount;
} StorageListCleanParam;

static int HashNodeCompare(const void* arg, const void* obj)
{
	return memcmp(((ListItemHashNode*)obj)->key, (const unsigned char*)arg, KEY_LEN);
}

static void HashNodeFree(void* obj)
{
	ListItemHashNode* node = (ListItemHashNode*)obj;
	//free(node->key);
	if(node->val) free(node->val);
	//free(node);
}

static void HashNodeClean(void* arg, void* obj)
{
	ListItemHashNode* node = (ListItemHashNode*)obj;
	StorageListCleanParam* param = (StorageListCleanParam*)arg;
	if (node->expireTime < param->now) {
		tommy_hashdyn_remove_existing_withoutshrink(param->map, &node->node);
		if(node->val) free(node->val);
		freeBlock(param->lalloc, node);
		param->deleteCount++;
	}
}

inline static void HashNodeInsert(struct MyAllocator* lalloc, tommy_hashdyn* map, const unsigned char* key, const unsigned char* val, int valLen, time_t expireTime, pthread_rwlock_t* lock)
{
	tommy_hash_t hash = tommy_hash_u32(HASH_INIT_VAL, key, KEY_LEN);
	pthread_rwlock_wrlock(lock);
	ListItemHashNode* node = (ListItemHashNode*)tommy_hashdyn_search(map, HashNodeCompare, key, hash);
	if (node) {
		if (expireTime > 0 && valLen > 0) {
			int totalLen = valLen + node->valLen;
			unsigned char* newVal = (unsigned char*)malloc(totalLen+1);
			memcpy(newVal, val, valLen);
			if (node->valLen > 0) {
				memcpy(newVal + valLen, node->val, node->valLen);
			}
			newVal[totalLen] = 0;

			if(node->val) free(node->val);
			node->val = newVal;
			node->valLen = totalLen;
		}
		node->expireTime = expireTime;
	} else if (expireTime > 0) {
		node = (ListItemHashNode*)allocBlock(lalloc);
		memcpy(node->key, key, KEY_LEN);
		if (valLen > 0) {
			node->val = (unsigned char*)malloc(valLen+1);
			memcpy(node->val, val, valLen);
			node->val[valLen] = 0;
			node->valLen = valLen;
		} else {
			node->val = NULL;
			node->valLen = 0;
		}
		node->expireTime = expireTime;
		tommy_hashdyn_insert(map, &node->node, node, hash);
	}
	pthread_rwlock_unlock(lock);
}

inline static void HashNodeRem(tommy_hashdyn* map, const unsigned char* key, const unsigned char* val, int valLen, pthread_rwlock_t* lock)
{
	tommy_hash_t hash = tommy_hash_u32(HASH_INIT_VAL, key, KEY_LEN);
	pthread_rwlock_wrlock(lock);
	ListItemHashNode* node = (ListItemHashNode*)tommy_hashdyn_search(map, HashNodeCompare, key, hash);
	if (node && node->valLen > 0) {
		char* str = (char*)malloc(valLen + 3);
		str[0] = RECORD_SEPRATOR;
		memcpy(str+1, val, valLen);
		valLen++;
		str[valLen] = RECORD_SEPRATOR;
		str[valLen + 1] = 0;
		char* dataVal = (char*)node->val;
		char* p;
		if (strncmp(dataVal, str+1, valLen) == 0) {
			p = dataVal;
		} else {
			p = strstr(dataVal, str);
			if (p) p++;
		}
		free(str);
		if (p) {
			str = p + valLen;
			memmove(p, str, node->valLen - (str-dataVal) + 1);
			node->valLen -= valLen;
		}
	}
	pthread_rwlock_unlock(lock);
}

inline static void HashNodeTrim(tommy_hashdyn* map, const unsigned char* key, int count, pthread_rwlock_t* lock)
{
	tommy_hash_t hash = tommy_hash_u32(HASH_INIT_VAL, key, KEY_LEN);
	pthread_rwlock_wrlock(lock);
	ListItemHashNode* node = (ListItemHashNode*)tommy_hashdyn_search(map, HashNodeCompare, key, hash);
	if (node && node->valLen > 0) {
		if (count == 0) {
			node->valLen = 0;
			node->val[0] = 0;
		} else {
			int cnt = 0;
			char* dataVal = (char*)node->val;
			int i = 0;
			for (; i<node->valLen; i++) {
				if (dataVal[i] == RECORD_SEPRATOR) {
					cnt++;
					if (cnt == count) {
						i++;
						node->valLen = i;
						node->val[i] = 0;
						break;
					}
				}
			}
		}
	}
	pthread_rwlock_unlock(lock);
}

inline static BOOL HashNodeFind(tommy_hashdyn* map, const unsigned char* key, pthread_rwlock_t* lock, unsigned char** result, int* valLen)
{
	BOOL found = 0;
	unsigned char* val = NULL;
	tommy_hash_t hash = tommy_hash_u32(HASH_INIT_VAL, key, KEY_LEN);
	pthread_rwlock_rdlock(lock);
	ListItemHashNode* node = (ListItemHashNode*)tommy_hashdyn_search(map, HashNodeCompare, key, hash);
	if (node) {
		if (node->expireTime >= time(NULL)) {
			*valLen = node->valLen;
			if (*valLen > 0) {
				val = (unsigned char*)malloc(*valLen);
				memcpy(val, node->val, *valLen);
			}
			found = 1;
		}
	}
	pthread_rwlock_unlock(lock);
	*result = val;
	return found;
}

typedef struct CListStorage
{
	tommy_hashdyn    map;
	pthread_rwlock_t lock;
} CListStorage;

inline static void initStorage(CListStorage* s)
{
	tommy_hashdyn_init(&s->map);
	pthread_rwlock_init(&s->lock, NULL);
}

inline static void destroyStorage(CListStorage* s)
{
	tommy_hashdyn_foreach(&s->map, HashNodeFree);
	tommy_hashdyn_done(&s->map);
	pthread_rwlock_destroy(&s->lock);
}

typedef struct CListCache {
	struct MyAllocator alloctor;
	CListStorage* buckets;
	uint32_t      bucketCount;
	uint32_t      currentCleanIndex;
} CListCache;

void* CListCacheNew(int bucketCount)
{
	CListCache* c = (CListCache*)malloc(sizeof(CListCache));
	initAllocator(&c->alloctor, sizeof(ListItemHashNode));
	c->bucketCount = (uint32_t)bucketCount;
	c->currentCleanIndex = 0;
	c->buckets = (CListStorage*)malloc(bucketCount*sizeof(CListStorage));
	int i;
	for (i = 0; i < bucketCount; i++) {
		initStorage(c->buckets + i);
	}
	return c;
}

void CListCacheDestroy(void* _c)
{
	CListCache* c = (CListCache*)_c;
	int i;
	for (i = 0; i < c->bucketCount; i++) {
		destroyStorage(c->buckets + i);
	}
	free(c->buckets);
	destroyAllocator(&c->alloctor);
	free(c);
}

void CListCachePush(void* _c, const void* key, int keyLen, const void* data, int dataLen, long long expireUnixTime)
{
	CListCache* c = (CListCache*)_c;
	unsigned char keyhash[KEY_LEN];
	MyMd5(key, keyLen, keyhash);
	uint32_t bucketId = MyCRC32(key, keyLen);
	bucketId %= c->bucketCount;
	CListStorage* stor = &c->buckets[bucketId];

	HashNodeInsert(&c->alloctor, &stor->map, keyhash, (const unsigned char*)data, dataLen, (time_t)expireUnixTime, &stor->lock);
}

void  CListCacheRem(void* _c, const void* key, int keyLen, const void* data, int dataLen)
{
	if (dataLen <= 0) {
		return;
	}

	CListCache* c = (CListCache*)_c;
	unsigned char keyhash[KEY_LEN];
	MyMd5(key, keyLen, keyhash);
	uint32_t bucketId = MyCRC32(key, keyLen);
	bucketId %= c->bucketCount;
	CListStorage* stor = &c->buckets[bucketId];

	HashNodeRem(&stor->map, keyhash, (const unsigned char*)data, dataLen, &stor->lock);
}

void  CListCacheTrim(void* _c, const void* key, int keyLen, int count)
{
	if (count < 0) {
		return;
	}
	CListCache* c = (CListCache*)_c;
	unsigned char keyhash[KEY_LEN];
	MyMd5(key, keyLen, keyhash);
	uint32_t bucketId = MyCRC32(key, keyLen);
	bucketId %= c->bucketCount;
	CListStorage* stor = &c->buckets[bucketId];

	HashNodeTrim(&stor->map, keyhash, count, &stor->lock);
}

int CListCacheGet(void* _c, const void* key, int keyLen, void** data, int* dataLen)
{
	CListCache* c = (CListCache*)_c;
	unsigned char keyhash[KEY_LEN];
	MyMd5(key, keyLen, keyhash);
	uint32_t bucketId = MyCRC32(key, keyLen);
	bucketId %= c->bucketCount;
	CListStorage* stor = &c->buckets[bucketId];

	return  HashNodeFind(&stor->map, keyhash, &stor->lock, (unsigned char**)data, dataLen);
}

int CListCacheClean(void* _c, void* msgBuffer)
{
	CListCache* c = (CListCache*)_c;
	uint32_t index = c->currentCleanIndex;
	c->currentCleanIndex = (index + 1) % c->bucketCount;
	CListStorage* stor = &c->buckets[index];

	StorageListCleanParam param;
	param.lalloc = &c->alloctor;
	param.map = &stor->map;
	param.now = time(NULL);
	param.deleteCount = 0;

	uint64_t allocSize;
	uint64_t memorySize;
	uint64_t bucketSize;
	uint64_t mapSize;

	pthread_rwlock_wrlock(&stor->lock);
	tommy_hashdyn_foreach_arg(&stor->map, HashNodeClean, &param);
	//if (param.deleteCount) {
	//	tommy_hashdyn_shrink(&stor->map);
	//}
	bucketSize = stor->map.bucket_max;
	mapSize = tommy_hashdyn_count(&stor->map);
	pthread_rwlock_unlock(&stor->lock);

	memorySize = bucketSize * (uint64_t)sizeof(void*)
		+ mapSize * (uint64_t)sizeof(ListItemHashNode);
	allocSize = bucketSize * (uint64_t)sizeof(void*) * c->bucketCount
		+ (uint64_t)c->alloctor.chunkCount * (uint64_t)c->alloctor.chunkSize;

	memorySize /= 1024;
	allocSize /= 1024;

	if (msgBuffer) {
		char* msg = (char*)msgBuffer;
		sprintf(msg, "processcache: alloc:%" PRId64 "K, storage:%d, count:%" PRId64 ", bucket:%" PRId64 ", memory:%" PRId64 "K, delete:%d", 
			allocSize, (int)index, mapSize, bucketSize, memorySize, param.deleteCount);
		return strlen(msg);
	}
	return 0;
}
