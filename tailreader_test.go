package tailreader

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTailingReader(t *testing.T) {
	file, _ := os.CreateTemp("", "test")
	defer os.Remove(file.Name())

	_, err := NewTailingReader(file.Name())
	assert.NoError(t, err)
}

func TestNewTailingReaderWithOptions(t *testing.T) {
	file, _ := os.CreateTemp("", "test")
	defer os.Remove(file.Name())

	tr, err := NewTailingReader(file.Name(), WithCloseOnDelete(true), WithCloseOnTruncate(true))
	assert.NoError(t, err)
	assert.True(t, tr.options.CloseOnDelete)
	assert.True(t, tr.options.CloseOnTruncate)
}
func TestTailingReader_Read(t *testing.T) {
	file, _ := os.CreateTemp("", "test")
	defer os.Remove(file.Name())

	tr, _ := NewTailingReader(file.Name())
	defer tr.Close()

	str := "Hello, World!"
	_, err := file.WriteString(str)
	assert.NoError(t, err)

	buf := make([]byte, 128)
	n, err := tr.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(str), n)
	assert.Equal(t, str, string(buf[:n]))
}

func TestTailingReader_ReadAfterFileTruncatedWithCloseOnTruncate(t *testing.T) {
	file, _ := os.CreateTemp("", "test")
	defer os.Remove(file.Name())

	tr, _ := NewTailingReader(file.Name(), WithCloseOnTruncate(true))
	defer tr.Close()

	str := "Hello, World!"
	_, err := file.WriteString(str)
	assert.NoError(t, err)

	buf := make([]byte, 128)
	n, err := tr.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(str), n)
	assert.Equal(t, str, string(buf[:n]))

	file.Truncate(0)
	n, err = tr.Read(buf)
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 0, n)
}

func TestTailingReader_ReadAfterFileTruncatedWithoutCloseOnTruncate(t *testing.T) {
	file, _ := os.CreateTemp("", "test")
	defer os.Remove(file.Name())

	tr, _ := NewTailingReader(file.Name(), WithCloseOnTruncate(false))
	defer tr.Close()

	str := "Hello, World!"
	_, err := file.WriteString(str)
	assert.NoError(t, err)

	buf := make([]byte, 128)
	n, err := tr.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(str), n)
	assert.Equal(t, str, string(buf[:n]))

	file.Truncate(0)
	str = "Hello, World again!"
	_, err = file.WriteString(str)

	buf = make([]byte, 128)
	n, err = tr.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(str), n)
	assert.Equal(t, str, string(buf[:n]))
}

func TestTailingReader_ReadAfterFileDeleted(t *testing.T) {
	file, _ := os.CreateTemp("", "test")

	tr, _ := NewTailingReader(file.Name(), WithCloseOnDelete(true), WithWaitForFile(true, 0))
	defer tr.Close()

	str := "Hello, World!"
	_, err := file.WriteString(str)
	assert.NoError(t, err)

	buf := make([]byte, 128)
	n, err := tr.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(str), n)
	assert.Equal(t, str, string(buf[:n]))

	err = os.Remove(file.Name())

	n, err = tr.Read(buf)
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 0, n)
}

func TestTailingReader_ReadWithIdleTimeout(t *testing.T) {
	file, _ := os.CreateTemp("", "test")
	defer os.Remove(file.Name())

	tr, _ := NewTailingReader(file.Name(), WithIdleTimeout(1*time.Second))
	defer tr.Close()

	str := "Hello, World!"
	_, err := file.WriteString(str)
	assert.NoError(t, err)

	buf := make([]byte, 128)
	n, err := tr.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(str), n)
	assert.Equal(t, str, string(buf[:n]))

	// nothing to read; should trigger an idle timeout after 1 second
	n, err = tr.Read(buf)
	assert.Equal(t, ErrIdleTimeout, err)
	assert.Equal(t, 0, n)
}

func TestTailingReader_Close(t *testing.T) {
	file, _ := os.CreateTemp("", "test")
	defer os.Remove(file.Name())

	tr, _ := NewTailingReader(file.Name())
	err := tr.Close()
	assert.NoError(t, err)
	assert.Nil(t, tr.file)
	assert.Nil(t, tr.watcher)
}
