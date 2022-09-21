package math

func DisableCaches() {
	sqrtCache.Resize(0)
}

func EnableCaches() {
	sqrtCache.Resize(SQRT_CACHE_SIZE)
}
