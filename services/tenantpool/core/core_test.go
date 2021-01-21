package core_test

import (
	"cloudservices/cloudmgmt/apitesthelper"
	"time"

	"github.com/pkg/errors"
)

func init() {
	apitesthelper.StartServices(&apitesthelper.StartServicesConfig{StartPort: 9015})
}

var (
	defaultDeadline = time.Minute * 15
	defaultInterval = time.Second * 30
	timedOutErr     = errors.New("Timed out")
)

func doWithDeadline(deadline time.Duration, interval time.Duration, callback func() (bool, error)) error {
	start := time.Now()
	for {
		exit, err := callback()
		if exit {
			return err
		}
		elapsed := time.Since(start)
		if elapsed > deadline {
			return timedOutErr
		} else {
			time.Sleep(interval)
		}
	}
}
