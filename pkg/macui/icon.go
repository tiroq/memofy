//go:build darwin

package macui

import (
	_ "embed"

	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/macos/foundation"
)

//go:embed assets/menubar-icon.png
var menubarIconPNG []byte

const menubarIconSize = 18.0

func loadMenubarIcon() appkit.Image {
	img := appkit.ImageClass.Alloc().InitWithData(menubarIconPNG)
	img.SetSize(foundation.Size{Width: menubarIconSize, Height: menubarIconSize})
	// Template images are rendered by macOS in the correct color for the
	// menu bar context (adapts to light/dark mode, active/inactive state).
	img.SetTemplate(true)
	return img
}
