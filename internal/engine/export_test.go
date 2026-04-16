package engine
// export_test.go exposes internal Engine state for white-box tests.
package engine

// DeviceSwitchChCap returns the capacity of the device-switch channel so that
// tests can assert it equals 1 (the invariant that prevents pollMonitor from
// blocking when the loop is busy).
func (e *Engine) DeviceSwitchChCap() int { return cap(e.deviceSwitchCh) }

// IsMicActive exposes the private isMicActive() method for tests.
func (e *Engine) IsMicActive() bool { return e.isMicActive() }
