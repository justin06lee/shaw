package shaw

// Color is an RGBA color. A == 0 means fully transparent; the canvas renders
// transparent pixels as black, and Set ignores them so sprites can carry holes.
type Color struct {
	R, G, B, A uint8
}
