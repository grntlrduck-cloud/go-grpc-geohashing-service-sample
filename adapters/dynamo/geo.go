package dynamo

import (
	"strconv"

	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/domain/poi"
)

const earthRadiusMeter = 6371000.0

// The GeoHash wraps and hides the actual geohashing complexity
type geoHash struct {
	hashId s2.CellID
}

func (h geoHash) hash() uint64 {
	return uint64(h.hashId)
}

func (h geoHash) trimmed(length uint8) uint64 {
	if length < 1 || length > 12 {
		return uint64(h.hashId)
	}
	stringHash := strconv.FormatUint(uint64(h.hashId), 10)
	v, _ := strconv.ParseInt(stringHash[:length], 10, 64)
	return uint64(v)
}

func (h geoHash) min() uint64 {
	return uint64(h.hashId.RangeMin())
}

func (h geoHash) max() uint64 {
	return uint64(h.hashId.RangeMax())
}

func newGeoHash(lat, lon float64) geoHash {
	latLonAngles := s2.LatLngFromDegrees(lat, lon)
	cell := s2.CellFromLatLng(latLonAngles)
	return geoHash{hashId: cell.ID()}
}

func newHashesFromRadiusCenter(c poi.Coordinates, radius float64) []geoHash {
	angle := s1.Angle(radius / earthRadiusMeter)
	centerPoint := pointFromCords(c)
	region := s2.CapFromCenterAngle(centerPoint, angle)
	// in the case of a radius search we want to return more results than in the radius intentiaionally
	// so that if a user zooms there are still enough PoI centered
	// http://s2geometry.io/resources/s2cell_statistics.html
	coverer := s2.RegionCoverer{
		MinLevel: 8,  // 27 km * 38 km
		MaxLevel: 12, // 1699 m * 2 km
		MaxCells: 10,
		LevelMod: 1,
	}
	covering := coverer.Covering(region)
	return newGeoHashesFromCells(covering)
}

func newHashesFromBbox(ne, sw poi.Coordinates) []geoHash {
	bounder := s2.NewRectBounder()
	bounder.AddPoint(pointFromCords(ne))
	bounder.AddPoint(pointFromCords(sw))
	// for bboxes we also want to 'over search'
	// http://s2geometry.io/resources/s2cell_statistics.html
	coverer := s2.RegionCoverer{
		MinLevel: 8,  // 27 km * 39 km
		MaxLevel: 12, // 1699 km * 2 km
		MaxCells: 10,
		LevelMod: 1,
	}
	covering := coverer.Covering(bounder.RectBound())
	return newGeoHashesFromCells(covering)
}

func newHashesFromRoute(path []poi.Coordinates) []geoHash {
	var latLngs []s2.LatLng
	for _, p := range path {
		latLngs = append(latLngs, s2.LatLngFromDegrees(p.Latitude, p.Longitude))
	}
	polyline := s2.PolylineFromLatLngs(latLngs)
	// to construct a smooth and tight coverage of cells along a plolyline
	// we set the levels to get fine grained compartments for the covering
	// http://s2geometry.io/resources/s2cell_statistics.html
	coverer := s2.RegionCoverer{
		MinLevel: 9,  // 14 km * 19 km
		MaxLevel: 16, // 108 m 148 m
		MaxCells: 30,
		LevelMod: 1,
	}
	covering := coverer.Covering(polyline)
	return newGeoHashesFromCells(covering)
}

func newGeoHashesFromCells(cells []s2.CellID) []geoHash {
	var hashes []geoHash
	for _, v := range cells {
		hashes = append(hashes, geoHash{hashId: v})
	}
	return hashes
}

func pointFromCords(c poi.Coordinates) s2.Point {
	return s2.PointFromLatLng(s2.LatLngFromDegrees(c.Latitude, c.Longitude))
}
