package dynamo

import (
	"fmt"

	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/domain/poi"
)

const (
	earthRadiusMeter = 6371000.0
	maxLatitude      = 90.0
	minLatitude      = -90.0
	maxLongitude     = 180.0
	minLongitude     = -180.0
)

// in the case of a radius search we want to return more results than in the radius intentiaionally
// so that if a user zooms there are still enough PoI centered
// http://s2geometry.io/resources/s2cell_statistics.html
// BBox and Radius search default
var defaultAreaCoverer = s2.RegionCoverer{
	MinLevel: 9, // more coarse
	MaxLevel: 13,
	MaxCells: 15,
	LevelMod: 1,
}

// default for Route search
var defaultPolylineCoverer = s2.RegionCoverer{
	MinLevel: 9,
	MaxLevel: 15,  // fine grainer
	MaxCells: 100, // needs to cover longer area
	LevelMod: 1,
}

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

func newGeoHash(lat, lon float64) (*geoHash, error) {
	if lat < minLatitude || lat > maxLatitude || lon < minLongitude || lon > maxLongitude {
		return nil, fmt.Errorf("invalid coordinates: lat=%f, lon=%f", lat, lon)
	}
	latLonAngles := s2.LatLngFromDegrees(lat, lon)
	cell := s2.CellFromLatLng(latLonAngles)
	return &geoHash{hashId: cell.ID()}, nil
}

func newHashesFromRadiusCenter(
	c poi.Coordinates,
	radius float64,
	coverer *s2.RegionCoverer,
) ([]geoHash, error) {
	if !isValidLatLon(c.Latitude, c.Longitude) {
		return nil, fmt.Errorf("invalid search center: lat=%f, lon=%f", c.Latitude, c.Longitude)
	}
	angle := s1.Angle(radius / earthRadiusMeter)
	centerPoint := pointFromCords(c)
	region := s2.CapFromCenterAngle(centerPoint, angle)
	if coverer == nil {
		coverer = &defaultAreaCoverer
	}
	covering := coverer.Covering(region)
	return newGeoHashesFromCells(covering), nil
}

func newHashesFromBbox(ne, sw poi.Coordinates, coverer *s2.RegionCoverer) ([]geoHash, error) {
	if !isValidLatLon(ne.Latitude, ne.Longitude) || !isValidLatLon(sw.Latitude, sw.Longitude) {
		return nil, fmt.Errorf(
			"invalid bounding box: ne.lat=%f, ne.lon=%f, sw.lat=%f, sw.lon=%f",
			ne.Latitude,
			ne.Longitude,
			sw.Latitude,
			sw.Longitude,
		)
	}
	bounder := s2.NewRectBounder()
	bounder.AddPoint(pointFromCords(ne))
	bounder.AddPoint(pointFromCords(sw))
	if coverer == nil {
		coverer = &defaultAreaCoverer
	}
	covering := coverer.Covering(bounder.RectBound())
	return newGeoHashesFromCells(covering), nil
}

func newHashesFromRoute(path []poi.Coordinates, coverer *s2.RegionCoverer) ([]geoHash, error) {
	if len(path) < 2 {
		return nil, fmt.Errorf("invalid path: length=%d", len(path))
	}

	// Validate coordinates before processing
	for _, p := range path {
		if !isValidLatLon(p.Latitude, p.Longitude) {
			return nil, fmt.Errorf(
				"invalid coordinates for route: lat=%f, lon:=%f",
				p.Latitude,
				p.Longitude,
			)
		}
	}

	// Pre-allocate slice with exact capacity
	latLngs := make([]s2.LatLng, len(path))
	for i, p := range path {
		latLngs[i] = s2.LatLngFromDegrees(p.Latitude, p.Longitude)
	}

	polyline := s2.PolylineFromLatLngs(latLngs)
	if coverer == nil {
		coverer = &defaultPolylineCoverer
	}
	covering := coverer.Covering(polyline)
	return newGeoHashesFromCells(covering), nil
}

func newGeoHashesFromCells(cells []s2.CellID) []geoHash {
	hashes := make([]geoHash, len(cells))
	for i, v := range cells {
		hashes[i] = geoHash{hashId: v}
	}
	return hashes
}

func pointFromCords(c poi.Coordinates) s2.Point {
	return s2.PointFromLatLng(s2.LatLngFromDegrees(c.Latitude, c.Longitude))
}

func isValidLatLon(lat, lon float64) bool {
	return lat >= minLatitude || lat <= maxLatitude ||
		lon >= minLongitude || lon <= maxLongitude
}
