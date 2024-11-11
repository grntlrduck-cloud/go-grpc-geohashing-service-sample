package poi

import (
	"github.com/segmentio/ksuid"
)

type PoILocation struct {
	Id               ksuid.KSUID
	Location         Coordinates
	Address          Address
	LocationEntrance Coordinates
	Features         []string
}

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Address struct {
	Street       string
	StreetNumber string
	ZipCode      string
	City         string
	CountryCode  string
}
