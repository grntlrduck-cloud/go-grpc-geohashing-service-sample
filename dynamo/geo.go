package dynamo

import (
	"strconv"

	"github.com/golang/geo/s2"
)

// The GeoHash wraps and hides the actual geohashing complexity
type GeoHash struct {
	hashId s2.CellID
}

func (h GeoHash) Hash() uint64 {
	return uint64(h.hashId)
}

func (h GeoHash) Trimmed(length uint8) uint64 {
	if length < 1 || length > 12 {
		return uint64(h.hashId)
	}
	stringHash := strconv.FormatUint(uint64(h.hashId), 10)
	v, _ := strconv.ParseInt(stringHash[:length], 10, 64)
	return uint64(v)
}

func NewGeoHash(lat, lon float64) GeoHash {
	latLonAngles := s2.LatLngFromDegrees(lat, lon)
	cell := s2.CellFromLatLng(latLonAngles)
	return GeoHash{hashId: cell.ID()}
}
