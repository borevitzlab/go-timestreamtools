package utils

import (
	"encoding/json"
	"fmt"
	"github.com/rwcarlsen/goexif/exif"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const (
	// ArchiveForm is the form that tar files should take (YYYY-MM-DD)
	ArchiveForm                 = "%s~2006-01-02.tar"
	// DefaultTsDirectoryStructure is the default directory structure for timestreams
	DefaultTsDirectoryStructure = "2006/2006_01/2006_01_02/2006_01_02_15/"
	// TsForm is the timestamp form for individual files.
	TsForm                      = "2006_01_02_15_04_05"
	dumbExifForm                = "2006:01:02 15:04:05"
	tsRegexPattern              = "[0-9][0-9][0-9][0-9]_[0-1][0-9]_[0-3][0-9]_[0-2][0-9]_[0-5][0-9]_[0-5][0-9]"
)

const (
	OS_READ = 04
	OS_WRITE = 02
	OS_EX = 01
	OS_USER_SHIFT = 6
	OS_GROUP_SHIFT = 3
	OS_OTH_SHIFT = 0

	OS_USER_R = OS_READ<<OS_USER_SHIFT
	OS_USER_W = OS_WRITE<<OS_USER_SHIFT
	OS_USER_X = OS_EX<<OS_USER_SHIFT
	OS_USER_RW = OS_USER_R | OS_USER_W
	OS_USER_RWX = OS_USER_RW | OS_USER_X

	OS_GROUP_R = OS_READ<<OS_GROUP_SHIFT
	OS_GROUP_W = OS_WRITE<<OS_GROUP_SHIFT
	OS_GROUP_X = OS_EX<<OS_GROUP_SHIFT
	OS_GROUP_RW = OS_GROUP_R | OS_GROUP_W
	OS_GROUP_RWX = OS_GROUP_RW | OS_GROUP_X

	OS_OTH_R = OS_READ<<OS_OTH_SHIFT
	OS_OTH_W = OS_WRITE<<OS_OTH_SHIFT
	OS_OTH_X = OS_EX<<OS_OTH_SHIFT
	OS_OTH_RW = OS_OTH_R | OS_OTH_W
	OS_OTH_RWX = OS_OTH_RW | OS_OTH_X

	OS_ALL_R = OS_USER_R | OS_GROUP_R | OS_OTH_R
	OS_ALL_W = OS_USER_W | OS_GROUP_W | OS_OTH_W
	OS_ALL_X = OS_USER_X | OS_GROUP_X | OS_OTH_X
	OS_ALL_RW = OS_ALL_R | OS_ALL_W
	OS_ALL_RWX = OS_ALL_RW | OS_GROUP_X
)

// EmitPath writes a filepath to stdout
func EmitPath(a ...interface{}) (n int, err error) {
	return fmt.Fprintln(os.Stdout, a...)
}

// TsRegex is a regexp to find a timestamp within a filename
var /* const */ TsRegex = regexp.MustCompile(tsRegexPattern)

// ParseExifDatetime parses a datetime string from the old dumb exif way to a time.Time{}
func ParseExifDatetime(datetimeString string) (time.Time, error) {
	thisTime, err := time.Parse(dumbExifForm, datetimeString)
	if err != nil {
		return time.Time{}, err
	}
	return thisTime, nil
}

type exifFromJSON struct {
	DateTime          string
	DateTimeOriginal  string
	DateTimeDigitized string
}

// GetTimeFromExif gets a time.Time from either the exif in an image, or the exif json for that image
func GetTimeFromExif(thisFile string) (datetime time.Time, err error) {

	var datetimeString string
	if _, ferr := os.Stat(thisFile + ".json"); ferr == nil {
		eData := exifFromJSON{}
		//	do something with the json.

		byt, err := ioutil.ReadFile(thisFile + ".json")
		if err != nil {
			return time.Time{}, err
		}
		if err := json.Unmarshal(byt, &eData); err != nil {
			return time.Time{}, err
		}
		datetimeString = eData.DateTime

	} else {
		fileHandler, err := os.Open(thisFile)
		if err != nil {
			// file wouldnt open
			return time.Time{}, err
		}
		exifData, err := exif.Decode(fileHandler)
		if err != nil {
			// exif wouldnt decode
			return time.Time{}, fmt.Errorf("[exif] couldn't decode exif from image %s", err)
		}
		dt, err := exifData.Get(exif.DateTime) // normally, don't ignore errors!
		if err != nil {
			// couldnt get DateTime from exifex
			return time.Time{}, err
		}
		datetimeString, err = dt.StringVal()
		if err != nil {
			// couldnt get
			return time.Time{}, err
		}
	}
	if datetime, err = ParseExifDatetime(datetimeString); err != nil {
		return
	}
	return
}

// GetTimeFromFileTimestamp gets a time.Time from the timestamp of an image
func GetTimeFromFileTimestamp(thisFile string) (time.Time, error) {
	timestamp := TsRegex.FindString(thisFile)
	if len(timestamp) < 1 {
		// no timestamp found in filename
		return time.Time{}, fmt.Errorf("failed regex timestamp from filename %s", thisFile)
	}

	t, err := time.Parse(TsForm, timestamp)
	if err != nil {
		// parse error
		return time.Time{}, err
	}
	return t, nil
}

// MoveFilebyCopy either copies a file of moves it depending on whether the del argument is true or false
func MoveFilebyCopy(src, dst string, del bool) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	// no need to check errors on read only file, we already got everything
	// we need from the filesystem, so nothing can go wrong now.
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	fileMode := os.FileMode(OS_USER_RW|OS_GROUP_RW)
	d.Chmod(fileMode)

	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	if del {
		absSrc, _ := filepath.Abs(src)
		absDest, _ := filepath.Abs(dst)
		if absSrc != absDest {
			os.Remove(src)
		}
	}
	return d.Close()
}
