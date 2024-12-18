package dynamo

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/amazon-ion/ion-go/ion"
	"github.com/segmentio/ksuid"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/domain/poi"
)

const (
	CPoIItemPK           = "pk"
	CPoIItemGeoIndexName = "gsi1_geo"
	CPoIItemGeoIndexPK   = "gsi1_geo_pk"
	CPoIItemGeoIndexSK   = "gsi1_geo_sk"
	CPoIItemCellLevel    = 9 //  edge length of min 27 km and max 38 km http://s2geometry.io/resources/s2cell_statistics.html
	countryCodeDeu       = "DEU"
	ac                   = "AC"
	dc                   = "DC"
)

// The CPoIItem is a flattened representation of the domain with a primary key (hashkey) to get a cPoI by its id
// and a global secondary geo index where the primary key (hashkey) is the trimmed geohash and the sortkey is the full precision geohash.
// The structure is flattened so that a import of the dataset from csv on table creation through IaC is easier and less errorprone.
type CPoIItem struct {
	Pk                string   `json:"pk"             csv:"pk"            dynamodbav:"pk"`
	GeoIndexPk        uint64   `json:"gsi1_geo_pk_pk" csv:"gsi1_geo_pk"   dynamodbav:"gsi1_geo_pk"` // the geohash with trimmed precision
	GeoIndexSk        uint64   `json:"gsi1_geo_pk"    csv:"gsi1_geo_sk"   dynamodbav:"gsi1_geo_sk"` // the geohash with full precision
	ID                string   `json:"id"             csv:"id"            dynamodbav:"id"`
	Street            string   `json:"street"         csv:"street"        dynamodbav:"street"`
	StreetNumber      string   `json:"street_number"  csv:"street_number" dynamodbav:"street_number"`
	ZipCode           string   `json:"zip_code"       csv:"zip_code"      dynamodbav:"zip_code"`
	City              string   `json:"city"           csv:"city"          dynamodbav:"city"`
	CountryCode       string   `json:"country_code"   csv:"country_code"  dynamodbav:"country_code"`
	Features          []string `json:"features"       csv:"features"      dynamodbav:"features"`
	Longitude         float64  `json:"lon"            csv:"lon"           dynamodbav:"lon"`
	Latitude          float64  `json:"lat"            csv:"lat"           dynamodbav:"lat"`
	EntranceLongitude float64  `json:"entrance_lon"   csv:"entrance_lon"  dynamodbav:"entrance_lon"`
	EntranceLatitude  float64  `json:"entrance_lat"   csv:"entrance_lat"  dynamodbav:"entrance_lat"`
}

func (cp *CPoIItem) Domain() (*poi.PoILocation, error) {
	id, err := ksuid.Parse(cp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to pares Pk of item to ksuid: %w", err)
	}
	return &poi.PoILocation{
		ID: id,
		Location: poi.Coordinates{
			Latitude:  cp.Latitude,
			Longitude: cp.Longitude,
		},
		Address: poi.Address{
			Street:       cp.Street,
			StreetNumber: cp.StreetNumber,
			ZipCode:      cp.ZipCode,
			City:         cp.City,
			CountryCode:  cp.CountryCode,
		},
		LocationEntrance: poi.Coordinates{
			Latitude:  cp.EntranceLatitude,
			Longitude: cp.EntranceLongitude,
		},
		Features: cp.Features,
	}, nil
}

func NewItemFromDomain(poiL *poi.PoILocation) (*CPoIItem, error) {
	gh, err := newGeoHash(poiL.Location.Latitude, poiL.Location.Longitude)
	if err != nil {
		return nil, fmt.Errorf("failed to create geo hash: %w", err)
	}
	id := poiL.ID.String()
	return &CPoIItem{
		Pk: id,
		GeoIndexPk: gh.trimmed(
			CPoIItemCellLevel,
		), // the trimmed geo hash adjusted to the level
		GeoIndexSk:        gh.hash(), // the full length geo hash
		ID:                id,
		Street:            poiL.Address.Street,
		StreetNumber:      poiL.Address.StreetNumber,
		ZipCode:           poiL.Address.ZipCode,
		City:              poiL.Address.City,
		CountryCode:       poiL.Address.CountryCode,
		Longitude:         poiL.Location.Longitude,
		Latitude:          poiL.Location.Latitude,
		EntranceLongitude: poiL.LocationEntrance.Longitude,
		EntranceLatitude:  poiL.LocationEntrance.Latitude,
		Features:          poiL.Features,
	}, nil
}

func (cp *CPoIItem) IonItem() *IonItem {
	return &IonItem{
		CPoIIonItem{
			Pk:                cp.Pk,
			GeoIndexPk:        *ion.NewDecimalInt(int64(cp.GeoIndexPk)), //nolint:gosec // no relevant risk
			GeoIndexSk:        *ion.NewDecimalInt(int64(cp.GeoIndexSk)), //nolint:gosec // no relevant risk
			ID:                cp.ID,
			Street:            cp.Street,
			StreetNumber:      cp.StreetNumber,
			ZipCode:           cp.ZipCode,
			City:              cp.City,
			CountryCode:       cp.CountryCode,
			Features:          cp.Features,
			Longitude:         floatToDecimal(cp.Longitude),
			Latitude:          floatToDecimal(cp.Latitude),
			EntranceLongitude: floatToDecimal(cp.EntranceLongitude),
			EntranceLatitude:  floatToDecimal(cp.EntranceLatitude),
		},
	}
}

func floatToDecimal(fnum float64) ion.Decimal {
	fstr := strconv.FormatFloat(fnum, 'f', -1, 64)
	n, e := ion.ParseDecimal(fstr)
	if e != nil {
		panic(fmt.Errorf("failed to convert float %s, %w", fstr, e))
	}
	return *n
}

// The struct to model the data contained in the dataset, see https://www.kaggle.com/datasets/mexwell/electric-vehicle-charging-in-germany
type ChargingCSVEntry struct {
	ChargingType         string  `csv:"art_der_ladeeinrichtung"`
	Power                float32 `csv:"anschlussleistung"`
	NumberOfChargePoints int8    `csv:"anzahl_ladepunkte"`
	PlugType1            string  `csv:"steckertypen1"`
	PlugType2            string  `csv:"steckertypen2"`
	PlugType3            string  `csv:"steckertypen3"`
	PlugType4            string  `csv:"steckertypen4"`
	City                 string  `csv:"ort"`
	ZipCode              string  `csv:"postleitzahl"`
	Street               string  `csv:"strasse"`
	StreetNumber         string  `csv:"hausnummer"`
	Longitude            float64 `csv:"laengengrad"`
	Latitude             float64 `csv:"breitengrad"`
}

func EntriesToDynamo(ctes []*ChargingCSVEntry) []*CPoIItem {
	dynamoItems := make([]*CPoIItem, len(ctes))
	c := make(chan *CPoIItem, 10)
	defer close(c)
	for _, cte := range ctes {
		go func() {
			item, err := cte.MapToDynamo()
			if err != nil {
				panic("unable to map items")
			}
			c <- item
		}()
	}
	for i := range len(ctes) {
		dynamoItems[i] = <-c
	}
	return dynamoItems
}

func (cte *ChargingCSVEntry) MapToDynamo() (*CPoIItem, error) {
	gh, err := newGeoHash(cte.Latitude, cte.Longitude)
	if err != nil {
		return nil, fmt.Errorf("failed to create geohash: %w", err)
	}
	id := ksuid.New().String()
	return &CPoIItem{
		Pk: id,
		GeoIndexPk: gh.trimmed(
			CPoIItemCellLevel,
		), // the trimmed geo hash representing a tile
		GeoIndexSk:        gh.hash(), // the full length geo hash
		ID:                id,
		Street:            cte.Street,
		StreetNumber:      cte.StreetNumber,
		ZipCode:           cte.ZipCode,
		City:              cte.City,
		CountryCode:       countryCodeDeu,
		Longitude:         cte.Longitude,
		Latitude:          cte.Latitude,
		EntranceLongitude: cte.Longitude,
		EntranceLatitude:  cte.Latitude,
		Features:          cte.features(),
	}, nil
}

func (cte *ChargingCSVEntry) features() []string {
	features := make([]string, 2)
	features[0] = fmt.Sprintf("%d_CHARGEPOINTS", cte.NumberOfChargePoints)
	features[1] = fmt.Sprintf("%d_KW_CHARGING", int32(cte.Power))
	if cte.hasAcCharging() {
		features = append(features, "AC_CHARGING")
	}
	if cte.hasDcCharging() {
		features = append(features, "DC_CHARGING")
	}
	return features
}

func (cte *ChargingCSVEntry) hasAcCharging() bool {
	return strings.Contains(cte.PlugType1, ac) || strings.Contains(cte.PlugType2, ac) ||
		strings.Contains(cte.PlugType3, ac) ||
		strings.Contains(cte.PlugType4, ac)
}

func (cte *ChargingCSVEntry) hasDcCharging() bool {
	return strings.Contains(cte.PlugType1, dc) || strings.Contains(cte.PlugType2, dc) ||
		strings.Contains(cte.PlugType3, dc) ||
		strings.Contains(cte.PlugType4, dc)
}

type IonItem struct {
	Item CPoIIonItem `ion:"Item"`
}

type CPoIIonItem struct {
	Pk                string      `ion:"pk"`
	GeoIndexPk        ion.Decimal `ion:"gsi1_geo_pk"`
	GeoIndexSk        ion.Decimal `ion:"gsi1_geo_sk"`
	ID                string      `ion:"id"`
	Street            string      `ion:"street"`
	StreetNumber      string      `ion:"street_number"`
	ZipCode           string      `ion:"zip_code"`
	City              string      `ion:"city"`
	CountryCode       string      `ion:"country_code"`
	Features          []string    `ion:"features"`
	Longitude         ion.Decimal `ion:"lon"`
	Latitude          ion.Decimal `ion:"lat"`
	EntranceLongitude ion.Decimal `ion:"entrance_lon"`
	EntranceLatitude  ion.Decimal `ion:"entrance_lat"`
}
