package provider

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"path/filepath"
)

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
