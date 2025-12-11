package main


import (
	"fmt"

	"connector-kube/settings"
	_ "connector-kube/src"
)


func main() {
	fmt.Println(settings.Config.Name)
}
