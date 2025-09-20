package store

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTarGzip(t *testing.T) {
	createTestDirectory := func() {
		os.MkdirAll("temp", os.ModePerm)
		os.WriteFile("temp/1.txt", []byte("hello world 1"), os.ModePerm)
		os.WriteFile("temp/2.txt", []byte("hello world 2"), os.ModePerm)
		os.WriteFile("temp/3.txt", []byte("hello world 3"), os.ModePerm)
	}

	Convey("TestTarGzip", t, func() {
		createTestDirectory()
		defer os.RemoveAll("temp")
		defer os.RemoveAll("temp.tar.gz")
		defer os.RemoveAll("temp1")

		So(createTarGz("temp", "temp.tar.gz"), ShouldBeNil)
		So(extractTarGz("temp.tar.gz", "temp1"), ShouldBeNil)

		txt1, err := os.ReadFile("temp1/1.txt")
		So(err, ShouldBeNil)
		So(string(txt1), ShouldEqual, "hello world 1")

		txt2, err := os.ReadFile("temp1/2.txt")
		So(err, ShouldBeNil)
		So(string(txt2), ShouldEqual, "hello world 2")

		txt3, err := os.ReadFile("temp1/3.txt")
		So(err, ShouldBeNil)
		So(string(txt3), ShouldEqual, "hello world 3")
	})
}
