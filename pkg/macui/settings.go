//go:build darwin

package macui

import (
	"log"

	"github.com/progrium/darwinkit/helper/action"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/progrium/darwinkit/objc"
	"github.com/tiroq/memofy/internal/config"
)

// SettingsWindow manages the native macOS settings window.
type SettingsWindow struct {
	cfg       config.Config
	window    appkit.Window
	isVisible bool

	// Audio tab controls
	device         appkit.TextField
	threshold      appkit.TextField
	activationMs   appkit.TextField
	silenceSeconds appkit.TextField
	formatProfile  appkit.TextField

	// Output tab controls
	outputDir appkit.TextField

	// Monitoring tab controls
	detectZoom     appkit.Button
	detectTeams    appkit.Button
	detectMicUsage appkit.Button
	keepSingleMic  appkit.Button

	// General tab controls
	autoCheckUpdates appkit.Button
	logLevel         appkit.TextField
}

// NewSettingsWindow creates a SettingsWindow with the given config.
func NewSettingsWindow(cfg config.Config) *SettingsWindow {
	return &SettingsWindow{cfg: cfg}
}

// Show builds (or focuses) the native settings window.
// Must be called on the main thread.
func (sw *SettingsWindow) Show() error {
	if sw.isVisible && sw.window.Ptr() != nil {
		// Reload config and refill fields
		if cfg, err := config.LoadConfig(""); err == nil {
			sw.cfg = *cfg
		}
		sw.reloadFields()
		sw.window.MakeKeyAndOrderFront(nil)
		appkit.Application_SharedApplication().ActivateIgnoringOtherApps(true)
		return nil
	}

	if cfg, err := config.LoadConfig(""); err == nil {
		sw.cfg = *cfg
	}
	sw.buildWindow()
	sw.window.MakeKeyAndOrderFront(nil)
	appkit.Application_SharedApplication().ActivateIgnoringOtherApps(true)
	sw.isVisible = true
	return nil
}

func (sw *SettingsWindow) buildWindow() {
	const (
		winW   = 500.0
		winH   = 500.0
		margin = 12.0
		btnH   = 28.0
		btnW   = 88.0
		btnGap = 8.0
	)

	win := appkit.NewWindowWithSize(winW, winH)
	win.SetTitle("Memofy Settings")
	win.Center()
	sw.window = win

	cv := win.ContentView()

	tabView := appkit.NewTabView()
	tabView.SetTranslatesAutoresizingMaskIntoConstraints(false)
	cv.AddSubview(tabView)

	cancelBtn := appkit.NewButtonWithTitle("Cancel")
	cancelBtn.SetTranslatesAutoresizingMaskIntoConstraints(false)
	action.Set(cancelBtn, func(_ objc.Object) { sw.onCancel() })
	cv.AddSubview(cancelBtn)

	saveBtn := appkit.NewButtonWithTitle("Save")
	saveBtn.SetTranslatesAutoresizingMaskIntoConstraints(false)
	saveBtn.SetKeyEquivalent("\r")
	action.Set(saveBtn, func(_ objc.Object) { sw.onSave() })
	cv.AddSubview(saveBtn)

	resetBtn := appkit.NewButtonWithTitle("Reset to Defaults")
	resetBtn.SetTranslatesAutoresizingMaskIntoConstraints(false)
	action.Set(resetBtn, func(_ objc.Object) { sw.onReset() })
	cv.AddSubview(resetBtn)

	// Layout: save bottom-right, cancel left of save, reset far left
	saveBtn.WidthAnchor().ConstraintEqualToConstant(btnW).SetActive(true)
	saveBtn.HeightAnchor().ConstraintEqualToConstant(btnH).SetActive(true)
	saveBtn.TrailingAnchor().ConstraintEqualToAnchorConstant(cv.TrailingAnchor(), -margin).SetActive(true)
	saveBtn.BottomAnchor().ConstraintEqualToAnchorConstant(cv.BottomAnchor(), -margin).SetActive(true)

	cancelBtn.WidthAnchor().ConstraintEqualToConstant(btnW).SetActive(true)
	cancelBtn.HeightAnchor().ConstraintEqualToConstant(btnH).SetActive(true)
	cancelBtn.TrailingAnchor().ConstraintEqualToAnchorConstant(saveBtn.LeadingAnchor(), -btnGap).SetActive(true)
	cancelBtn.BottomAnchor().ConstraintEqualToAnchorConstant(cv.BottomAnchor(), -margin).SetActive(true)

	resetBtn.HeightAnchor().ConstraintEqualToConstant(btnH).SetActive(true)
	resetBtn.LeadingAnchor().ConstraintEqualToAnchorConstant(cv.LeadingAnchor(), margin).SetActive(true)
	resetBtn.BottomAnchor().ConstraintEqualToAnchorConstant(cv.BottomAnchor(), -margin).SetActive(true)

	tabView.LeadingAnchor().ConstraintEqualToAnchor(cv.LeadingAnchor()).SetActive(true)
	tabView.TrailingAnchor().ConstraintEqualToAnchor(cv.TrailingAnchor()).SetActive(true)
	tabView.TopAnchor().ConstraintEqualToAnchor(cv.TopAnchor()).SetActive(true)
	tabView.BottomAnchor().ConstraintEqualToAnchorConstant(saveBtn.TopAnchor(), -margin).SetActive(true)

	tabView.AddTabViewItem(sw.makeTabItem("Audio", sw.buildAudioTab))
	tabView.AddTabViewItem(sw.makeTabItem("Monitoring", sw.buildMonitoringTab))
	tabView.AddTabViewItem(sw.makeTabItem("General", sw.buildGeneralTab))

	sw.reloadFields()
}

func (sw *SettingsWindow) makeTabItem(label string, builder func() appkit.IView) appkit.TabViewItem {
	item := appkit.NewTabViewItem()
	item.SetLabel(label)
	item.SetView(builder())
	return item
}

func (sw *SettingsWindow) buildAudioTab() appkit.IView {
	scroll, root := makeScrollStack()

	root.AddArrangedSubview(makeBoldLabel("Audio Input"))
	root.AddArrangedSubview(makeSeparator())

	sw.device = makeEditableField("auto")
	sw.device.SetToolTip("\"auto\" to detect automatically, or a device name substring (e.g. \"BlackHole 2ch\")")
	root.AddArrangedSubview(makeLabeledRow("Device:", sw.device))
	root.AddArrangedSubview(makeHintLabel("Enter \"auto\" for automatic detection, or part of the device name."))

	sw.threshold = makeEditableField("0.02")
	sw.threshold.SetToolTip("RMS level (0-1) above which audio is considered sound. Default: 0.02")
	root.AddArrangedSubview(makeLabeledRow("Threshold:", sw.threshold))

	sw.activationMs = makeEditableField("400")
	sw.activationMs.SetToolTip("Milliseconds of continuous sound required before recording starts. Default: 400")
	root.AddArrangedSubview(makeLabeledRow("Activation (ms):", sw.activationMs))

	sw.silenceSeconds = makeEditableField("60")
	sw.silenceSeconds.SetToolTip("Seconds of silence before splitting into a new file. Default: 60")
	root.AddArrangedSubview(makeLabeledRow("Silence Split (s):", sw.silenceSeconds))

	root.AddArrangedSubview(makeBoldLabel("Recording"))
	root.AddArrangedSubview(makeSeparator())

	sw.formatProfile = makeEditableField("high")
	sw.formatProfile.SetToolTip("Format profile: high, balanced, lightweight, wav")
	root.AddArrangedSubview(makeLabeledRow("Format Profile:", sw.formatProfile))
	root.AddArrangedSubview(makeHintLabel("high = M4A/AAC 32kHz 64kbps, balanced = 24kHz 48kbps, lightweight = 16kHz 32kbps, wav = raw"))

	root.AddArrangedSubview(makeBoldLabel("Output"))
	root.AddArrangedSubview(makeSeparator())

	sw.outputDir = makeEditableField("~/Recordings/Memofy")
	sw.outputDir.SetToolTip("Directory where recordings and metadata are saved")
	root.AddArrangedSubview(makeLabeledRow("Output Folder:", sw.outputDir))

	pinStackToScroll(root, scroll)
	return scroll
}

func (sw *SettingsWindow) buildMonitoringTab() appkit.IView {
	scroll, root := makeScrollStack()

	root.AddArrangedSubview(makeBoldLabel("Process Detection"))
	root.AddArrangedSubview(makeHintLabel("These signals enrich recording metadata only. They do not trigger recording."))
	root.AddArrangedSubview(makeSeparator())

	sw.detectZoom = appkit.NewCheckBox("Detect Zoom")
	root.AddArrangedSubview(sw.detectZoom)

	sw.detectTeams = appkit.NewCheckBox("Detect Microsoft Teams")
	root.AddArrangedSubview(sw.detectTeams)

	sw.detectMicUsage = appkit.NewCheckBox("Detect microphone activity")
	root.AddArrangedSubview(sw.detectMicUsage)

	sw.keepSingleMic = appkit.NewCheckBox("Keep single session while mic active")
	sw.keepSingleMic.SetToolTip("When enabled, prevents file splitting while microphone is in use")
	root.AddArrangedSubview(sw.keepSingleMic)

	pinStackToScroll(root, scroll)
	return scroll
}

func (sw *SettingsWindow) buildGeneralTab() appkit.IView {
	scroll, root := makeScrollStack()

	root.AddArrangedSubview(makeBoldLabel("Updates"))
	root.AddArrangedSubview(makeSeparator())

	sw.autoCheckUpdates = appkit.NewCheckBox("Automatically check for updates on startup")
	root.AddArrangedSubview(sw.autoCheckUpdates)

	root.AddArrangedSubview(makeBoldLabel("Logging"))
	root.AddArrangedSubview(makeSeparator())

	sw.logLevel = makeEditableField("info")
	sw.logLevel.SetToolTip("Log level: debug, info, warn, error")
	root.AddArrangedSubview(makeLabeledRow("Log Level:", sw.logLevel))

	pinStackToScroll(root, scroll)
	return scroll
}

func (sw *SettingsWindow) reloadFields() {
	fields := FieldsFromConfig(sw.cfg)

	sw.device.SetStringValue(fields.Device)
	sw.threshold.SetStringValue(fields.Threshold)
	sw.activationMs.SetStringValue(fields.ActivationMs)
	sw.silenceSeconds.SetStringValue(fields.SilenceSeconds)
	sw.formatProfile.SetStringValue(fields.FormatProfile)
	sw.outputDir.SetStringValue(fields.OutputDir)

	setCheckbox(sw.detectZoom, fields.DetectZoom)
	setCheckbox(sw.detectTeams, fields.DetectTeams)
	setCheckbox(sw.detectMicUsage, fields.DetectMicUsage)
	setCheckbox(sw.keepSingleMic, fields.KeepSingleSessionWhileMicActive)
	setCheckbox(sw.autoCheckUpdates, fields.AutoCheckUpdates)

	sw.logLevel.SetStringValue(fields.LogLevel)
}

func (sw *SettingsWindow) readFields() SettingsFields {
	return SettingsFields{
		Device:                          sw.device.StringValue(),
		Threshold:                       sw.threshold.StringValue(),
		ActivationMs:                    sw.activationMs.StringValue(),
		SilenceSeconds:                  sw.silenceSeconds.StringValue(),
		FormatProfile:                   sw.formatProfile.StringValue(),
		OutputDir:                       sw.outputDir.StringValue(),
		DetectZoom:                      isCheckboxOn(sw.detectZoom),
		DetectTeams:                     isCheckboxOn(sw.detectTeams),
		DetectMicUsage:                  isCheckboxOn(sw.detectMicUsage),
		KeepSingleSessionWhileMicActive: isCheckboxOn(sw.keepSingleMic),
		AutoCheckUpdates:                isCheckboxOn(sw.autoCheckUpdates),
		LogLevel:                        sw.logLevel.StringValue(),
	}
}

func (sw *SettingsWindow) onSave() {
	fields := sw.readFields()
	cfg, err := BuildConfigFromFields(fields, sw.cfg)
	if err != nil {
		log.Printf("Settings validation failed: %v", err)
		_ = SendErrorNotification("Memofy Settings", err.Error())
		return
	}

	if err := cfg.Save(config.DefaultConfigPath()); err != nil {
		log.Printf("Failed to save config: %v", err)
		_ = SendErrorNotification("Memofy Settings", "Failed to save: "+err.Error())
		return
	}

	sw.cfg = cfg
	log.Println("Settings saved")
	_ = SendNotification("Memofy", "Settings Saved", "Changes saved. Restart to apply audio changes.")
	sw.window.OrderOut(nil)
	sw.isVisible = false
}

func (sw *SettingsWindow) onCancel() {
	sw.window.OrderOut(nil)
	sw.isVisible = false
}

func (sw *SettingsWindow) onReset() {
	sw.cfg = config.Default()
	sw.cfg.Output.Dir = config.ResolvePath(sw.cfg.Output.Dir)
	sw.reloadFields()
	log.Println("Settings reset to defaults")
}

// UI helpers

func makeScrollStack() (appkit.ScrollView, appkit.StackView) {
	scroll := appkit.NewScrollView()
	scroll.SetHasVerticalScroller(true)
	scroll.SetDrawsBackground(false)

	root := appkit.NewVerticalStackView()
	root.SetTranslatesAutoresizingMaskIntoConstraints(false)
	root.SetSpacing(6)
	root.SetEdgeInsets(foundation.EdgeInsets{Top: 16, Left: 16, Bottom: 16, Right: 16})
	root.SetAlignment(appkit.LayoutAttributeLeading)
	return scroll, root
}

func pinStackToScroll(root appkit.StackView, scroll appkit.ScrollView) {
	scroll.SetDocumentView(root)
	clip := scroll.ContentView()
	root.LeadingAnchor().ConstraintEqualToAnchor(clip.LeadingAnchor()).SetActive(true)
	root.TrailingAnchor().ConstraintEqualToAnchor(clip.TrailingAnchor()).SetActive(true)
	root.TopAnchor().ConstraintEqualToAnchor(clip.TopAnchor()).SetActive(true)
}

func makeEditableField(placeholder string) appkit.TextField {
	f := appkit.NewTextField()
	f.SetBezeled(true)
	f.SetEditable(true)
	f.SetPlaceholderString(placeholder)
	f.SetTranslatesAutoresizingMaskIntoConstraints(false)
	return f
}

func makeBoldLabel(text string) appkit.TextField {
	lbl := appkit.TextField_LabelWithString(text)
	lbl.SetFont(appkit.Font_BoldSystemFontOfSize(13))
	lbl.SetTranslatesAutoresizingMaskIntoConstraints(false)
	return lbl
}

func makeHintLabel(text string) appkit.TextField {
	lbl := appkit.TextField_WrappingLabelWithString(text)
	lbl.SetTextColor(appkit.Color_SecondaryLabelColor())
	lbl.SetFont(appkit.Font_SystemFontOfSize(11))
	lbl.SetTranslatesAutoresizingMaskIntoConstraints(false)
	return lbl
}

func makeSeparator() appkit.View {
	sep := appkit.NewView()
	sep.SetTranslatesAutoresizingMaskIntoConstraints(false)
	sep.HeightAnchor().ConstraintEqualToConstant(1).SetActive(true)
	sep.SetWantsLayer(true)
	sep.Layer().SetBackgroundColor(appkit.Color_SeparatorColor().CGColor())
	return sep
}

func makeLabeledRow(labelText string, field appkit.TextField) appkit.StackView {
	row := appkit.NewHorizontalStackView()
	row.SetTranslatesAutoresizingMaskIntoConstraints(false)
	row.SetSpacing(8)

	lbl := appkit.TextField_LabelWithString(labelText)
	lbl.SetTranslatesAutoresizingMaskIntoConstraints(false)
	lbl.SetAlignment(appkit.TextAlignmentRight)
	lbl.WidthAnchor().ConstraintEqualToConstant(140).SetActive(true)
	row.AddArrangedSubview(lbl)
	row.AddArrangedSubview(field)
	return row
}

func setCheckbox(btn appkit.Button, on bool) {
	if on {
		btn.SetState(appkit.OnState)
	} else {
		btn.SetState(appkit.OffState)
	}
}

func isCheckboxOn(btn appkit.Button) bool {
	return btn.State() == appkit.OnState
}
