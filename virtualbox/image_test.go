package virtualbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/go-test/deep"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnpackImage_gzip(t *testing.T) {
	Convey("Unpack image to a temporary directory", t, func() {
		dir, err := os.MkdirTemp("", "tfvbox-test-")
		So(err, ShouldBeNil)
		err = unpackImage(context.Background(), "testdata/hello.tar.gz", dir)
		So(err, ShouldBeNil)

		Convey("The unpacked file should be there", func() {
			bytes, err := os.ReadFile(filepath.Join(dir, "hello"))
			So(err, ShouldBeNil)

			Convey("And the uncompressed content should match the original file", func() {
				origin, err := os.ReadFile(filepath.Join("testdata", "hello"))
				So(err, ShouldBeNil)
				So(string(bytes), ShouldEqual, string(origin))
			})
		})

	})
}

func TestUnpackImage_bzip2(t *testing.T) {
	Convey("Unpack image to a temporary directory", t, func() {
		dir, err := os.MkdirTemp("", "tfvbox-test-")
		So(err, ShouldBeNil)
		err = unpackImage(context.Background(), "testdata/hello.tar.bz2", dir)
		So(err, ShouldBeNil)

		Convey("The unpacked file should be there", func() {
			bytes, err := os.ReadFile(filepath.Join(dir, "hello"))
			So(err, ShouldBeNil)

			Convey("And the uncompressed content should match the original file", func() {
				origin, err := os.ReadFile(filepath.Join("testdata", "hello"))
				So(err, ShouldBeNil)
				So(string(bytes), ShouldEqual, string(origin))
			})
		})

	})
}

func TestUnpackImage_xz(t *testing.T) {
	Convey("Unpack image to a temporary directory", t, func() {
		dir, err := os.MkdirTemp("", "tfvbox-test-")
		So(err, ShouldBeNil)
		err = unpackImage(context.Background(), "testdata/hello.tar.xz", dir)
		So(err, ShouldBeNil)

		Convey("The unpacked file should be there", func() {
			bytes, err := os.ReadFile(filepath.Join(dir, "hello"))
			So(err, ShouldBeNil)

			Convey("And the uncompressed content should match the original file", func() {
				origin, err := os.ReadFile(filepath.Join("testdata", "hello"))
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
			if err := tc.img.verify(context.Background()); !errors.Is(err, tc.err) {
				t.Errorf("verify() = %v, want %v", err, tc.err)
			}
		})
	}

}

func TestGatherDisks(t *testing.T) {
	disks, err := gatherDisks("./testdata/fakedisks")
	if err != nil {
		t.Fatalf("unable to gether disks: %v", err)
	}

	want := []string{
		filepath.Join("testdata", "fakedisks", "ubuntu-cloudimg.vmdk"),
		filepath.Join("testdata", "fakedisks", "ubuntu-cloudimg-configdrive.vmdk"),
	}

	if diff := deep.Equal(disks, want); diff != nil {
		t.Errorf("gatherDisks() diff = %v", diff)
	}
}

func TestByDiskPriority(t *testing.T) {
	testCases := map[string]struct {
		in   []string
		want []string
	}{
		"good order in": {
			[]string{"ubuntu-cloudimg.vmdk", "ubuntu-cloudimg-configdrive.vmdk"},
			[]string{"ubuntu-cloudimg.vmdk", "ubuntu-cloudimg-configdrive.vmdk"},
		},
		"bad order in": {
			[]string{"ubuntu-cloudimg-configdrive.vmdk", "ubuntu-cloudimg.vmdk"},
			[]string{"ubuntu-cloudimg.vmdk", "ubuntu-cloudimg-configdrive.vmdk"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := ByDiskPriority(tc.in)
			sort.Sort(got)
			if diff := deep.Equal([]string(got), tc.want); diff != nil {
				t.Errorf("ByDiskPriority() diff = %v", diff)
			}
		})
	}
}
