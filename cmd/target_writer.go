package cmd

import (
	"os"

	"gopkg.in/yaml.v2"
)

// WriteTarget writes <target> to <targetPath>.
func (w *GardenctlTargetWriter) WriteTarget(targetPath string, target TargetInterface) (err error) {
	var content []byte
	if content, err = yaml.Marshal(target); err != nil {
		return err
	}

	var file *os.File
	if file, err = os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		return err
	}
	defer file.Close()

	file.Write(content)
	file.Sync()

	return
}
