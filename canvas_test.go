package shaw

import "testing"

func TestNewCanvasForcesEvenHeight(t *testing.T) {
	c := NewCanvas(4, 3)
	if c.Width() != 4 {
		t.Errorf("width = %d, want 4", c.Width())
	}
	if c.Height() != 4 {
		t.Errorf("height = %d, want 4 (rounded up to even)", c.Height())
	}
}

func TestSetSkipsTransparentAndOutOfBounds(t *testing.T) {
	c := NewCanvas(2, 2)
	red := Color{R: 255, A: 255}
	c.Set(0, 0, red)
	c.Set(0, 1, Color{R: 9, G: 9, B: 9}) // A==0 -> ignored
	c.Set(5, 5, red)                      // out of bounds -> ignored
	if got := c.at(0, 0); got != red {
		t.Errorf("at(0,0) = %+v, want %+v", got, red)
	}
	if got := c.at(0, 1); got != (Color{}) {
		t.Errorf("at(0,1) = %+v, want zero (transparent set ignored)", got)
	}
}

func TestClearFillsEveryPixel(t *testing.T) {
	c := NewCanvas(2, 2)
	blue := Color{B: 255, A: 255}
	c.Clear(blue)
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			if got := c.at(x, y); got != blue {
				t.Errorf("at(%d,%d) = %+v, want %+v", x, y, got, blue)
			}
		}
	}
}
