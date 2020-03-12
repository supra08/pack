package layer_test

import (
	"archive/tar"
	"github.com/buildpacks/pack/internal/layer"
	h "github.com/buildpacks/pack/testhelpers"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestWindowsWriter(t *testing.T) {
	spec.Run(t, "windows-writer", testWindowsWriter, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testWindowsWriter(t *testing.T, when spec.G, it spec.S) {
	when("#WriteHeader", func() {
		it("writes with parent directoriess", func() {
			var err error

			f, err := ioutil.TempFile("", "windows-writer.tar")
			h.AssertNil(t, err)
			defer func() { f.Close(); os.Remove(f.Name()) }()

			lw := layer.NewWindowsWriter(f)

			err = lw.WriteHeader(&tar.Header{
				Name:     "/cnb/lifecycle/my-file",
				Typeflag: tar.TypeReg,
			})
			h.AssertNil(t, err)

			err = lw.Close()
			h.AssertNil(t, err)

			f.Seek(0, 0)
			tr := tar.NewReader(f)

			th, _ := tr.Next()
			h.AssertEq(t, th.Name, "Files")

			th, _ = tr.Next()
			h.AssertEq(t, th.Name, "Hives")

			th, _ = tr.Next()
			h.AssertEq(t, th.Name, "Files/cnb")

			th, _ = tr.Next()
			h.AssertEq(t, th.Name, "Files/cnb/lifecycle")

			th, _ = tr.Next()
			h.AssertEq(t, th.Name, "Files/cnb/lifecycle/my-file")
			h.AssertEq(t, th.Typeflag, uint8(tar.TypeReg))

			_, err = tr.Next()
			h.AssertEq(t, err, io.EOF)
		})
	})

	when("#Close", func() {
		it("writes required parent dirs on empty image", func() {
			var err error

			f, err := ioutil.TempFile("", "windows-writer.tar")
			h.AssertNil(t, err)
			defer func() { f.Close(); os.Remove(f.Name()) }()

			lw := layer.NewWindowsWriter(f)

			err = lw.WriteHeader(&tar.Header{
				Name:     "/cnb/lifecycle/my-file",
				Typeflag: tar.TypeReg,
			})
			h.AssertNil(t, err)

			err = lw.Close()
			h.AssertNil(t, err)

			f.Seek(0, 0)
			tr := tar.NewReader(f)

			th, _ := tr.Next()
			h.AssertEq(t, th.Name, "Files")

			th, _ = tr.Next()
			h.AssertEq(t, th.Name, "Hives")

			_, err = tr.Next()
			h.AssertTrue(t, err, io.EOF)
		})
	})
}
