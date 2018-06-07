package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/aws/aws-xray-daemon/daemon/cfg"
)

// version-gen is a simple program that generates the daemon version number and writes to VERSION file.
func main() {
	fmt.Printf("AWS X-Ray daemon version: %v\n", cfg.Version)

	// Write X-Ray daemon version to VERSION file.
	if err := ioutil.WriteFile(filepath.Join("VERSION"), []byte(cfg.Version), 0600); err != nil {
		log.Fatalf("Error writing to VERSION file. %v", err)
	}
}
