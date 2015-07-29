package provider

import (
	. "github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"path/filepath"
	"testing"
)

var HelloText = "Hello, world!"

func TestUnpackImage_gzip(t *testing.T) {
	Convey("Unpack image to a temporary directory", t, func() {
		dir, err := ioutil.TempDir("", "tfvbox-test-")
		So(err, ShouldBeNil)
		err = unpackImage("testdata/hello.tar.gz", dir)
		So(err, ShouldBeNil)

		Convey("The unpacked file should be there", func() {
			bytes, err := ioutil.ReadFile(filepath.Join(dir, "hello"))
			So(err, ShouldBeNil)

			Convey("And the uncompressed content should match the original file", func() {
				origin, err := ioutil.ReadFile(filepath.Join("testdata", "hello"))
				So(err, ShouldBeNil)
				So(string(bytes), ShouldEqual, string(origin))
			})
		})

	})
}

func TestUnpackImage_bzip2(t *testing.T) {
	Convey("Unpack image to a temporary directory", t, func() {
		dir, err := ioutil.TempDir("", "tfvbox-test-")
		So(err, ShouldBeNil)
		err = unpackImage("testdata/hello.tar.bz2", dir)
		So(err, ShouldBeNil)

		Convey("The unpacked file should be there", func() {
			bytes, err := ioutil.ReadFile(filepath.Join(dir, "hello"))
			So(err, ShouldBeNil)

			Convey("And the uncompressed content should match the original file", func() {
				origin, err := ioutil.ReadFile(filepath.Join("testdata", "hello"))
				So(err, ShouldBeNil)
				So(string(bytes), ShouldEqual, string(origin))
			})
		})

	})
}

func TestUnpackImage_xz(t *testing.T) {
	Convey("Unpack image to a temporary directory", t, func() {
		dir, err := ioutil.TempDir("", "tfvbox-test-")
		So(err, ShouldBeNil)
		err = unpackImage("testdata/hello.tar.xz", dir)
		So(err, ShouldBeNil)

		Convey("The unpacked file should be there", func() {
			bytes, err := ioutil.ReadFile(filepath.Join(dir, "hello"))
			So(err, ShouldBeNil)

			Convey("And the uncompressed content should match the original file", func() {
				origin, err := ioutil.ReadFile(filepath.Join("testdata", "hello"))
				So(err, ShouldBeNil)
				So(string(bytes), ShouldEqual, string(origin))
			})
		})

	})
}
