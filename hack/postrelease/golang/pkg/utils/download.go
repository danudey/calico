package utils

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/cheggaaa/pb/v3"
)

type WriteCounter struct {
	Total        uint64
	DownloadSize uint64
	ProgressBar  *pb.ProgressBar
}

// PrintProgress prints the progress of a file write
func (wc WriteCounter) PrintProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	// fmt.Printf("\r%s", strings.Repeat(" ", 50))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	// fmt.Printf("\rDownloading... %s/%s complete", humanize.Bytes(wc.Total), humanize.Bytes(wc.DownloadSize))

	wc.ProgressBar.Add64(int64(wc.Total))
	wc.ProgressBar.Increment()
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total = uint64(n)
	wc.PrintProgress()
	return n, nil
}

func MaybeDownloadFile(url string, filepath string) error {

	if stat, err := os.Stat(filepath + ".tmp"); err == nil {
		resp, err := http.Head(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Size
		size, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
		downloadSize := int64(size)

		if stat.Size() == downloadSize {
			// We already have the file
			return nil
		} else if stat.Size() < downloadSize {
			fmt.Printf("Need to resume from %s\n", stat.Size())

		}

	} else if errors.Is(err, os.ErrNotExist) {
		// path/to/whatever does *not* exist
		return DownloadFile(url, filepath)
	} else {
		return fmt.Errorf("could not maybe download file: %w", err)
	}

	return nil
}

func DownloadFile(url string, filepath string) error {
	// Create the file with .tmp extension, so that we won't overwrite a
	// file until it's downloaded fully
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Size
	size, _ := strconv.Atoi(resp.Header.Get("Content-Length"))
	downloadSize := uint64(size)

	tmpl := `{{string . "filename" }} {{ bar . }} {{speed . }} {{percent .}}`

	// Progress Bar
	bar := pb.New64(int64(downloadSize))
	bar.SetMaxWidth(100)
	bar.SetWriter(os.Stdout)
	bar.Set(pb.Bytes, true)
	bar.SetTemplateString(tmpl)
	bar.Set("filename", filepath)
	bar.Start()
	defer bar.Finish()

	// Create our bytes counter and pass it to be used alongside our writer
	counter := &WriteCounter{
		DownloadSize: downloadSize,
		ProgressBar:  bar,
	}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return err
	}

	// The progress use the same line so print a new line once it's finished downloading
	// fmt.Println()

	// Rename the tmp file back to the original file
	err = os.Rename(filepath+".tmp", filepath)
	if err != nil {
		return err
	}

	return nil
}
