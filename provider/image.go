package provider

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

func unpackImage(image, toDir string) error {
	/* Check if toDir exists */
	_, err := os.Stat(toDir)
	finfo, _ := ioutil.ReadDir(toDir)
	dirEmpty := len(finfo) == 0
	if os.IsNotExist(err) || dirEmpty {
		os.MkdirAll(toDir, os.ModeDir)
		fp, err := os.Open(image)
		if err != nil {
			return err
		}
		defer fp.Close()

		/* Unpack */
		// log.Printf("[DEBUG] Unpacking Gold virtual machine into %s\n", toDir)
		cmd := exec.Command("tar", "-xv", "-C", toDir, "-f", image)
		err = cmd.Run()
		if err != nil {
			log.Printf("[ERROR] Unpacking Gold image %s\n", fp.Name())
			log.Println(err.Error())
			return err
		}
	}
	return nil
}
