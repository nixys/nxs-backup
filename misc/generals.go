package misc

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"nxs-backup/modules/logger"
)

const (
	YearlyBackupDay  = "1"
	MonthlyBackupDay = "1"
	WeeklyBackupDay  = "0"
	IncBackupType    = "inc_files"
)

var DecadesBackupDays = []string{"1", "11", "21"}

func GetOfsPart(regex, target string) string {
	var pathParts []string

	regexParts := strings.Split(regex, "/")
	targetParts := strings.Split(target, "/")

	for i, p := range regexParts {
		if p != targetParts[i] {
			pathParts = append(pathParts, targetParts[i])
		}
	}

	if len(pathParts) > 0 {
		return strings.Join(pathParts, "___")
	}

	return targetParts[len(targetParts)-1]
}

func GetDateTimeNow(unit string) (res string) {

	currentTime := time.Now()

	switch unit {
	case "dom":
		res = strconv.Itoa(currentTime.Day())
	case "dow":
		res = strconv.Itoa(int(currentTime.Weekday()))
	case "doy":
		res = strconv.Itoa(currentTime.YearDay())
	case "moy":
		res = strconv.Itoa(int(currentTime.Month()))
	case "year":
		res = strconv.Itoa(currentTime.Year())
	case "previous_year":
		res = strconv.Itoa(currentTime.Year() - 1)
	default:
		res = currentTime.Format("2006-01-2_15-04")
	}

	return res
}

func GetDecadeDaySubdir() (decadeDay string) {
	intDom, _ := strconv.Atoi(GetDateTimeNow("dom"))
	if intDom < 11 {
		decadeDay = "day_01"
	} else if intDom > 20 {
		decadeDay = "day_21"
	} else {
		decadeDay = "day_11"
	}
	return
}

func GetFileFullPath(dirPath, baseName, baseExtension, prefix string, gZip bool) (fullPath string) {

	fileName := fmt.Sprintf("%s_%s.%s", baseName, GetDateTimeNow(""), baseExtension)

	if prefix != "" {
		fileName = fmt.Sprintf("%s-%s", prefix, fileName)
	}

	if gZip {
		fileName += ".gz"
	}

	fullPath = filepath.Join(dirPath, fileName)

	return fullPath
}

// Contains checks if a string is present in a slice
func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// RandString generates random string
func RandString(strLen int64) string {
	var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	rand.Seed(time.Now().UnixNano())
	b := make([]rune, strLen)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}

	return string(b)
}

// GetMessage generates notification message from event log record
func GetMessage(n logger.LogRecord, project, server string) (m string) {

	switch n.Level {
	case logrus.DebugLevel:
		m += "[DEBUG]\n\n"
	case logrus.InfoLevel:
		m += "[INFO]\n\n"
	case logrus.WarnLevel:
		m += "⚠️[WARNING]\n\n"
	case logrus.ErrorLevel:
		m += "‼️[ERROR]\n\n"
	}

	if project != "" {
		m += fmt.Sprintf("Project: %s\n", project)
	}
	if server != "" {
		m += fmt.Sprintf("Server: %s\n\n", server)
	}

	if n.JobName != "" {
		m += fmt.Sprintf("Job: %s\n", n.JobName)
	}
	if n.StorageName != "" {
		m += fmt.Sprintf("Storage: %s\n", n.StorageName)
	}
	m += fmt.Sprintf("\nMessage: %s\n", n.Message)

	return
}
