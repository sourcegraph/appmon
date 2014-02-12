package appmon

import (
	"log"
	"os"
)

var hostname string

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		log.Fatal("couldn't determine hostname: %s")
	}
}
