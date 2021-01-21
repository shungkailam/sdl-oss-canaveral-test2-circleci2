package main

import (
	"cloudservices/account/config"

	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestCoverage(t *testing.T) {

	if *config.Cfg.EnableCodeCoverage {

		sigs := make(chan os.Signal, 1)
		done := make(chan bool, 1)

		signal.Notify(sigs, syscall.SIGUSR1)

		go func() {
			go main()
			sig := <-sigs
			t.Log(sig)
			done <- true
		}()

		t.Log("Starting code coverage...")
		<-done
		t.Log("Exiting code coverage.")
	}
}
