// lrucache implements a simple memory-based LRU cache.
// @link        https://github.com/chanxuehong/lrucache for the canonical source repository
// @license     https://github.com/chanxuehong/lrucache/blob/master/LICENSE
// @authors     chanxuehong(chanxuehong@gmail.com)

// lrucache implements a simple memory-based LRU cache.
package lrucache

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrNotFound  = errors.New("item not found")
	ErrNotStored = errors.New("item not stored")
)

type payload struct {
	Key   interface{}
	Value interface{}
}

//                              front                           back
//                            +-------+       +-------+       +-------+
// lruList:list.List          |       |------>|       |------>|       |
//                            |payload|<------|payload|<------|payload|
//                            +-------+       +-------+       +-------+
//                                ^               ^               ^
// payloadMap:                    |               |               |
// map[interface{}]*list.Element  |               |               |
//      +------+------+           |               |               |
//      | key  |value-+-----------+               |               |
//      +------+------+                           |               |
//      | key  |value-+---------------------------+               |
//      +------+------+                                           |
//      | key  |value-+-------------------------------------------+
//      +------+------+
//
// Principle:
//   1. len(payloadMap) == lruList.Len();
//   2. for Element of lruList, we get payloadMap[Element.Value.(*payload).Key] == Element;
//   3. in the list lruList, the younger element is always
//      in front of the older elements;
//

// Cache is a thread-safe fixed size LRU cache.
type Cache struct {
	mutex      sync.Mutex
	size       int
	lruList    *list.List
	payloadMap map[interface{}]*list.Element
}

// New creates an LRU of the given size
func New(size int) (cache *Cache) {
	if size <= 0 {
		panic(fmt.Sprintf("size must be > 0 and now == %d", size))
	}

	cache = &Cache{
		size:       size,
		lruList:    list.New(),
		payloadMap: make(map[interface{}]*list.Element, size),
	}
	return
}

// Size returns the size of cache.
func (cache *Cache) Size() (n int) {
	cache.mutex.Lock()
	n = cache.size
	cache.mutex.Unlock()
	return
}

// SetSize set a new size for the cache. if size <=0, we do nothing.
func (cache *Cache) SetSize(size int) {
	if size <= 0 {
		return
	}

	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	n := cache.lruList.Len() - size
	if n <= 0 {
		cache.size = size
		return
	}

	for i, e := n, cache.lruList.Back(); i > 0; i, e = i-1, cache.lruList.Back() {
		cache.remove(e)
	}

	cache.size = size
	return
}

// Len returns the number of items in the cache.
func (cache *Cache) Len() (n int) {
	cache.mutex.Lock()
	n = cache.lruList.Len()
	cache.mutex.Unlock()
	return
}

// add key-value to cache.
// ensure that there is no item with the same key in cache
func (cache *Cache) add(key, value interface{}) (err error) {
	if cache.lruList.Len() >= cache.size {
		e := cache.lruList.Back() // e != nil
		payload := e.Value.(*payload)

		delete(cache.payloadMap, payload.Key)

		payload.Key = key
		payload.Value = value

		cache.payloadMap[key] = e
		cache.lruList.MoveToFront(e)
		return
	}

	// now create new
	cache.payloadMap[key] = cache.lruList.PushFront(&payload{
		Key:   key,
		Value: value,
	})
	return
}

// remove Element e from Cache.lruList.
// ensure that e != nil and e is an element of list lruList.
func (cache *Cache) remove(e *list.Element) {
	delete(cache.payloadMap, e.Value.(*payload).Key)
	cache.lruList.Remove(e)
}

// Add key-value to cache.
// if there already exists a item with the same key, it returns ErrNotStored.
func (cache *Cache) Add(key string, value interface{}) (err error) {
	cache.mutex.Lock()
	if _, hit := cache.payloadMap[key]; hit {
		err = ErrNotStored

		cache.mutex.Unlock()
		return
	} else {
		err = cache.add(key, value)

		cache.mutex.Unlock()
		return
	}
}

// Set key-value to cache, unconditional
func (cache *Cache) Set(key, value interface{}) (err error) {
	cache.mutex.Lock()
	if e, hit := cache.payloadMap[key]; hit {
		payload := e.Value.(*payload)

		// payload.Key = key
		payload.Value = value
		cache.lruList.MoveToFront(e)

		cache.mutex.Unlock()
		return
	} else {
		err = cache.add(key, value)

		cache.mutex.Unlock()
		return
	}
}

// Get looks up a key's value from the cache.
//  if there is no such element with the key it returns ErrNotFound.
func (cache *Cache) Get(key interface{}) (value interface{}, err error) {
	cache.mutex.Lock()
	if e, hit := cache.payloadMap[key]; hit {
		cache.lruList.MoveToFront(e)
		value = e.Value.(*payload).Value

		cache.mutex.Unlock()
		return

	} else {
		err = ErrNotFound

		cache.mutex.Unlock()
		return
	}
}

// Remove removes the provided key from the cache.
//  if there is no such element with the key it returns ErrNotFound,
//  normally you can ignore this error.
func (cache *Cache) Remove(key interface{}) (err error) {
	cache.mutex.Lock()
	if e, hit := cache.payloadMap[key]; hit {
		cache.remove(e)

		cache.mutex.Unlock()
		return
	} else {
		err = ErrNotFound

		cache.mutex.Unlock()
		return
	}
}
