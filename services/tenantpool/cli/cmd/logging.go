package cmd

import (
	"fmt"
	"os"
)

var verbose = 0

type V int

func (v V) Infoln(msg ...interface{}) {
	if int(v) <= verbose {
		fmt.Println(msg...)
	}
}

func (v V) Infof(format string, args ...interface{}) {
	if int(v) <= verbose {
		buf := fmt.Sprintf(format, args...)
		fmt.Println(buf)
	}
}

func Infoln(msg ...interface{}) {
	V(0).Infoln(msg...)
}

func Infof(format string, args ...interface{}) {
	V(0).Infof(format, args...)
}

func Errorln(msg ...interface{}) {
	fmt.Fprintln(os.Stderr, msg...)
}

func Errorf(format string, args ...interface{}) {
	buf := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, buf)
}

func Fatalln(msg ...interface{}) {
	Errorln(msg...)
	os.Exit(1)
}

func Fatalf(format string, args ...interface{}) {
	Errorf(format, args...)
	os.Exit(1)
}
