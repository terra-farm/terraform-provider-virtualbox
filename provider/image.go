package provider

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"path/filepath"
	"hash"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"io"
)
type Image struct {
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
		err = cmd.Run()
		if err != nil {
			log.Printf("[ERROR] Unpacking Gold image %s\n", fp.Name())
			log.Println(err.Error())
			return err
		}
	}
	return nil
}

func gatherDisks(path string) ([]string, error) {
	VDIs, err := filepath.Glob(filepath.Join(path, "**.vdi"))
	if err != nil {
		log.Printf("[ERROR] Get *.vdi in path '%s': %s", path, err.Error())
		return nil, err
	}
	VMDKs, err := filepath.Glob(filepath.Join(path, "**.vmdk"))
	if err != nil {
		log.Printf("[ERROR] Get *.vmdk in path '%s': %s", path, err.Error())
		return nil, err
	}
	disks := append(VDIs, VMDKs...)
	if len(disks) == 0 {
		err = fmt.Errorf("No VM disk files (*.vdi, *.vmdk) found in path '%s'", path)
		log.Printf("[ERROR] %s", err.Error())
		return nil, err
	}
	return disks, nil
}
func (img *Image) verify() error {
	// Makes sure the file cursor is positioned at the beginning of the file
	_, err := img.file.Seek(0, 0)
	if err != nil {
		return err
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
		return fmt.Errorf("[ERROR] Crypto algorithm no supported: %s", img.ChecksumType)
	}
	_, err = io.Copy(hasher, img.file)
	if err != nil {
		return err
	}

	result := fmt.Sprintf("%x", hasher.Sum(nil))

	if result != img.Checksum {
		return fmt.Errorf("[ERROR] Checksum does not match\n Result: %s\n Expected: %s", result, img.Checksum)
	}

	return nil
}