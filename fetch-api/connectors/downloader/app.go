package main


import (
	"fmt"

	"connector-downloader/settings"
	_ "connector-downloader/src"
)


func main() {
	fmt.Println(settings.Config.Name)
}
