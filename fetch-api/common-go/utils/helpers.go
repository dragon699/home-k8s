package utils

import (
	"fmt"
	"strings"
	"math"
	"os"
	"strconv"
	"time"

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

func HasItemWithPrefix(items []string, prefix string) bool {
	for _, item := range items {
		if strings.HasPrefix(item, prefix) {
			return true
		}
	}

	return false
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

func TimeGetOrdinalDay(day int) string {
	if day >= 11 && day <= 13 {
		return "th"
	}

	switch day % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}

func TimeFromUnix(unixTime int64) string {
	timeObj := time.Unix(unixTime, 0)
	day := timeObj.Day()

	return fmt.Sprintf(
		"%d%s of %s %d at %s",
		day,
		TimeGetOrdinalDay(day),
		timeObj.Format("Jan"),
		timeObj.Year(),
		timeObj.Format("15:04"),
	)
}

func BytesToMegabytes(bytes int64) int64 {
	return bytes / (1024 * 1024)
}

func RoundToTwoDecimals(value float64) float64 {
	return math.Round(value*100) / 100
}

func BytesPerSecondToMBps(bytesPerSec int64) float64 {
	mbps := float64(bytesPerSec) / (1024 * 1024)
	return RoundToTwoDecimals(mbps)
}

func SecondsToMinutes(seconds int64) int64 {
	if seconds < 60 {
		return 1
	}
	return seconds / 60
}

// func RenameMovieFile(filePath string) error {
// 	pattern := regexp.MustCompile(``)

// 	return nil
// }
