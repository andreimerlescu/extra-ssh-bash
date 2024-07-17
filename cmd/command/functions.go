package command

import (
	"bytes"
	"log"
	"os"
)

func BB(s string) (b []byte) {
	return bytes.NewBufferString(s).Bytes()
}

func PrintLogs(_logs []string) {
	if len(_logs) > 0 {
		for line, value := range _logs {
			log.Printf("%3d: %v", line, value)
		}
	}
}

func CreateDirectory(dir string) bool {
	return CreateDirectoryWithFileInfo(dir).IsDir()
}

func CreateDirectoryWithFileInfo(dir string) os.FileInfo {
	var dirInfo os.FileInfo
	var err error
	if dirInfo, err = os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Fatal(err)
			return nil
		}
		dirInfo, err = os.Stat(dir)
		if os.IsNotExist(err) {
			log.Fatal("\n\nYOUR JOURNEY HAS ENDED\n\ncannot create directory as requested ", dir, " due to: ", err)
			return nil
		}
	}
	return dirInfo
}
