package gorivets

import (
	"fmt"
	"os"

	"gopkg.in/check.v1"
)

type gMapSuite struct {
}

var _ = check.Suite(&gMapSuite{})

func (s *gMapSuite) TestCreateFile(c *check.C) {
	c.Assert(gmap.file, check.IsNil)

	GMapGet("key")
	c.Assert(gmap.file, check.NotNil)
	fn := gmap.file.Name()
	fmt.Printf("fileName=%s", fn)
	_, err := os.Stat(fn)
	if os.IsNotExist(err) {
		c.Fatalf("Expecting the file %s exists, but it does not.", fn)
	}

	GMapShutdown()
	_, err = os.Stat(fn)
	if os.IsExist(err) {
		c.Fatalf("Expecting the file %s does not exist, but it does", fn)
	}
	c.Assert(gmap.file, check.IsNil)
}

func (s *gMapSuite) TestPanic(c *check.C) {
	defer GMapShutdown()
	c.Assert(gmap.file, check.IsNil)

	createGMapFile()
	c.Assert(gmap.file, check.NotNil)
	fn := gmap.file.Name()
	fmt.Printf("fileName=%s", fn)

	err := CheckPanic(func() {
		createGMapFile()
	})
	c.Assert(err, check.NotNil)
	fmt.Printf("error=%v", err)
}

func (s *gMapSuite) TestGetPut(c *check.C) {
	defer GMapShutdown()
	c.Assert(gmap.file, check.IsNil)

	v, ok := GMapGet("key")
	c.Assert(ok, check.Equals, false)
	c.Assert(GMapPut("key", 123), check.Equals, false)
	v, _ = GMapGet("key")
	c.Assert(v, check.Equals, 123)
	c.Assert(GMapPut("key", 345), check.Equals, true)
	v, _ = GMapGet("key")
	c.Assert(v, check.Equals, 345)
}

func (s *gMapSuite) TestDelete(c *check.C) {
	defer GMapShutdown()
	c.Assert(gmap.file, check.IsNil)

	c.Assert(GMapPut("key", 123), check.Equals, false)
	v, _ := GMapGet("key")
	c.Assert(v, check.Equals, 123)
	c.Assert(GMapDelete("key"), check.Equals, 123)
	c.Assert(GMapDelete("key"), check.IsNil)
}
