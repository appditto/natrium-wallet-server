package utils

import (
	"bytes"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"text/tabwriter"

	"github.com/golang/glog"
)

// AssertEqual checks if values are equal
func AssertEqual(t testing.TB, expected, actual interface{}, description ...string) {
	if reflect.DeepEqual(expected, actual) {
		return
	}

	var aType = "<nil>"
	var bType = "<nil>"

	if expected != nil {
		aType = reflect.TypeOf(expected).String()
	}
	if actual != nil {
		bType = reflect.TypeOf(actual).String()
	}

	testName := "AssertEqual"
	if t != nil {
		testName = t.Name()
	}

	_, file, line, _ := runtime.Caller(1)

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 5, ' ', 0)
	fmt.Fprintf(w, "\nTest:\t%s", testName)
	fmt.Fprintf(w, "\nTrace:\t%s:%d", filepath.Base(file), line)
	if len(description) > 0 {
		fmt.Fprintf(w, "\nDescription:\t%s", description[0])
	}
	fmt.Fprintf(w, "\nExpect:\t%v\t(%s)", expected, aType)
	fmt.Fprintf(w, "\nResult:\t%v\t(%s)", actual, bType)

	result := ""
	if err := w.Flush(); err != nil {
		result = err.Error()
	} else {
		result = buf.String()
	}

	if t != nil {
		t.Fatal(result)
	} else {
		glog.Fatal(result)
	}
}

// AssertNotEqual checks if values are not equal
func AssertNotEqual(t testing.TB, expected, actual interface{}, description ...string) {
	if !reflect.DeepEqual(expected, actual) {
		return
	}

	var aType = "<nil>"
	var bType = "<nil>"

	if expected != nil {
		aType = reflect.TypeOf(expected).String()
	}
	if actual != nil {
		bType = reflect.TypeOf(actual).String()
	}

	testName := "AssertNotEqual"
	if t != nil {
		testName = t.Name()
	}

	_, file, line, _ := runtime.Caller(1)

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 5, ' ', 0)
	fmt.Fprintf(w, "\nTest:\t%s", testName)
	fmt.Fprintf(w, "\nTrace:\t%s:%d", filepath.Base(file), line)
	if len(description) > 0 {
		fmt.Fprintf(w, "\nDescription:\t%s", description[0])
	}
	fmt.Fprintf(w, "\nExpect:\t%v\t(%s)", expected, aType)
	fmt.Fprintf(w, "\nResult:\t%v\t(%s)", actual, bType)

	result := ""
	if err := w.Flush(); err != nil {
		result = err.Error()
	} else {
		result = buf.String()
	}

	if t != nil {
		t.Fatal(result)
	} else {
		glog.Fatal(result)
	}
}
