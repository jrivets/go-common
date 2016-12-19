package gorivets

import (
	"gopkg.in/check.v1"
)

type sortedSliceSuite struct {
}

var _ = check.Suite(&sortedSliceSuite{})

type paramType struct {
	v int
}

func (p1 *paramType) Compare(val Comparable) int {
	p2v := val.(*paramType).v
	switch {
	case p1.v < p2v:
		return -1
	case p1.v > p2v:
		return 1
	default:
		return 0
	}
}

func (s *sortedSliceSuite) TestNewSortedSlice(c *check.C) {
	ss, err := NewSortedSlice(-1)
	c.Assert(ss, check.IsNil)
	c.Assert(err, check.NotNil)

	ss, err = NewSortedSlice(1)
	c.Assert(ss, check.NotNil)
	c.Assert(err, check.IsNil)
}

func (s *sortedSliceSuite) TestNewSortedSliceByParams(c *check.C) {
	ss, err := NewSortedSliceByParams()
	c.Assert(ss, check.IsNil)
	c.Assert(err, check.NotNil)

	ss, err = NewSortedSliceByParams(make([]interface{}, 1)...)
	c.Assert(ss, check.NotNil)
	c.Assert(err, check.IsNil)

	ss, err = NewSortedSliceByParams([]interface{}{&paramType{2}, &paramType{1}}...)
	c.Assert(ss.Len(), check.Equals, 2)
	c.Assert(ss.At(0).(*paramType).v, check.Equals, 1)
	c.Assert(ss.At(1).(*paramType).v, check.Equals, 2)
}

func (s *sortedSliceSuite) TestLen(c *check.C) {
	ss, _ := NewSortedSliceByParams(&paramType{2}, &paramType{1})
	c.Assert(ss.Len(), check.Equals, 2)

	ss, _ = NewSortedSlice(24)
	c.Assert(ss.Len(), check.Equals, 0)
}

func (s *sortedSliceSuite) TestAdd(c *check.C) {
	ss, _ := NewSortedSlice(1)
	ss.Add(nil)
	c.Assert(ss.Len(), check.Equals, 0)

	ss.Add(&paramType{3})
	c.Assert(ss.Len(), check.Equals, 1)

	ss.Add(&paramType{1})
	c.Assert(ss.Len(), check.Equals, 2)
	c.Assert(ss.At(0).(*paramType).v, check.Equals, 1)
	c.Assert(ss.At(1).(*paramType).v, check.Equals, 3)
}

func (s *sortedSliceSuite) TestReverseAdd(c *check.C) {
	ss, _ := NewSortedSliceByComp(func(v1, v2 interface{}) int {
		p1 := v1.(*paramType)
		p2 := v2.(*paramType)
		return p2.Compare(p1)
	}, 1)
	ss.Add(nil)
	c.Assert(ss.Len(), check.Equals, 0)

	ss.Add(&paramType{3})
	c.Assert(ss.Len(), check.Equals, 1)

	ss.Add(&paramType{1})
	c.Assert(ss.Len(), check.Equals, 2)
	c.Assert(ss.At(0).(*paramType).v, check.Equals, 3)
	c.Assert(ss.At(1).(*paramType).v, check.Equals, 1)
}

func (s *sortedSliceSuite) TestFind(c *check.C) {
	ss, _ := NewSortedSlice(1)
	ss.Add(&paramType{3})
	c.Check(ss.At(0).(*paramType).v, check.Equals, 3)

	idx, ok := ss.Find(&paramType{2})
	c.Assert(ok, check.Equals, false)
	c.Assert(idx < 0, check.Equals, true)

	idx, ok = ss.Find(&paramType{3})
	c.Check(ok, check.Equals, true)
	c.Assert(idx, check.Equals, 0)
}

func (s *sortedSliceSuite) TestDelete(c *check.C) {
	ss, _ := NewSortedSlice(1)
	ss.Add(&paramType{3})
	ss.Add(&paramType{4})

	c.Assert(ss.Delete(&paramType{2}), check.Equals, false)
	c.Assert(ss.Len(), check.Equals, 2)

	c.Assert(ss.Delete(&paramType{3}), check.Equals, true)
	c.Assert(ss.Len(), check.Equals, 1)
	c.Check(ss.At(0).(*paramType).v, check.Equals, 4)

	ss.Delete(&paramType{4})
	c.Assert(ss.Len(), check.Equals, 0)
}

func (s *sortedSliceSuite) TestDeleteAt(c *check.C) {
	ss, _ := NewSortedSlice(1)
	ss.Add(&paramType{3})
	ss.Add(&paramType{4})

	e := ss.DeleteAt(1).(*paramType)
	c.Assert(e.v, check.Equals, 4)
	c.Assert(ss.Len(), check.Equals, 1)
	c.Check(ss.At(0).(*paramType).v, check.Equals, 3)

	ss.DeleteAt(0)
	c.Assert(ss.Len(), check.Equals, 0)
}

func (s *sortedSliceSuite) TestCopy(c *check.C) {
	ss, _ := NewSortedSlice(1)
	ss.Add(&paramType{3})
	ss.Add(&paramType{4})

	slice := ss.Copy()
	c.Check(slice[0].(*paramType).v, check.Equals, 3)
	c.Check(slice[1].(*paramType).v, check.Equals, 4)
	slice[0] = &paramType{25}
	c.Check(slice[0].(*paramType).v, check.Equals, 25)
	c.Check(ss.At(0).(*paramType).v, check.Equals, 3)
}
