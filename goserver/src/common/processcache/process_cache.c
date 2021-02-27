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
typedef int BOOL;

typedef struct StorageItemHashNode
{
	tommy_hashdyn_node node;
	unsigned char  key[KEY_LEN];
	unsigned char* val;
	int            valLen;
	time_t         expireTime;
} StorageItemHashNode;

typedef struct StorageCleanParam
{
	struct MyAllocator* lalloc;
	tommy_hashdyn* map;
	time_t now;
	int    deleteCount;
} StorageCleanParam;

static int HashNodeCompare(const void* arg, const void* obj)
{
	return memcmp(((StorageItemHashNode*)obj)->key, (const unsigned char*)arg, KEY_LEN);
}

static void HashNodeFree(void* obj)
{
	StorageItemHashNode* node = (StorageItemHashNode*)obj;
	//free(node->key);
	if(node->val) free(node->val);
	//free(node);
}

static void HashNodeClean(void* arg, void* obj)
{
	StorageItemHashNode* node = (StorageItemHashNode*)obj;
	StorageCleanParam* param = (StorageCleanParam*)arg;
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
	StorageItemHashNode* node = (StorageItemHashNode*)tommy_hashdyn_search(map, HashNodeCompare, key, hash);
	if (node) {
		if (node->valLen >= valLen) {
			if (valLen > 0) {
				memcpy(node->val, val, valLen);
			}
			node->valLen = valLen;
		} else {
			if(node->val) free(node->val);

			if (valLen > 0) {
				node->val = (unsigned char*)malloc(valLen);
				memcpy(node->val, val, valLen);
				node->valLen = valLen;
			} else {
				node->val = NULL;
				node->valLen = 0;
			}
		}
		node->expireTime = expireTime;
	} else if (expireTime > 0) {
		node = (StorageItemHashNode*)allocBlock(lalloc);
		memcpy(node->key, key, KEY_LEN);
		if (valLen > 0) {
			node->val = (unsigned char*)malloc(valLen);
			memcpy(node->val, val, valLen);
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

inline static BOOL HashNodeFind(tommy_hashdyn* map, const unsigned char* key, pthread_rwlock_t* lock, unsigned char** result, int* valLen)
{
	BOOL found = 0;
	unsigned char* val = NULL;
	tommy_hash_t hash = tommy_hash_u32(HASH_INIT_VAL, key, KEY_LEN);
	pthread_rwlock_rdlock(lock);
	StorageItemHashNode* node = (StorageItemHashNode*)tommy_hashdyn_search(map, HashNodeCompare, key, hash);
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

typedef struct CProcessStorage
{
	tommy_hashdyn    map;
	pthread_rwlock_t lock;
} CProcessStorage;

inline static void initStorage(CProcessStorage* s)
{
	tommy_hashdyn_init(&s->map);
	pthread_rwlock_init(&s->lock, NULL);
}

inline static void destroyStorage(CProcessStorage* s)
{
	tommy_hashdyn_foreach(&s->map, HashNodeFree);
	tommy_hashdyn_done(&s->map);
	pthread_rwlock_destroy(&s->lock);
}

typedef struct CProcessCache {
	struct MyAllocator alloctor;
	CProcessStorage* buckets;
	uint32_t         bucketCount;
	uint32_t         currentCleanIndex;
} CProcessCache;

void* CProcessCacheNew(int bucketCount)
{
	CProcessCache* c = (CProcessCache*)malloc(sizeof(CProcessCache));
	initAllocator(&c->alloctor, sizeof(StorageItemHashNode));
	c->bucketCount = (uint32_t)bucketCount;
	c->currentCleanIndex = 0;
	c->buckets = (CProcessStorage*)malloc(bucketCount*sizeof(CProcessStorage));
	int i;
	for (i = 0; i < bucketCount; i++) {
		initStorage(c->buckets + i);
	}
	return c;
}

void CProcessCacheDestroy(void* _c)
{
	CProcessCache* c = (CProcessCache*)_c;
	int i;
	for (i = 0; i < c->bucketCount; i++) {
		destroyStorage(c->buckets + i);
	}
	free(c->buckets);
	destroyAllocator(&c->alloctor);
	free(c);
}

void CProcessCacheSet(void* _c, const void* key, int keyLen, const void* data, int dataLen, long long expireUnixTime)
{
	CProcessCache* c = (CProcessCache*)_c;
	unsigned char keyhash[KEY_LEN];
	MyMd5(key, keyLen, keyhash);
	uint32_t bucketId = MyCRC32(key, keyLen);
	bucketId %= c->bucketCount;
	CProcessStorage* stor = &c->buckets[bucketId];

	HashNodeInsert(&c->alloctor, &stor->map, keyhash, (const unsigned char*)data, dataLen, (time_t)expireUnixTime, &stor->lock);
}

int CProcessCacheGet(void* _c, const void* key, int keyLen, void** data, int* dataLen)
{
	CProcessCache* c = (CProcessCache*)_c;
	unsigned char keyhash[KEY_LEN];
	MyMd5(key, keyLen, keyhash);
	uint32_t bucketId = MyCRC32(key, keyLen);
	bucketId %= c->bucketCount;
	CProcessStorage* stor = &c->buckets[bucketId];

	return  HashNodeFind(&stor->map, keyhash, &stor->lock, (unsigned char**)data, dataLen);
}

int CProcessCacheClean(void* _c, void* msgBuffer)
{
	CProcessCache* c = (CProcessCache*)_c;
	uint32_t index = c->currentCleanIndex;
	c->currentCleanIndex = (index + 1) % c->bucketCount;
	CProcessStorage* stor = &c->buckets[index];

	StorageCleanParam param;
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
		+ mapSize * (uint64_t)sizeof(StorageItemHashNode);
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
