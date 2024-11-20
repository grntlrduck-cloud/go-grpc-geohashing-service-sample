package rpc

func headerMatcher(key string) (string, bool) {
	switch key {
	case apiKeyHeader:
		return key, true
	case correlationHeader:
		return key, true
	default:
		return key, false
	}
}
