package shaw

// Canvas is a width x height grid of pixels. Height is always even: each
// terminal cell row holds two vertical pixels (see Render).
type Canvas struct {
	w, h   int
	pixels []Color
}

// NewCanvas allocates a w x h canvas. An odd h is rounded up to the next even
// number so it maps cleanly onto half-block cell rows.
func NewCanvas(w, h int) *Canvas {
	if h%2 != 0 {
		h++
	}
	return &Canvas{w: w, h: h, pixels: make([]Color, w*h)}
}

// Width returns the canvas width in pixels.
func (c *Canvas) Width() int { return c.w }

// Height returns the canvas height in pixels (even).
func (c *Canvas) Height() int { return c.h }

// Set paints one pixel. Transparent colors (A == 0) and out-of-bounds
// coordinates are ignored, so callers can blit freely without clipping.
func (c *Canvas) Set(x, y int, col Color) {
	if col.A == 0 || x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	c.pixels[y*c.w+x] = col
}

// Clear fills every pixel with col, bypassing the transparency skip in Set.
func (c *Canvas) Clear(col Color) {
	for i := range c.pixels {
		c.pixels[i] = col
	}
}

// at returns the stored color at (x,y). Test helper; assumes in bounds.
func (c *Canvas) at(x, y int) Color { return c.pixels[y*c.w+x] }
