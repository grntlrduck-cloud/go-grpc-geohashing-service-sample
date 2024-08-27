package poi

type Repository interface {
  UpsertBatch(pois []PoILocation)
  Upsert(poi PoILocation) (PoILocation, error)
  GetById(id string) (PoILocation, error)
  GetByProximity(cntr Coordinates, radius float64) ([]PoILocation, error)
  GetByBbox(sw, ne Coordinates) ([]PoILocation, error)
  BetByRoute(path []Coordinates, radius float64) ([]PoILocation, error)
}
