package macui

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/progrium/darwinkit/helper/action"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/progrium/darwinkit/objc"
	"github.com/tiroq/memofy/internal/config"
)

// SettingsFields holds raw form values captured from the UI.
// Decoupled from AppKit so it can be used in pure unit tests.
type SettingsFields struct {
	ZoomEnabled     bool
	ZoomProcesses   string
	ZoomHints       string
	TeamsEnabled    bool
	TeamsProcesses  string
	TeamsHints      string
	MeetEnabled     bool
	MeetProcesses   string
	MeetHints       string
	PollInterval    string
	StartThreshold  string
	StopThreshold   string
	AllowDevUpdates bool
}

// SettingsWindow manages the native, tabbed macOS settings window.
// All config fields are surfaced through the UI – no raw JSON is ever shown.
type SettingsWindow struct {
	detectionRules *config.DetectionConfig

	// macOS window kept alive to prevent GC while visible.
	window    appkit.Window
	isVisible bool

	// Detection tab controls.
	zoomEnabled    appkit.Button
	zoomProcesses  appkit.TextField
	zoomHints      appkit.TextField
	teamsEnabled   appkit.Button
	teamsProcesses appkit.TextField
	teamsHints     appkit.TextField
	meetEnabled    appkit.Button
	meetProcesses  appkit.TextField
	meetHints      appkit.TextField

	// Behavior tab controls.
	pollInterval   appkit.TextField
	startThreshold appkit.TextField
	stopThreshold  appkit.TextField

	// Updates tab controls.
	allowDevUpdates appkit.Button
}

// NewSettingsWindow creates a SettingsWindow and loads the current config.
func NewSettingsWindow() *SettingsWindow {
	rules, err := config.LoadDetectionRules()
	if err != nil {
		log.Printf("Failed to load detection rules: %v – using defaults", err)
		rules = defaultDetectionConfig()
	}
	return &SettingsWindow{detectionRules: rules}
}

// Show builds (or focuses) the native settings window.
// Must be called on the main thread.
func (sw *SettingsWindow) Show() error {
	if sw.isVisible && sw.window.Ptr() != nil {
		if rules, err := config.LoadDetectionRules(); err == nil {
			sw.detectionRules = rules
		}
		sw.reloadFields()
		sw.window.MakeKeyAndOrderFront(nil)
		appkit.Application_SharedApplication().ActivateIgnoringOtherApps(true)
		return nil
	}
	if rules, err := config.LoadDetectionRules(); err == nil {
		sw.detectionRules = rules
	}
	sw.buildWindow()
	sw.window.MakeKeyAndOrderFront(nil)
	appkit.Application_SharedApplication().ActivateIgnoringOtherApps(true)
	sw.isVisible = true
	return nil
}

// buildWindow constructs the NSWindow with a three-tab settings form.
func (sw *SettingsWindow) buildWindow() {
	const (
		winW   = 540.0
		winH   = 590.0
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

	// Save – bottom-right.
	saveBtn.WidthAnchor().ConstraintEqualToConstant(btnW).SetActive(true)
	saveBtn.HeightAnchor().ConstraintEqualToConstant(btnH).SetActive(true)
	saveBtn.TrailingAnchor().ConstraintEqualToAnchorConstant(cv.TrailingAnchor(), -margin).SetActive(true)
	saveBtn.BottomAnchor().ConstraintEqualToAnchorConstant(cv.BottomAnchor(), -margin).SetActive(true)

	// Cancel – left of Save.
	cancelBtn.WidthAnchor().ConstraintEqualToConstant(btnW).SetActive(true)
	cancelBtn.HeightAnchor().ConstraintEqualToConstant(btnH).SetActive(true)
	cancelBtn.TrailingAnchor().ConstraintEqualToAnchorConstant(saveBtn.LeadingAnchor(), -btnGap).SetActive(true)
	cancelBtn.BottomAnchor().ConstraintEqualToAnchorConstant(cv.BottomAnchor(), -margin).SetActive(true)

	// Tab view fills everything above button row.
	tabView.LeadingAnchor().ConstraintEqualToAnchor(cv.LeadingAnchor()).SetActive(true)
	tabView.TrailingAnchor().ConstraintEqualToAnchor(cv.TrailingAnchor()).SetActive(true)
	tabView.TopAnchor().ConstraintEqualToAnchor(cv.TopAnchor()).SetActive(true)
	tabView.BottomAnchor().ConstraintEqualToAnchorConstant(saveBtn.TopAnchor(), -margin).SetActive(true)

	tabView.AddTabViewItem(sw.makeTabItem("Detection Rules", sw.buildDetectionTab))
	tabView.AddTabViewItem(sw.makeTabItem("Behavior", sw.buildBehaviorTab))
	tabView.AddTabViewItem(sw.makeTabItem("Updates", sw.buildUpdatesTab))

	sw.reloadFields()
}

func (sw *SettingsWindow) makeTabItem(label string, builder func() appkit.IView) appkit.TabViewItem {
	item := appkit.NewTabViewItem()
	item.SetLabel(label)
	item.SetView(builder())
	return item
}

// buildDetectionTab creates the "Detection Rules" tab with per-app sections.
func (sw *SettingsWindow) buildDetectionTab() appkit.IView {
	scroll, root := makeScrollStack()

	sw.zoomEnabled = appkit.NewCheckBox("Enable Zoom detection")
	sw.zoomProcesses = makeEditableField("zoom.us, CptHost")
	sw.zoomHints = makeEditableField("Zoom Meeting, Zoom Webinar")
	appendAppSection(root, "Zoom", sw.zoomEnabled, sw.zoomProcesses, sw.zoomHints)

	sw.teamsEnabled = appkit.NewCheckBox("Enable Microsoft Teams detection")
	sw.teamsProcesses = makeEditableField("Microsoft Teams")
	sw.teamsHints = makeEditableField("Meeting, Call")
	appendAppSection(root, "Microsoft Teams", sw.teamsEnabled, sw.teamsProcesses, sw.teamsHints)

	sw.meetEnabled = appkit.NewCheckBox("Enable Google Meet detection")
	sw.meetProcesses = makeEditableField("Google Chrome, Safari, Firefox")
	sw.meetHints = makeEditableField("Google Meet, meet.google.com")
	appendAppSection(root, "Google Meet", sw.meetEnabled, sw.meetProcesses, sw.meetHints)

	pinStackToScroll(root, scroll)
	return scroll
}

// buildBehaviorTab creates the "Behavior" tab with timing and threshold settings.
func (sw *SettingsWindow) buildBehaviorTab() appkit.IView {
	scroll, root := makeScrollStack()

	root.AddArrangedSubview(makeBoldLabel("Detection Timing"))
	root.AddArrangedSubview(makeHintLabel("Controls how responsive the recorder is to meeting state changes."))
	root.AddArrangedSubview(makeSeparator())

	sw.pollInterval = makeEditableField("2")
	sw.pollInterval.SetToolTip("Seconds between each detection check. Range: 1-10.")
	root.AddArrangedSubview(makeLabeledRow("Poll Interval (s):", sw.pollInterval))
	root.AddArrangedSubview(makeHintLabel("How often meeting detection runs. Allowed: 1-10 seconds."))

	sw.startThreshold = makeEditableField("3")
	sw.startThreshold.SetToolTip("Consecutive positive detections needed before recording starts. Range: 1-10.")
	root.AddArrangedSubview(makeLabeledRow("Start Threshold:", sw.startThreshold))
	root.AddArrangedSubview(makeHintLabel("Consecutive detections needed to start recording. Allowed: 1-10."))

	sw.stopThreshold = makeEditableField("6")
	sw.stopThreshold.SetToolTip("Consecutive non-detections needed before recording stops. Must be >= start threshold.")
	root.AddArrangedSubview(makeLabeledRow("Stop Threshold:", sw.stopThreshold))
	root.AddArrangedSubview(makeHintLabel("Consecutive non-detections needed to stop recording. Must be >= start threshold."))

	pinStackToScroll(root, scroll)
	return scroll
}

// buildUpdatesTab creates the "Updates" tab.
func (sw *SettingsWindow) buildUpdatesTab() appkit.IView {
	scroll, root := makeScrollStack()

	root.AddArrangedSubview(makeBoldLabel("Software Updates"))
	root.AddArrangedSubview(makeSeparator())

	sw.allowDevUpdates = appkit.NewCheckBox("Allow pre-release / development updates")
	sw.allowDevUpdates.SetToolTip("When enabled, Memofy also offers release candidates and development builds.")
	root.AddArrangedSubview(sw.allowDevUpdates)
	root.AddArrangedSubview(makeHintLabel("Enable only if you want to test cutting-edge features before stable releases."))

	pinStackToScroll(root, scroll)
	return scroll
}

// makeScrollStack creates a vertical-scroll container (NSScrollView + NSStackView).
func makeScrollStack() (appkit.ScrollView, appkit.StackView) {
	scroll := appkit.NewScrollView()
	scroll.SetHasVerticalScroller(true)
	scroll.SetDrawsBackground(false)
	scroll.SetTranslatesAutoresizingMaskIntoConstraints(false)

	root := appkit.NewVerticalStackView()
	root.SetTranslatesAutoresizingMaskIntoConstraints(false)
	root.SetSpacing(6)
	root.SetEdgeInsets(foundation.EdgeInsets{Top: 16, Left: 16, Bottom: 16, Right: 16})
	root.SetAlignment(appkit.LayoutAttributeLeading)
	return scroll, root
}

// pinStackToScroll sets the stack as document view and pins it to the clip view width.
func pinStackToScroll(root appkit.StackView, scroll appkit.ScrollView) {
	scroll.SetDocumentView(root)
	clip := scroll.ContentView()
	root.LeadingAnchor().ConstraintEqualToAnchor(clip.LeadingAnchor()).SetActive(true)
	root.TrailingAnchor().ConstraintEqualToAnchor(clip.TrailingAnchor()).SetActive(true)
	root.TopAnchor().ConstraintEqualToAnchor(clip.TopAnchor()).SetActive(true)
}

func appendAppSection(parent appkit.StackView, name string,
	enabledCheck appkit.Button, processField, hintsField appkit.TextField) {
	parent.AddArrangedSubview(makeSeparator())
	parent.AddArrangedSubview(makeBoldLabel(name))
	parent.AddArrangedSubview(enabledCheck)
	parent.AddArrangedSubview(makeLabeledRow("Process Names:", processField))
	parent.AddArrangedSubview(makeHintLabel("Comma-separated list of process names to watch."))
	parent.AddArrangedSubview(makeLabeledRow("Window Hints:", hintsField))
	parent.AddArrangedSubview(makeHintLabel("Comma-separated substrings matched against window titles."))
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
	row.SetAlignment(appkit.LayoutAttributeCenterY)

	lbl := appkit.TextField_LabelWithString(labelText)
	lbl.SetTranslatesAutoresizingMaskIntoConstraints(false)
	lbl.SetAlignment(appkit.TextAlignmentRight)
	lbl.WidthAnchor().ConstraintEqualToConstant(140).SetActive(true)

	row.AddArrangedSubview(lbl)
	row.AddArrangedSubview(field)
	return row
}

// reloadFields populates all UI controls from the current detectionRules.
func (sw *SettingsWindow) reloadFields() {
	zoom := sw.detectionRules.RuleByApp("zoom")
	teams := sw.detectionRules.RuleByApp("teams")
	meet := sw.detectionRules.RuleByApp("google_meet")

	setCheckbox(sw.zoomEnabled, zoom != nil && zoom.Enabled)
	setCSV(sw.zoomProcesses, processNames(zoom))
	setCSV(sw.zoomHints, windowHints(zoom))

	setCheckbox(sw.teamsEnabled, teams != nil && teams.Enabled)
	setCSV(sw.teamsProcesses, processNames(teams))
	setCSV(sw.teamsHints, windowHints(teams))

	setCheckbox(sw.meetEnabled, meet != nil && meet.Enabled)
	setCSV(sw.meetProcesses, processNames(meet))
	setCSV(sw.meetHints, windowHints(meet))

	sw.pollInterval.SetStringValue(strconv.Itoa(sw.detectionRules.PollInterval))
	sw.startThreshold.SetStringValue(strconv.Itoa(sw.detectionRules.StartThreshold))
	sw.stopThreshold.SetStringValue(strconv.Itoa(sw.detectionRules.StopThreshold))
	setCheckbox(sw.allowDevUpdates, sw.detectionRules.AllowDevUpdates)
}

// ReadFields captures all UI control values into a SettingsFields.
func (sw *SettingsWindow) ReadFields() SettingsFields {
	return SettingsFields{
		ZoomEnabled:     sw.zoomEnabled.State() == appkit.OnState,
		ZoomProcesses:   sw.zoomProcesses.StringValue(),
		ZoomHints:       sw.zoomHints.StringValue(),
		TeamsEnabled:    sw.teamsEnabled.State() == appkit.OnState,
		TeamsProcesses:  sw.teamsProcesses.StringValue(),
		TeamsHints:      sw.teamsHints.StringValue(),
		MeetEnabled:     sw.meetEnabled.State() == appkit.OnState,
		MeetProcesses:   sw.meetProcesses.StringValue(),
		MeetHints:       sw.meetHints.StringValue(),
		PollInterval:    sw.pollInterval.StringValue(),
		StartThreshold:  sw.startThreshold.StringValue(),
		StopThreshold:   sw.stopThreshold.StringValue(),
		AllowDevUpdates: sw.allowDevUpdates.State() == appkit.OnState,
	}
}

// BuildConfigFromFields converts raw SettingsFields into a validated DetectionConfig.
// No AppKit types are required – fully unit-testable without a display.
func BuildConfigFromFields(f SettingsFields) (*config.DetectionConfig, error) {
	pollInterval, err := strconv.Atoi(strings.TrimSpace(f.PollInterval))
	if err != nil {
		return nil, fmt.Errorf("poll interval must be a number, got %q", f.PollInterval)
	}
	startThresh, err := strconv.Atoi(strings.TrimSpace(f.StartThreshold))
	if err != nil {
		return nil, fmt.Errorf("start threshold must be a number, got %q", f.StartThreshold)
	}
	stopThresh, err := strconv.Atoi(strings.TrimSpace(f.StopThreshold))
	if err != nil {
		return nil, fmt.Errorf("stop threshold must be a number, got %q", f.StopThreshold)
	}

	cfg := &config.DetectionConfig{
		PollInterval:    pollInterval,
		StartThreshold:  startThresh,
		StopThreshold:   stopThresh,
		AllowDevUpdates: f.AllowDevUpdates,
		Rules: []config.DetectionRule{
			{
				Application:  "zoom",
				ProcessNames: ParseCSVField(f.ZoomProcesses),
				WindowHints:  ParseCSVField(f.ZoomHints),
				Enabled:      f.ZoomEnabled,
			},
			{
				Application:  "teams",
				ProcessNames: ParseCSVField(f.TeamsProcesses),
				WindowHints:  ParseCSVField(f.TeamsHints),
				Enabled:      f.TeamsEnabled,
			},
			{
				Application:  "google_meet",
				ProcessNames: ParseCSVField(f.MeetProcesses),
				WindowHints:  ParseCSVField(f.MeetHints),
				Enabled:      f.MeetEnabled,
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ParseCSVField splits s on commas, trims whitespace, and drops empty tokens.
func ParseCSVField(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (sw *SettingsWindow) onSave() {
	fields := sw.ReadFields()
	cfg, err := BuildConfigFromFields(fields)
	if err != nil {
		showSettingsError(err.Error())
		return
	}
	if err := config.SaveDetectionRules(cfg); err != nil {
		showSettingsError(fmt.Sprintf("Failed to save settings: %v", err))
		return
	}
	sw.detectionRules = cfg
	log.Printf("Settings saved: poll=%ds, start=%d, stop=%d, devUpdates=%t",
		cfg.PollInterval, cfg.StartThreshold, cfg.StopThreshold, cfg.AllowDevUpdates)
	if notifErr := SendNotification("Memofy", "Settings Saved", "Detection rules updated"); notifErr != nil {
		log.Printf("Warning: notification failed: %v", notifErr)
	}
	sw.isVisible = false
	sw.window.Close()
}

func (sw *SettingsWindow) onCancel() {
	sw.isVisible = false
	sw.window.Close()
}

func setCheckbox(btn appkit.Button, on bool) {
	if on {
		btn.SetState(appkit.OnState)
	} else {
		btn.SetState(appkit.OffState)
	}
}

func setCSV(field appkit.TextField, values []string) {
	field.SetStringValue(strings.Join(values, ", "))
}

func processNames(rule *config.DetectionRule) []string {
	if rule == nil {
		return nil
	}
	return rule.ProcessNames
}

func windowHints(rule *config.DetectionRule) []string {
	if rule == nil {
		return nil
	}
	return rule.WindowHints
}

func showSettingsError(msg string) {
	log.Printf("Settings error: %s", msg)
	if err := SendErrorNotification("Memofy Settings Error", msg); err != nil {
		log.Printf("Warning: failed to send error notification: %v", err)
	}
}

func defaultDetectionConfig() *config.DetectionConfig {
	return &config.DetectionConfig{
		PollInterval:   2,
		StartThreshold: 3,
		StopThreshold:  6,
		Rules: []config.DetectionRule{
			{Application: "zoom", Enabled: true,
				ProcessNames: []string{"zoom.us", "CptHost"},
				WindowHints:  []string{"Zoom Meeting", "Zoom Webinar"}},
			{Application: "teams", Enabled: true,
				ProcessNames: []string{"Microsoft Teams"},
				WindowHints:  []string{"Meeting", "Call"}},
			{Application: "google_meet", Enabled: true,
				ProcessNames: []string{"Google Chrome", "Safari", "Firefox"},
				WindowHints:  []string{"Google Meet", "meet.google.com"}},
		},
	}
}
