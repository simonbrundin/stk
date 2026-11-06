package counter

type Counter struct {
	value int
}

func New() *Counter {
	return &Counter{value: 0}
}

func (c *Counter) Inc() { c.value++ }
func (c *Counter) Dec() { c.value-- }
func (c *Counter) Get() int {
	return c.value
}
func (c *Counter) Reset() { c.value = 0 }
