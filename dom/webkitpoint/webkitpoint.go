package webkitpoint

import "github.com/matthewmueller/joy/js"

// New fn
func New(x *float32, y *float32) *WebKitPoint {
	js.Rewrite("WebKitPoint")
	return &WebKitPoint{}
}

// WebKitPoint struct
// js:"WebKitPoint,omit"
type WebKitPoint struct {
}

// X prop
// js:"x"
func (*WebKitPoint) X() (x float32) {
	js.Rewrite("$_.x")
	return x
}

// SetX prop
// js:"x"
func (*WebKitPoint) SetX(x float32) {
	js.Rewrite("$_.x = $1", x)
}

// Y prop
// js:"y"
func (*WebKitPoint) Y() (y float32) {
	js.Rewrite("$_.y")
	return y
}

// SetY prop
// js:"y"
func (*WebKitPoint) SetY(y float32) {
	js.Rewrite("$_.y = $1", y)
}