package geophoto

import (
	"fmt"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	GeoPhotoDataWalk = make(map[int]GeoPhoto) // Global var for the walk in DirGeoPhotoData
)

type GeoPhoto struct {
	GPSLatitude, GPSLatitudeRef, GPSLongitude, GPSLongitudeRef, GPSTimeStamp, GPSDateStamp *tiff.Tag
}

func NewGeoPhotoFromExif(exifData *exif.Exif) GeoPhoto {
	GPSLatitude, _ := exifData.Get(exif.GPSLatitude)
	GPSLatitudeRef, _ := exifData.Get(exif.GPSLatitudeRef)
	GPSLongitude, _ := exifData.Get(exif.GPSLongitude)
	GPSLongitudeRef, _ := exifData.Get(exif.GPSLongitudeRef)
	GPSTimeStamp, _ := exifData.Get(exif.GPSTimeStamp)
	GPSDateStamp, _ := exifData.Get(exif.GPSDateStamp)

	return GeoPhoto{
		GPSLatitude:     GPSLatitude,
		GPSLatitudeRef:  GPSLatitudeRef,
		GPSLongitude:    GPSLongitude,
		GPSLongitudeRef: GPSLongitudeRef,
		GPSTimeStamp:    GPSTimeStamp,
		GPSDateStamp:    GPSDateStamp,
	}
}

func NewGeoPhotoFromFile(path string) (GeoPhoto, error) {
	file, err := os.Open(path)
	if err != nil {
		return GeoPhoto{}, err
	}
	defer file.Close()

	exifData, err := exif.Decode(file)
	if err != nil {
		return GeoPhoto{}, err
	}

	return NewGeoPhotoFromExif(exifData), nil
}

func (geo *GeoPhoto) StringDegrees() string {
	latitude := sexToDec(geo.GPSLatitude.Rat(0), geo.GPSLatitude.Rat(1), geo.GPSLatitude.Rat(2), geo.GPSLatitudeRef.StringVal())
	longitude := sexToDec(geo.GPSLongitude.Rat(0), geo.GPSLongitude.Rat(1), geo.GPSLongitude.Rat(2), geo.GPSLongitudeRef.StringVal())

	// http://stackoverflow.com/questions/7167604/how-accurately-should-i-store-latitude-and-longitude
	return latitude.FloatString(6) + "," + longitude.FloatString(6)
}

func (geo *GeoPhoto) Unix() int64 {
	if geo.GPSTimeStamp == nil || geo.GPSDateStamp == nil {
		return 0
	}

	dateParts := strings.Split(geo.GPSDateStamp.StringVal(), ":")
	year, _ := strconv.Atoi(dateParts[0])
	month, _ := strconv.Atoi(dateParts[1])
	day, _ := strconv.Atoi(dateParts[2])

	hour := int(geo.GPSTimeStamp.Rat(0).Num().Int64())
	minute := int(geo.GPSTimeStamp.Rat(1).Num().Int64())
	second := int(geo.GPSTimeStamp.Rat(2).Num().Int64())

	loc, err := time.LoadLocation("UTC")
	if err != nil {
		return 0
	}

	monthConstants := map[int]time.Month{
		1:  time.January,
		2:  time.February,
		3:  time.March,
		4:  time.April,
		5:  time.May,
		6:  time.June,
		7:  time.July,
		8:  time.August,
		9:  time.September,
		10: time.October,
		11: time.November,
		12: time.December,
	}

	t := time.Date(year, monthConstants[month], day, hour, minute, second, 0, loc)
	return t.Unix()
}

func (geo *GeoPhoto) StreetViewUrl() string {
	return fmt.Sprintf("https://maps.googleapis.com/maps/api/streetview?location=%s&size=640x640&fov=120&heading=0&sensor=false", geo.StringDegrees())
}

func DirGeoPhotoData(dir string) map[int]GeoPhoto {
	// Re-init the walk map
	GeoPhotoDataWalk = make(map[int]GeoPhoto)
	filepath.Walk(dir, walk)
	return GeoPhotoDataWalk
}

func walk(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return nil
	}

	if fi.Mode().IsDir() {
		return nil
	}

	geo, err := NewGeoPhotoFromFile(path)
	if err != nil {
		return nil
	}

	if geo.GPSTimeStamp == nil || geo.GPSDateStamp == nil || geo.GPSLatitude == nil || geo.GPSLatitudeRef == nil || geo.GPSLongitude == nil || geo.GPSLongitudeRef == nil {
		return nil
	}

	GeoPhotoDataWalk[int(geo.Unix())] = geo

	return nil
}

func sexToDec(deg, min, sec *big.Rat, dir string) *big.Rat {
	// sexagesimal (base 60) to decimal
	// https://imm.dtf.wa.gov.au/helpfiles/Latitude_Longitude_conversion_hlp.htm

	deg.Add(deg, min.Quo(min, big.NewRat(60, 1)))
	deg.Add(deg, sec.Quo(sec, big.NewRat(3600, 1)))

	// N and E are the positive directions (like on an x,y axis)
	if dir == "S" || dir == "W" {
		deg.Neg(deg)
	}

	return deg
}
