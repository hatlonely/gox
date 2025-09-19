package store

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTarGzip(t *testing.T) {
	createTestDirectory := func() {
		os.MkdirAll("test", os.ModePerm)
		os.WriteFile("test/1.txt", []byte("hello world 1"), os.ModePerm)
		os.WriteFile("test/2.txt", []byte("hello world 2"), os.ModePerm)
		os.WriteFile("test/3.txt", []byte("hello world 3"), os.ModePerm)
	}

	Convey("TestTarGzip", t, func() {
		createTestDirectory()
		defer os.RemoveAll("test")
		defer os.RemoveAll("test.tar.gz")
		defer os.RemoveAll("test1")

		So(createTarGz("test", "test.tar.gz"), ShouldBeNil)
		So(extractTarGz("test.tar.gz", "test1"), ShouldBeNil)

		txt1, err := os.ReadFile("test1/1.txt")
		So(err, ShouldBeNil)
		So(string(txt1), ShouldEqual, "hello world 1")

		txt2, err := os.ReadFile("test1/2.txt")
		So(err, ShouldBeNil)
		So(string(txt2), ShouldEqual, "hello world 2")

		txt3, err := os.ReadFile("test1/3.txt")
		So(err, ShouldBeNil)
		So(string(txt3), ShouldEqual, "hello world 3")
	})
}
