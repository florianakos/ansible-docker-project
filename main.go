package main

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
)

const openFileOptions int = os.O_CREATE | os.O_RDWR
const openFilePermissions os.FileMode = 0660

func openFile(f string) (*os.File, error) {
	newName := "archive/" + strings.ReplaceAll(f, "/", "_") + ".gz"
	return os.OpenFile(newName, openFileOptions, openFilePermissions)
}

func closeFile(handle *os.File) {
	if handle == nil {
		return
	}

	err := handle.Close()
	if err != nil {
		fmt.Println("[ERROR] Closing file:", err)
	}
}

func readFile(s string) []byte {
	bts, err := ioutil.ReadFile(s)
	if err != nil {
		return nil
	}
	return bts
}

func zipFile(filePath string) {
	handle, err := openFile(filePath[10:])
	if err != nil {
		log.Println("[ERROR] Opening file:", err)
	}
	zipWriter, err := gzip.NewWriterLevel(handle, 9)
	if err != nil {
		log.Println("[ERROR] New gzip writer:", err)
	}
	numberOfBytesWritten, err := zipWriter.Write(readFile(filePath))
	if err != nil {
		log.Println("[ERROR] Writing:", err)
	}
	err = zipWriter.Close()
	if err != nil {
		log.Println("[ERROR] Closing zip writer:", err)
	}
	fmt.Println("[INFO] Number of bytes written:", numberOfBytesWritten)

	closeFile(handle)

	///////////////////////////////////////////////////////////////////////////
	// f, err := os.Create()
	// if err != nil {
	// 	return err
	// }
	// defer f.Close()

	// bfw := bufio.NewWriter(f)
	// gw := gzip.NewWriter(bfw)
	// gw.Name = "asd.txt"
	// gw.Comment = "an epic space opera by George Lucas"
	// gw.ModTime = time.Date(1977, time.May, 25, 0, 0, 0, 0, time.UTC)
	// gw.Write([]byte("LOLLOLOLOLOLO"))
	// bfw.Flush()
	// gw.Close()
	// return nil
}

func main() {
	// new watcher instance
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					// log.Println("sending to compress and archive", event.Name)
					zipFile(event.Name)
				} else if event.Op&fsnotify.Create == fsnotify.Create {
					fi, err := os.Stat(event.Name)
					if err != nil {
						return
					}
					if mode := fi.Mode(); mode.IsDir() {
						err = watcher.Add(event.Name)
						if err != nil {
							log.Fatal(err)
						}
					} else {
						log.Println("No new dir...")
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("an error was detected:", err)
			}
		}
	}()

	// set up local directory 'monitored' as the target for watching changes
	err = watcher.Add("monitored/")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
