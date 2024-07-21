package dynamo

const (
	CPoIItemPK               = "pk"
	CPoIItemGeoIndexName     = "gsi1_geo"
	CPoIItemGeoIndexPK       = "gsi1_geo_pk"
	CPoIItemGeoIndexSK       = "gsi1_geo_sk"
	CPoIItemGeoHashKeyLength = 4
)

// The CPoIItem is a flattened representation of the domain with a primary key (hashkey) to get a cPoI by its id
// and a global secondary geo index where the primary key (hashkey) is the trimmed geohash and the sortkey is the full precision geohash.
// The structure is flattend so that a import of the dataset from csv on table creation through IaC is easier and less errorprone.
type CPoIItem struct {
	Pk                string   `json:"pk"             csv:"pk"            dynamodbav:"pk"`
	GeoIndexPk        uint64   `json:"gsi1_geo_pk_pk" csv:"gsi1_geo_pk"   dynamodbav:"gsi1_geo_pk"` // the geohash with trimmed precision
	GeoIndexSk        uint64   `json:"gsi1_geo_pk"    csv:"gsi1_geo_sk"   dynamodbav:"gsi1_geo_sk"` // the geohash with full precision
	Id                string   `json:"id"             csv:"id"            dynamodbav:"id"`
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
