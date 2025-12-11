package utils

import (
	"fmt"
	"os"
	"strconv"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)


func ToInt(val string) (int64, error) {
	i, err := strconv.Atoi(val)

	if err != nil {
		return 0, fmt.Errorf("expected int-like value, got %s", val)
	}
	return int64(i), nil
}


func ToBool(val string) (bool, error) {
	b, err := strconv.ParseBool(val)

	if err != nil {
		return false, fmt.Errorf("expected bool-like value, got %s", val)
	}
	return b, nil
}


func ReadFile(path string) (string, error) {
	contents, err := os.ReadFile(path)

	if err != nil {
		return "", err
	}

	return string(contents), nil
}


func WriteFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}


func TrimKubeTime(time metav1.Time) string {
	return time.Time.Format("2006-01-02T15:04:05")
}
