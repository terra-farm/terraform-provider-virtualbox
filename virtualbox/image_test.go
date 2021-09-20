package virtualbox

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

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

func TestVerify(t *testing.T) {
	testCases := map[string]struct {
		img image
		err error
	}{
		"invalid checksum type": {
			img: image{
				ChecksumType: "invalid",
			},
			err: InvalidChecksumTypeError("invalid"),
		},
		// TODO: Fill this with actual test cases which test verification of the
		//       checksums.
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if err := tc.img.verify(); !errors.Is(err, tc.err) {
				t.Errorf("verify() = %v, want %v", err, tc.err)
			}
		})
	}
}
