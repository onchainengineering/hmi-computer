package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func generateIconList() int {
	files, err := os.ReadDir("site/static/icon")
	if err != nil {
		_, _ = fmt.Println("failed to read icon/ directory")
		_, _ = fmt.Println("err:", err.Error())
		return 71 // OSERR
	}

	icons := make([]string, len(files))
	i := 0
	for _, file := range files {
		if !file.Type().IsRegular() {
			continue
		}

		icons[i] = file.Name()
		i++
	}
	icons = icons[:i]

	outputFile, err := os.Create("./site/static/icons.json")
	if err != nil {
		_, _ = fmt.Println("failed to create file")
		_, _ = fmt.Println("err:", err.Error())
		return 73 // CANTCREAT
	}
	defer outputFile.Close()

	iconsJSON, err := json.Marshal(icons)
	if err != nil {
		_, _ = fmt.Println("failed to serialize JSON")
		_, _ = fmt.Println("err:", err.Error())
		return 70 // SOFTWARE
	}

	written, err := outputFile.Write(iconsJSON)
	if err != nil || written != len(iconsJSON) {
		_, _ = fmt.Println("failed to write JSON")
		if err != nil {
			_, _ = fmt.Println("err:", err.Error())
		}
		return 74 // IOERR
	}

	return 0
}
