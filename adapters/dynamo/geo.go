package dynamo

import (
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

func (h geoHash) trimmed(level int) uint64 {
	if level < 0 || level > 30 {
		return uint64(h.hashId)
	}
	parent := s2.CellIDFromFacePosLevel(h.hashId.Face(), h.hashId.Pos(), level)
	return uint64(parent)
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
		MinLevel: 9,
		MaxLevel: 13,
		MaxCells: 15,
		LevelMod: 1,
	}
	covering := coverer.Covering(region)
	return newGeoHashesFromCells(covering)
}

func newHashesFromBbox(ne, sw poi.Coordinates) []geoHash {
	bounder := s2.NewRectBounder()
	bounder.AddPoint(pointFromCords(ne))
	bounder.AddPoint(pointFromCords(sw))
	coverer := s2.RegionCoverer{
		MinLevel: 9,
		MaxLevel: 13,
		MaxCells: 15,
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
	coverer := s2.RegionCoverer{
		MinLevel: 9,
		MaxLevel: 16,
		MaxCells: 100,
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
