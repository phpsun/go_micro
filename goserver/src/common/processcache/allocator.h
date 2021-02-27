#ifndef ALLOCTOR__H__
#define ALLOCTOR__H__

#include <pthread.h>

struct MyAllocator {
	pthread_mutex_t lock;
	struct AllocFreeNode* freelist;
	struct AllocMemChunk* chunklist;
	char* ptr;
	char* end;
    unsigned int blockSize;
    unsigned int chunkSize;
	unsigned int chunkCount;
};

void  initAllocator(struct MyAllocator* lalloc, unsigned int blockSize);
void  destroyAllocator(struct MyAllocator* lalloc);
void* allocBlock(struct MyAllocator* lalloc);
void  freeBlock(struct MyAllocator* lalloc, void* ptr);

#endif
