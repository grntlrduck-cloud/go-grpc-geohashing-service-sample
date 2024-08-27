package poi

import (
	"fmt"

	"github.com/segmentio/ksuid"

	poi_v1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/poi"
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

type PoIParseError struct {
	message string
	poi     *poi_v1.PoI
}

func (e PoIParseError) Error() string {
	return fmt.Sprintf("%s for poi=%s", e.message, e.poi.Id)
}

func Parse(poipb *poi_v1.PoI) (PoILocation, error) {
	id, err := ksuid.Parse(poipb.Id)
	if err != nil {
		return PoILocation{}, PoIParseError{message: err.Error(), poi: poipb}
	}
	poil := PoILocation{
		Id: id,
		Address: Address{
			Street:       poipb.Address.Street,
			StreetNumber: poipb.Address.StreetNumber,
			ZipCode:      poipb.Address.ZipCode,
			City:         poipb.Address.City,
		},
		Location: Coordinates{
			Longitude: poipb.Coordinate.Lon,
			Latitude:  poipb.Coordinate.Lat,
		},
		LocationEntrance: Coordinates{
			Longitude: poipb.Entrance.Lon,
			Latitude:  poipb.Entrance.Lat,
		},
		Features: poipb.Features,
	}
	return poil, nil
}
