package gorivets

import (
	"errors"

	"gopkg.in/check.v1"
)

type utilsSuite struct {
}

var _ = check.Suite(&utilsSuite{})

func (s *utilsSuite) TestNoPanic(c *check.C) {
	defer EndQuietly()
	panic("")
}

func (s *utilsSuite) TestMin(c *check.C) {
	c.Assert(Min(-1, 0), check.Equals, -1)
	c.Assert(Min(1, 0), check.Equals, 0)
	c.Assert(Min(2, 2), check.Equals, 2)
	c.Assert(Min(10, 100), check.Equals, 10)
}

func (s *utilsSuite) TesCompareInt(c *check.C) {
	c.Assert(CompareInt(10, 30), check.Equals, -1)
	c.Assert(CompareInt(-10, 5), check.Equals, -1)
	c.Assert(CompareInt(10, 10), check.Equals, 0)
	c.Assert(CompareInt(-5, -5), check.Equals, 0)
	c.Assert(CompareInt(10, 5), check.Equals, 1)
	c.Assert(CompareInt(-10, -50), check.Equals, 1)
}

func (s *utilsSuite) TestParseInt64(c *check.C) {
	_, err := ParseInt64("123", 1, 220, 0)
	c.Assert(err, check.NotNil)
	_, err = ParseInt64("123", 1, 220, 221)
	c.Assert(err, check.NotNil)

	v, err := ParseInt64("123", 1, 220, 10)
	c.Assert(err, check.IsNil)
	c.Assert(v, check.Equals, int64(123))

	v, err = ParseInt64("", 1, 220, 10)
	c.Assert(err, check.IsNil)
	c.Assert(v, check.Equals, int64(10))

	v, err = ParseInt64("k", 1, 220, 10)
	c.Assert(err, check.NotNil)

	v, err = ParseInt64("1k", 1, 220, 10)
	c.Assert(err, check.NotNil)

	v, err = ParseInt64("1k", 1, 2200, 10)
	c.Assert(v, check.Equals, int64(1000))

	v, err = ParseInt64("1Mb", 1, 2200000, 10)
	c.Assert(v, check.Equals, int64(1000000))

	v, err = ParseInt64("1MiB", 1, 2200000, 10)
	c.Assert(v, check.Equals, int64(1024*1024))
	c.Assert(int(v), check.Equals, 1024*1024)
}

func (s *utilsSuite) TestParseInt(c *check.C) {
	_, err := ParseInt("123", 1, 220, 0)
	c.Assert(err, check.NotNil)
	_, err = ParseInt("123", 1, 220, 221)
	c.Assert(err, check.NotNil)

	v, err := ParseInt("123", 1, 220, 10)
	c.Assert(err, check.IsNil)
	c.Assert(v, check.Equals, 123)

	v, err = ParseInt("", 1, 220, 10)
	c.Assert(err, check.IsNil)
	c.Assert(v, check.Equals, 10)

	v, err = ParseInt("1k", 1, 2200, 10)
	c.Assert(v, check.Equals, 1000)
}

func (s *utilsSuite) TestParseBool(c *check.C) {
	_, err := ParseBool("123", true)
	c.Assert(err, check.NotNil)

	v, err := ParseBool("true", true)
	c.Assert(err, check.IsNil)
	c.Assert(v, check.Equals, true)

	v, err = ParseBool("false", true)
	c.Assert(err, check.IsNil)
	c.Assert(v, check.Equals, false)

	v, err = ParseBool("", false)
	c.Assert(err, check.IsNil)
	c.Assert(v, check.Equals, false)

	v, err = ParseBool("", true)
	c.Assert(err, check.IsNil)
	c.Assert(v, check.Equals, true)
}

func (s *utilsSuite) TestCheckPanic(c *check.C) {
	err := errors.New("ddd")
	c.Assert(CheckPanic(func() { AssertNoError(nil) }), check.IsNil)
	c.Assert(CheckPanic(func() { AssertNoError(err) }), check.Equals, "ddd")
}

func (s *utilsSuite) TestIsNil(c *check.C) {
	var ss *utilsSuite = nil
	c.Assert(IsNil(s), check.Equals, false)
	c.Assert(IsNil(nil), check.Equals, true)
	c.Assert(IsNil(ss), check.Equals, true)
}
