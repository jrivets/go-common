package gorivets

import (
	"strconv"
	"testing"
)

func TestSimple(t *testing.T) {
	l := NewLRU(1000, nil)
	l.Add("a", 23, 100)
	l.Add("b", 23, 100)
	if l.Len() != 2 {
		t.Fatal("expecting lru len == 2, but len=" + strconv.Itoa(l.Len()))
	}
	if l.Size() != 200 {
		t.Fatal("expecting lru size == 200, but it is " + strconv.FormatInt(l.Size(), 10))
	}

	l.Add("a", 23, 50)
	if l.Len() != 2 {
		t.Fatal("expecting lru len == 2, but len=" + strconv.Itoa(l.Len()))
	}
	if l.Size() != 150 {
		t.Fatal("expecting lru size == 150, but it is " + strconv.FormatInt(l.Size(), 10))
	}
}

func TestSize(t *testing.T) {
	i := 0
	arr := []string{"b", "bb", "a", "c"}
	l := NewLRU(1000, func(k, v interface{}) {
		ks := k.(string)
		if ks != arr[0] {
			t.Fatal("expecting key=" + arr[0] + ", k=" + ks)
		}
		arr = arr[1:]
		i++
	})
	l.Add("a", 23, 500)
	l.Add("b", 23, 250)
	l.Add("bb", 23, 250)
	l.Get("bb")
	l.Get("a")
	if l.Len() != 3 {
		t.Fatal("expecting lru len == 2, but len=" + strconv.Itoa(l.Len()))
	}
	if l.Size() != 1000 {
		t.Fatal("expecting lru size == 1000, but it is " + strconv.FormatInt(l.Size(), 10))
	}
	if i != 0 {
		t.Fatal("expecting i=0, but i=" + strconv.Itoa(i))
	}

	l.Add("c", 54, 500)
	if l.Len() != 2 {
		t.Fatal("expecting lru len == 2, but len=" + strconv.Itoa(l.Len()))
	}
	if i != 2 {
		t.Fatal("expecting i=2, but i=" + strconv.Itoa(i))
	}

	l.Add("d", 54, 1000)
	if l.Len() != 1 {
		t.Fatal("expecting lru len == 1, but len=" + strconv.Itoa(l.Len()))
	}
	if i != 4 {
		t.Fatal("expecting i=4, but i=" + strconv.Itoa(i))
	}
}

func TestDelete(t *testing.T) {
	arr := []string{"bb", "aa", "a", "b", "c", "d", "bbb"}
	l := NewLRU(1000, func(k, v interface{}) {
		ks := k.(string)
		if ks != arr[0] {
			t.Fatal("expecting key=" + arr[0] + ", k=" + ks)
		}
		arr = arr[1:]
	})
	l.Add("a", 23, 250)
	l.Add("aa", 23, 250)
	l.Add("b", 23, 250)
	l.Add("bb", 23, 250)
	l.Delete("bb")
	l.Delete("aa")
	l.Add("c", 23, 250)
	l.Add("d", 23, 250)
	l.Add("bbb", 23, 10300)
}
