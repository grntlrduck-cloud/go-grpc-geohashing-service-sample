package dynamo

const (
	CPoIItemPK           = "pk"
	CPoIItemGeoIndexName = "gsi1_geo"
	CPoIItemGeoIndexPK   = "gsi1_geo_pk"
	CPoIItemGeoIndexSK   = "gsi1_geo_sk"
)

type CPoIItem struct {
	Pk           string   `json:"pk"             dynamodbav:"pk"`
	GeoIndexPk   uint64   `json:"gsi1_geo_pk_pk" dynamodbav:"gsi1_geo_pk"` // the geohash with trimmed precision
	GeoIndexSk   uint64   `json:"gsi1_geo_pk"    dynamodbav:"gsi1_geo_sk"` // the geohash with full precision
	Id           string   `json:"id"             dynamodbav:"id"`
	WgsPoint        WgsPoint `json:"wgs_point"      dynamodbav:"wgs_point"`
	Street       string   `json:"street"         dynamodbav:"street"`
	StreetNumber string   `json:"street_number"  dynamodbav:"street_number"`
	ZipCode      string   `json:"zip_code"       dynamodbav:"zip_code"`
	City         string   `json:"city"           dynamodbav:"city"`
	CountryCode  string   `json:"country_code"   dynamodbav:"country_code"`
	Features     []string `json:"features"       dynamodbav:"features"`
}

type WgsPoint struct {
	Longitude float64 `json:"lon" dynamodbav:"lon"`
	Latitude  float64 `json:"lat" dynamodbav:"lat"`
}
