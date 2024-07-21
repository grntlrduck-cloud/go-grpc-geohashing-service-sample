package poi

import (
	"fmt"

	"github.com/google/uuid"

	poiv1 "github.com/grntlduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1"
)

type PoILocation struct {
	Id               uuid.UUID
	Location         Coordiantes
	Address          Address
	LocationEntrance Coordiantes
	Features         []string
}

type Coordiantes struct {
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
	poi     *poiv1.PoI
}

func (e PoIParseError) Error() string {
	return fmt.Sprintf("%s for poi=%s", e.message, e.poi.Id)
}

func Parse(poipb *poiv1.PoI) (PoILocation, error) {
	id, err := uuid.Parse(poipb.Id)
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
		Location: Coordiantes{
			Longitude: poipb.Coordinate.Lon,
			Latitude:  poipb.Coordinate.Lat,
		},
		LocationEntrance: Coordiantes{
			Longitude: poipb.Entrance.Lon,
			Latitude:  poipb.Entrance.Lat,
		},
		Features: poipb.Features,
	}
	return poil, nil
}
