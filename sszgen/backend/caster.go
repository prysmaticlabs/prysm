package backend

type caster interface {
	setToOverlay(func(string) string)
	setFromOverlay(func(string) string)
}

type casterConfig struct {
	toOverlayFunc func(string) string
	fromOverlayFunc func(string) string
}

func (c *casterConfig) setToOverlay(castFunc func(string) string) {
	c.toOverlayFunc = castFunc
}

func (c *casterConfig) toOverlay(value string) string {
	if c.toOverlayFunc == nil {
		return value
	}
	return c.toOverlayFunc(value)
}

func (c *casterConfig) setFromOverlay(castFunc func(string) string) {
	c.fromOverlayFunc = castFunc
}

func (c *casterConfig) fromOverlay(value string) string {
	if c.fromOverlayFunc == nil {
		return value
	}
	return c.fromOverlayFunc(value)
}
