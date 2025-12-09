package main


import (
	"fmt"

	"connector-kube/settings"
)


func main() {
	fmt.Println(settings.Config.Name)
	fmt.Println(settings.Config.OtelServiceNamespace)
	fmt.Println(settings.Config.OtelServiceVersion)
	fmt.Println(settings.Config.HealthEndpoint)
	fmt.Println(settings.Config.KubeConfigPath)
}
