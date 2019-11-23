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

func logItAll(msg string) {
	log.Print(msg)
	f, err := os.OpenFile("service_history.log",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()
	if _, err := f.WriteString(msg); err != nil {
		log.Println(err)
	}
}

func percentage(old, new float64) (delta float64) {
	diff := float64(old - new)
	delta = (diff / float64(old)) * 100
	return
}

func compressionRateCalc(old string, new string) {
	f1, err := os.Open(old)
	if err != nil {
		return
	}
	defer f1.Close()

	f2, err := os.Open(new)
	if err != nil {
		return
	}
	defer f2.Close()

	s1, err := f1.Stat()
	if err != nil {
		return
	}

	s2, err := f2.Stat()
	if err != nil {
		return
	}
	oldSize := float64(s1.Size() + 0.0)
	newSize := float64(s2.Size() + 0.0)

	logItAll(fmt.Sprintf("File: '%s' (size: %.2f KB), archived as 'archive/%s' (size: %.2f KB) compression rate: %.4f%%\n", old, oldSize/1024.0, strings.ReplaceAll(new[8:], "/", "_"), newSize/1024.0, percentage(oldSize, newSize)))
}

func zipAndArchive(filePath string) {
	oldName := filePath
	newName := "archive/" + strings.ReplaceAll(filePath[10:], "/", "_") + ".gz"

	handle, err := os.OpenFile(newName, openFileOptions, openFilePermissions)
	if err != nil {
		log.Fatalln("[ERROR] Opening file:", err)
		return
	}
	zipWriter, err := gzip.NewWriterLevel(handle, 9)
	if err != nil {
		log.Println("[ERROR] New gzip writer:", err)
		return
	}
	_, err = zipWriter.Write(readFile(filePath))
	if err != nil {
		log.Println("[ERROR] Writing:", err)
		return
	}
	err = zipWriter.Close()
	if err != nil {
		log.Println("[ERROR] Closing zip writer:", err)
		return
	}
	// fmt.Println("[INFO] Number of bytes written:", numberOfBytesWritten)
	closeFile(handle)
	// calc and log compression compr rate
	compressionRateCalc(oldName, newName)
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
				// if new file was written into the watched folder
				if event.Op&fsnotify.Write == fsnotify.Write {
					// log.Println("sending to compress and archive", event.Name)
					zipAndArchive(event.Name)
				// if new folder was created in watched folder add it as watched too
				} else if event.Op&fsnotify.Create == fsnotify.Create {
					fi, _ := os.Stat(event.Name)
					if mode := fi.Mode(); mode.IsDir() {
						err = watcher.Add(event.Name)
						if err != nil {
							log.Println(err)
							return
						}
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
