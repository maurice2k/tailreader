# tailreader

_tailreader_ is a Go package that implements an `io.Reader` that tails a file.

It is intended to be used for binary files that are being written to by another process. It does not implement any kind of "line" reading and thus is fully compatible with the `io.Reader` interface and agnostic to the file's content.

## Installation


```bash
go get -u github.com/maurice2k/tailreader
```

## Usage

```go
package main

import (
	"io"
	"log"
	"time"

	"github.com/maurice2k/tailreader"
)

func main() {
	tr, err := tailreader.NewTailingReader(
		"/tmp/test/the-file-to-tail",
		tailreader.WithWaitForFile(true, 30*time.Second), // wait for file to appear, but only for 30 seconds
		tailreader.WithIdleTimeout(60*time.Second),       // close reader if no data is read for 60 seconds
		tailreader.WithCloseOnDelete(true),               // close reader if file is deleted
	)
	if err != nil {
		log.Fatal(err)
	}
	defer tr.Close()

	// if you don't want to wait within tr.Read() you can call tr.WaitForFile()
	// upfront and then start reading, which will block until the file is available
	// or the timeout is reached

	//err = tr.WaitForFile()
	//if err != nil {
	//	log.Fatal(err)
	//}

	buf := make([]byte, 1024)
	for {
		n, err := tr.Read(buf)

		if err == io.EOF {
			// there is no more data to read
			break
		} else if err != nil {
			log.Fatal(err)
		}

		log.Print(string(buf[:n]))
	}
}
```

## License

*tailreader* is available under the MIT [license](LICENSE).
