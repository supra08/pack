package layer

import (
	"archive/tar"
	"io"
	"path"
	"strings"
)

type WindowsWriter struct {
	tarWriter *tar.Writer
}

func NewWindowsWriter(dataWriter io.WriteCloser) *WindowsWriter {
	return &WindowsWriter{tar.NewWriter(dataWriter)}
}

func (w WindowsWriter) Write(content []byte) (int, error) {
	return w.tarWriter.Write(content)
}

func (w WindowsWriter) WriteHeader(header *tar.Header) error {
	if err := w.initializeLayer(); err != nil {
		return err
	}

	err := w.writeParentPaths(header.Name)
	if err != nil {
		return err
	}

	header.Name = layerFilesPath(header.Name)

	return w.tarWriter.WriteHeader(header)
}

func (w WindowsWriter) writeParentPaths(childPath string) error {
	parentDir := path.Dir(childPath)
	shallowestDir := ""
	for _, pathPart := range strings.Split(parentDir, "/") {
		shallowestDir = path.Join(shallowestDir, pathPart)

		// skip root, already initialized
		if shallowestDir == "" {
			continue
		}

		if err := w.tarWriter.WriteHeader(&tar.Header{
			Name:     layerFilesPath(shallowestDir),
			Typeflag: tar.TypeDir,
		}); err != nil {
			return err
		}
	}
	return nil
}

func layerFilesPath(origPath string) string {
	return path.Join("Files", origPath)
}

func (w WindowsWriter) initializeLayer() error {
	if err := w.tarWriter.WriteHeader(&tar.Header{
		Name:     "Files",
		Typeflag: tar.TypeDir,
	}); err != nil {
		return err
	}
	if err := w.tarWriter.WriteHeader(&tar.Header{
		Name:     "Hives",
		Typeflag: tar.TypeDir,
	}); err != nil {
		return err
	}
	return nil
}

func (w WindowsWriter) Close() error {
	return w.tarWriter.Close()
}
