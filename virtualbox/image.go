package virtualbox

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// InvalidChecksumTypeError is returned when the passed checksum algorithm
// type is not supported
type InvalidChecksumTypeError string

func (e InvalidChecksumTypeError) Error() string {
	return fmt.Sprintf("invalid checksum algorithm: %q", string(e))
}

//nolint:unused
type image struct {
	// Image URL where to download from
	URL string
	// Checksum of the image, used to check integrity after downloading it
	Checksum string
	// Algorithm use to check the checksum
	ChecksumType string
	// Internal file reference
	file io.ReadSeeker
}

func unpackImage(image, toDir string) error {
	/* Check if toDir exists */
	_, err := os.Stat(toDir)
	finfo, _ := os.ReadDir(toDir)
	dirEmpty := len(finfo) == 0
	if os.IsNotExist(err) || dirEmpty {
		err := os.MkdirAll(toDir, 0740)
		if err != nil {
			return fmt.Errorf("unable to create %s directory: %w", toDir, err)
		}
		fp, err := os.Open(image)
		if err != nil {
			return err
		}
		defer fp.Close()

		/* Unpack */
		// log.Printf("[DEBUG] Unpacking Gold virtual machine into %s\n", toDir)
		cmd := exec.Command("tar", "-xv", "-C", toDir, "-f", image)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error unpacking gold image %s: %w", fp.Name(), err)
		}
	}
	return nil
}

func gatherDisks(path string) ([]string, error) {
	VDIs, err := filepath.Glob(filepath.Join(path, "**.vdi"))
	if err != nil {
		return nil, fmt.Errorf("get *.vdi in %q: %w", path, err)
	}
	VMDKs, err := filepath.Glob(filepath.Join(path, "**.vmdk"))
	if err != nil {
		return nil, fmt.Errorf("get *.vmdk in path %q: %w", path, err)
	}
	disks := append(VDIs, VMDKs...)
	if len(disks) == 0 {
		return nil, fmt.Errorf(
			"no VM disk files (*.vdi, *.vmdk) found in path %q", path)
	}
	prioritized := ByDiskPriority(disks)
	sort.Sort(prioritized)
	return prioritized, nil
}

// ByDiskPriority adds a simple sort to make sure that configdisk is not first
// in the returned boot order.
type ByDiskPriority []string

func (ss ByDiskPriority) Len() int      { return len(ss) }
func (ss ByDiskPriority) Swap(i, j int) { ss[i], ss[j] = ss[j], ss[i] }
func (ss ByDiskPriority) Less(i, j int) bool {
	return !strings.Contains(ss[i], "configdrive")
}

func (img *image) verify() error {
	log.Printf("[DEBUG] Verifying image checksum...")
	var hasher hash.Hash

	switch img.ChecksumType {
	case "md5":
		hasher = md5.New()
	case "sha1":
		hasher = sha1.New()
	case "sha256":
		hasher = sha256.New()
	case "sha512":
		hasher = sha512.New()
	default:
		return InvalidChecksumTypeError(img.ChecksumType)
	}

	// Makes sure the file cursor is positioned at the beginning of the file
	if _, err := img.file.Seek(0, 0); err != nil {
		return fmt.Errorf("can't seek image file: %w", err)
	}

	if _, err := io.Copy(hasher, img.file); err != nil {
		return fmt.Errorf("cannot hash image file: %w", err)
	}

	result := fmt.Sprintf("%x", hasher.Sum(nil))
	if result != img.Checksum {
		return fmt.Errorf("checksum does not match\n Result: %s\n Expected: %s", result, img.Checksum)
	}

	return nil
}
