//go:build darwin

package macui

import (
	_ "embed"

	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/progrium/darwinkit/objc"
)

//go:embed assets/menubar-icon.png
var menubarIconPNG []byte

const menubarIconSize = 18.0

func loadMenubarIcon() appkit.Image {
	img := appkit.ImageClass.Alloc().InitWithData(menubarIconPNG)
	img.SetSize(foundation.Size{Width: menubarIconSize, Height: menubarIconSize})
	return img
}

// tintedMenubarIcon returns a copy of the base icon tinted with the given NSColor.
func tintedMenubarIcon(color appkit.Color) appkit.Image {
	size := foundation.Size{Width: menubarIconSize, Height: menubarIconSize}
	base := loadMenubarIcon()

	tinted := appkit.ImageClass.Alloc().InitWithSize(size)
	objc.Call[objc.Void](tinted, objc.Sel("lockFocus"))

	rect := foundation.Rect{
		Origin: foundation.Point{X: 0, Y: 0},
		Size:   size,
	}

	base.DrawInRectFromRectOperationFraction(rect, rect, appkit.CompositeCopy, 1.0)

	ctx := appkit.GraphicsContext_CurrentContext()
	ctx.SetCompositingOperation(appkit.CompositeDestinationIn)
	color.SetFill()
	appkit.BezierPath_FillRect(rect)

	objc.Call[objc.Void](tinted, objc.Sel("unlockFocus"))
	return tinted
}
