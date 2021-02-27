#include <stdlib.h>
#include <string.h>
#include "allocator.h"

#define CHUNK_SIZE (4*1024*1024)

struct AllocFreeNode {
	struct AllocFreeNode* next;
};

struct AllocMemChunk {
	struct AllocMemChunk* next;
};

void initAllocator(struct MyAllocator* lalloc, unsigned int blockSize)
{
	pthread_mutex_init(&lalloc->lock, NULL);
	lalloc->freelist = NULL;
	lalloc->chunklist = NULL;
	lalloc->ptr = NULL;
	lalloc->end = NULL;
	lalloc->blockSize = blockSize;
	lalloc->chunkSize = (((CHUNK_SIZE - sizeof(struct AllocMemChunk))/blockSize) * blockSize) + sizeof(struct AllocMemChunk);
	lalloc->chunkCount = 0;
}

void destroyAllocator(struct MyAllocator* lalloc)
{
	struct AllocMemChunk* mc = lalloc->chunklist;
	while(mc) {
		struct AllocMemChunk* tmp = mc;
		mc = mc->next;
		free(tmp);
	}
	pthread_mutex_destroy(&lalloc->lock);
}

static void allocChunk(struct MyAllocator* lalloc)
{
	struct AllocMemChunk* mc = (struct AllocMemChunk*)malloc(lalloc->chunkSize);
	mc->next = lalloc->chunklist;
	lalloc->ptr = (char*)(mc+1);
	lalloc->end = ((char*)mc) + lalloc->chunkSize;
	lalloc->chunklist = mc;
	lalloc->chunkCount++;
}

void* allocBlock(struct MyAllocator* lalloc)
{
	void* ret;
	pthread_mutex_lock(&lalloc->lock);
	struct AllocFreeNode* fn = lalloc->freelist;
	if (fn) {
		lalloc->freelist = fn->next;
		ret = fn;
	} else {
		if ((lalloc->end - lalloc->ptr) < lalloc->blockSize) {
			allocChunk(lalloc);
		}
		ret = lalloc->ptr;
		lalloc->ptr += lalloc->blockSize;
	}
	pthread_mutex_unlock(&lalloc->lock);
	return ret;
}

void freeBlock(struct MyAllocator* lalloc, void* ptr)
{
	pthread_mutex_lock(&lalloc->lock);
	struct AllocFreeNode* fn = lalloc->freelist;
	struct AllocFreeNode* node = (struct AllocFreeNode *)ptr;
	node->next = fn;
	lalloc->freelist = node;
	pthread_mutex_unlock(&lalloc->lock);
}
