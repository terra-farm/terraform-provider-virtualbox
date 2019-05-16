package virtualbox

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

type image struct {
	// Image URL where to download from
	URL string
	// Checksum of the image, used to check integrity after downloading it
	Checksum string
	// Algorithm use to check the checksum
	ChecksumType string
	// Internal file reference
	file *os.File
}

func unpackImage(image, toDir string) error {
	/* Check if toDir exists */
	_, err := os.Stat(toDir)
	finfo, _ := ioutil.ReadDir(toDir)
	dirEmpty := len(finfo) == 0
	if os.IsNotExist(err) || dirEmpty {
		os.MkdirAll(toDir, 0740)
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
			return errors.Wrapf(err, "unpacking gold image %s", fp.Name())
		}
	}
	return nil
}

func gatherDisks(path string) ([]string, error) {
	VDIs, err := filepath.Glob(filepath.Join(path, "**.vdi"))
	if err != nil {
		return nil, errors.Wrapf(err, "get *.vdi in '%s", path)
	}
	VMDKs, err := filepath.Glob(filepath.Join(path, "**.vmdk"))
	if err != nil {
		return nil, errors.Wrapf(err, "get *.vmdk in path '%s'", path)
	}
	disks := append(VDIs, VMDKs...)
	if len(disks) == 0 {
		return nil, errors.Wrapf(err,
			"no VM disk files (*.vdi, *.vmdk) found in path '%s'", path)
	}
	return disks, nil
}
func (img *image) verify() error {
	// Makes sure the file cursor is positioned at the beginning of the file
	if _, err := img.file.Seek(0, 0); err != nil {
		return errors.Wrap(err, "can't seek image file")
	}

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
		return fmt.Errorf(" Crypto algorithm no supported: %s", img.ChecksumType)
	}

	if _, err := io.Copy(hasher, img.file); err != nil {
		return errors.Wrap(err, "cannot hash image file")
	}

	result := fmt.Sprintf("%x", hasher.Sum(nil))
	if result != img.Checksum {
		return fmt.Errorf("checksum does not match\n Result: %s\n Expected: %s", result, img.Checksum)
	}

	return nil
}
