package dynamo

import (
	"math"
	"strconv"

	"github.com/golang/geo/s2"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/domain/poi"
)

const earthCircumferenceMeters = 40075000.0

// The GeoHash wraps and hides the actual geohashing complexity
type GeoHash struct {
	hashId s2.CellID
}

func (h GeoHash) hash() uint64 {
	return uint64(h.hashId)
}

func (h GeoHash) trimmed(length uint8) uint64 {
	if length < 1 || length > 12 {
		return uint64(h.hashId)
	}
	stringHash := strconv.FormatUint(uint64(h.hashId), 10)
	v, _ := strconv.ParseInt(stringHash[:length], 10, 64)
	return uint64(v)
}

func (h GeoHash) min() uint64 {
	return uint64(h.hashId.RangeMin())
}

func (h GeoHash) max() uint64 {
	return uint64(h.hashId.RangeMax())
}

func newGeoHash(lat, lon float64) GeoHash {
	latLonAngles := s2.LatLngFromDegrees(lat, lon)
	cell := s2.CellFromLatLng(latLonAngles)
	return GeoHash{hashId: cell.ID()}
}

func newHashesFromRadiusCenter(c poi.Coordinates, radius float64) []GeoHash {
	searchRadiusRadians := (2 * math.Pi) * (radius / earthCircumferenceMeters)
	centerPoint := pointFromCords(c)
	region := s2.CapFromCenterHeight(centerPoint, searchRadiusRadians)
	coverer := s2.NewRegionCoverer()
	covering := coverer.Covering(region)
	return newGeoHashesFromCells(covering)
}

func newHashesFromBbox(ne, sw poi.Coordinates) []GeoHash {
	bounder := s2.NewRectBounder()
	bounder.AddPoint(pointFromCords(ne))
	bounder.AddPoint(pointFromCords(sw))
	coverer := s2.NewRegionCoverer()
	covering := coverer.Covering(bounder.RectBound())
	return newGeoHashesFromCells(covering)
}

func newHashesFromRoute(path []poi.Coordinates) []GeoHash {
	var latLngs []s2.LatLng
	for _, p := range path {
		latLngs = append(latLngs, s2.LatLngFromDegrees(p.Latitude, p.Longitude))
	}
	polyline := s2.PolylineFromLatLngs(latLngs)
	coverer := s2.NewRegionCoverer()
	covering := coverer.Covering(polyline.RectBound())
	return newGeoHashesFromCells(covering)
}

func newGeoHashesFromCells(cells []s2.CellID) []GeoHash {
	var hashes []GeoHash
	for _, v := range cells {
		hashes = append(hashes, GeoHash{hashId: v})
	}
	return hashes
}

func pointFromCords(c poi.Coordinates) s2.Point {
	return s2.PointFromLatLng(s2.LatLngFromDegrees(c.Latitude, c.Longitude))
}
