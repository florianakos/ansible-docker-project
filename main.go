package main

import (
	"compress/gzip"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

// some constants related to files and permissions on them
const CREATE_READ_WRITE int = os.O_CREATE | os.O_RDWR
const OPEN_FILE_PERMISSION os.FileMode = 0660
const APPEND_OR_CREATE_READONLY int = os.O_APPEND | os.O_CREATE | os.O_WRONLY

// helper func that closes a file descriptor properly
func closeFile(handle *os.File) {
	if handle == nil {
		return
	}
	err := handle.Close()
	if err != nil {
		fmt.Println("[ERROR] Closing file:", err)
	}
}

// convenience function that returns the contents of a file as a byte slice
func extractFileContents(s string) []byte {
	bts, err := ioutil.ReadFile(s)
	if err != nil {
		return nil
	}
	return bts
}

// helper function that is configured to log to a local file in directory
func logLocal(msg string) {
	// open file for writing, create if doesnt exist
	f, err := os.OpenFile("service_history.log", APPEND_OR_CREATE_READONLY, 0644)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()

	// write log msg to file
	f.WriteString(time.Now().Format("2006-01-02 15:04:05") + ": " + msg)
}

// helper function that returns the size of a file given by its relative path
func getFileSize(path string) float64 {
	// open file for reading
	f, _ := os.Open(path)
	defer f.Close()

	// get file stats and return size as float64
	s, _ := f.Stat()
	return float64(s.Size())
}

// function to handle the compression and logging for statistics
func processFile(oldName string) {
	//time.Sleep(1000*time.Nanosecond)
	
	// construct filename for new gzipped file
	newName := "archive/" + strings.ReplaceAll(oldName[10:], "/", "_") + ".gz"

	// create the file for writing
	handle, err := os.OpenFile(newName, CREATE_READ_WRITE, OPEN_FILE_PERMISSION)
	if err != nil {
		log.Fatalln("[ERROR] Opening file:", err)
		return
	}
	defer closeFile(handle)

	// create the GZIP writer
	zipWriter, err := gzip.NewWriterLevel(handle, 9)
	if err != nil {
		log.Println("[ERROR] trying to create gzip writer:", err)
		return
	}
	// read concents of original file and write to gzipped file
	_, err = zipWriter.Write(extractFileContents(oldName))
	if err != nil {
		log.Println("[ERROR] trying to write with gzip:", err)
		return
	}
	defer zipWriter.Close()

	// calc and log compression rate
	oS := getFileSize(oldName)
	nS := getFileSize(newName)
	compRate := ((oS - nS) / oS) * 100.0

	// log the result of file operation
	logLocal(fmt.Sprintf("New file put to archive 'archive/%s' with compression rate: %.4f%%\n", strings.ReplaceAll(newName[8:], "/", "_"), compRate))
}

func main() {
	// Initial startup message
	log.Println("File watcher is starting up ...")
	
	// create new watcher instance
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	
	// create a channel that will be blocking the main routine indefinitely
	done := make(chan bool)
	
	// while in a parallel goroutine we look for events incoming from fsnotify
	go func() {
		log.Println("File watcher started monitoring!")
		for {
			select {
			// catch valid Events from fsnotify
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// a tradeoff exists between listening for Create vs Write!!!
				if event.Op&fsnotify.Create == fsnotify.Create {
					fi, _ := os.Stat(event.Name)
					// if the object created was a folder, then add it to the watcher
					if mode := fi.Mode(); mode.IsDir() {
						watcher.Add(event.Name)
					// if the object created is not dir, then its a file so we process it
					} else {
						go processFile(event.Name)
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

	// by default the local directory is set up for notifications
	err = watcher.Add("monitored/")
	if err != nil {
		log.Fatal(err)
	}
	
	// blocking indefinitely
	<-done
}