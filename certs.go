package main

import (
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"
)

func DumpCerts(certs map[string]string) {

	logrus.Infof("Dumping certs to %s", certDir)
	for name, content := range certs {
		_, err := writeFile(certDir, name, content)
		if err != nil {
			logrus.Errorf("Cannot write file to %s/%s: %v", certDir, name, err)
		}
	}

	os.Setenv("CONSUL_CACERT", certDir+"/ca.crt")
	os.Setenv("CONSUL_TLSCERT", certDir+"/client.crt")
	os.Setenv("CONSUL_TLSKEY", certDir+"/client.key")
}

func fileExists(name string) bool {

	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false
	}

	return err == nil
}

func writeFile(directory string, filename string, content string) (bool, error) {

	file := directory + "/" + filename
	exists := fileExists(file)

	if exists {
		return false, nil
	}

	err := os.MkdirAll(directory, 0755)
	if err != nil {
		return false, err
	}

	err = ioutil.WriteFile(file, []byte(content), 0644)
	if err != nil {
		return false, err
	}

	return true, nil
}
