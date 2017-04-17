package gorivets

import (
	"container/list"
	"strconv"
)

type (
	Lru struct {
		list     *list.List
		elements map[interface{}]*list.Element
		size     int64
		maxSize  int64
		callback Callback
	}

	element struct {
		key  interface{}
		val  interface{}
		size int64
	}

	Callback func(k, v interface{})
)

func NewLRU(maxSize int64, callback Callback) *Lru {
	if maxSize < 1 {
		panic("LRU size=" + strconv.FormatInt(maxSize, 10) + " should be positive.")
	}
	l := new(Lru)
	l.list = list.New()
	l.elements = make(map[interface{}]*list.Element)
	l.size = 0
	l.maxSize = maxSize
	l.callback = callback
	return l
}

func (lru *Lru) Add(k, v interface{}, size int64) {
	lru.Delete(k)
	e := &element{key: k, val: v, size: size}
	el := lru.list.PushBack(e)
	lru.elements[k] = el
	lru.size += size
	for lru.size > lru.maxSize && lru.deleteLast() {

	}
}

func (lru *Lru) Get(k interface{}) (interface{}, bool) {
	if e, ok := lru.elements[k]; ok {
		lru.list.MoveToBack(e)
		return e.Value.(*element).val, true
	}
	return nil, false
}

func (lru *Lru) Delete(k interface{}) interface{} {
	return lru.DeleteWithCallback(k, true)
}

func (lru *Lru) DeleteWithCallback(k interface{}, callback bool) interface{} {
	el, ok := lru.elements[k]
	if !ok {
		return nil
	}
	delete(lru.elements, k)
	e := lru.list.Remove(el).(*element)
	lru.size -= e.size
	if callback && lru.callback != nil {
		lru.callback(e.key, e.val)
	}
	return e.val
}

func (lru *Lru) Len() int {
	return len(lru.elements)
}

func (lru *Lru) Size() int64 {
	return lru.size
}

func (lru *Lru) deleteLast() bool {
	el := lru.list.Front()
	if el == nil {
		return false
	}
	e := el.Value.(*element)
	lru.Delete(e.key)
	return true
}
