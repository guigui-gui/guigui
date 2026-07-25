// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package ebitenginewidget

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost/vmhostutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/guigui-gui/guigui"
)

var _ guigui.Widget = (*Ebitengine)(nil)

var (
	ebitengineEventLaunched     guigui.EventKey = guigui.GenerateEventKey()
	ebitengineEventExited       guigui.EventKey = guigui.GenerateEventKey()
	ebitengineEventError        guigui.EventKey = guigui.GenerateEventKey()
	ebitengineEventTPSRequested guigui.EventKey = guigui.GenerateEventKey()
)

// GuestNotConnectedError is reported to the OnError handler when the launched binary did not connect
// to the widget as a virtualization guest, typically because it is not an Ebitengine program built
// with the "ebitenginevm" build tag.
type GuestNotConnectedError struct {
	// BinaryPath is the path of the launched binary.
	BinaryPath string

	// Err is the underlying error.
	Err error
}

func (e *GuestNotConnectedError) Error() string {
	return fmt.Sprintf("ebitenginewidget: %s did not connect as a guest (is it built with -tags ebitenginevm?): %v", e.BinaryPath, e.Err)
}

func (e *GuestNotConnectedError) Unwrap() error {
	return e.Err
}

// AudioSampleRateMismatchError is reported to the OnError handler when the guest's audio cannot be
// played because the guest's sample rate does not match the process's audio context.
type AudioSampleRateMismatchError struct {
	// CurrentSampleRate is the audio context's sample rate.
	CurrentSampleRate int

	// NewSampleRate is the guest's sample rate.
	NewSampleRate int
}

func (e *AudioSampleRateMismatchError) Error() string {
	return fmt.Sprintf("ebitenginewidget: the guest's audio sample rate (%d) does not match the audio context's (%d)", e.NewSampleRate, e.CurrentSampleRate)
}

// ebitengineState is the lifecycle state of the widget's launcher: the one-time host setup and the
// asynchronous guest launches. Whether a guest is running is tracked separately by the gp field, since
// a guest keeps running while the next one is launching.
type ebitengineState int

const (
	// ebitengineStateIdle means no launch is in flight.
	ebitengineStateIdle ebitengineState = iota

	// ebitengineStateLaunching means a guest launch is in flight; its result arrives on results.
	ebitengineStateLaunching

	// ebitengineStateClosed means Close was called.
	ebitengineStateClosed
)

// Ebitengine is a Guigui widget that runs a prebuilt Ebitengine binary as a guest process and renders
// it in place. Its zero value is ready to use; a binary is selected with [Ebitengine.SetBinaryPath].
//
// The widget owns a child process and related OS resources. Call [Ebitengine.Close] to release them;
// otherwise they are reclaimed only when the process exits.
type Ebitengine struct {
	guigui.DefaultWidget

	// binPath is the requested guest binary.
	binPath string

	// currentBinPath is the binary path most recently processed: launched, attempted, or stopped with.
	// It differs from binPath exactly when a new launch or stop is due.
	currentBinPath string

	// commandArgs is the command-line arguments the guest binary is launched with.
	commandArgs []string

	// commandEnv is the additional environment variables for the guest process, in "key=value" form.
	commandEnv []string

	tps                     int
	tpsSet                  bool
	inputForwardingDisabled bool
	audioDisabled           bool

	// requestedTPS is the current guest's own requested rate (ebiten.SyncWithFPS resolved), 0 until its
	// first tick is processed.
	requestedTPS int

	// tpsReported guards reporting requestedTPS once per guest.
	tpsReported bool

	// state is the launcher's lifecycle state; the zero value is ebitengineStateIdle.
	state ebitengineState

	// dir is the temporary directory holding the guests' sockets.
	dir string

	// launchGen numbers the launches, giving each guest's listener its own socket path.
	launchGen int

	// launchResultCh receives the outcome of the launch in flight; it is created by setup.
	launchResultCh chan launchResult

	gp          *guestProcess
	guestScreen *ebiten.Image
	screenSet   bool

	// prevGuestScreen is the previous guest screen, drawn stretched to the bounds while the resized
	// guestScreen has not yet received a frame, so the widget keeps showing the last frame during a
	// resize.
	prevGuestScreen *ebiten.Image

	// screenPresented reports whether a frame has been composited into guestScreen since its creation.
	screenPresented bool

	// audioContext is the host audio context guest streams are played on; it is created at the first
	// guest's sample rate, or adopted from the process's existing context.
	audioContext *audio.Context

	// audioPlayers maps each guest stream to the host player playing it.
	audioPlayers map[*vmhost.GuestAudioStream]*audio.Player

	// audioStreams holds the guest's audio streams, from the OnAudioStream handler.
	audioStreams []*vmhost.GuestAudioStream

	// audioRateWarned guards reporting a sample-rate mismatch once per guest.
	audioRateWarned bool

	// textInput is the guest's text-input session awaiting or being served, nil when none.
	textInput *vmhost.GuestTextInput

	// composerForwarder serves textInput on the host's platform IME.
	composerForwarder vmhostutil.ComposerForwarder

	// pressedKeys holds the keys whose presses were forwarded to the guest and whose releases were not
	// yet, so releases can reach the guest even when the widget is unfocused.
	pressedKeys map[ebiten.Key]struct{}

	// pressedMouseButtons holds the mouse buttons whose presses were forwarded to the guest and whose
	// releases were not yet, so releases can reach the guest after a drag leaves the widget.
	pressedMouseButtons map[ebiten.MouseButton]struct{}

	forwardedTouches map[ebiten.TouchID]struct{}
	keyBuf           []ebiten.Key
	runeBuf          []rune
	touchIDsBuf      []ebiten.TouchID
	gamepadIDsBuf    []ebiten.GamepadID
	gamepadStatesBuf []vmhost.GamepadState

	tickAccum int

	// manualTicks is the number of updates requested by AdvanceTicks and not yet applied.
	manualTicks int

	// pendingErrsMu guards pendingErrs.
	pendingErrsMu sync.Mutex

	// pendingErrs holds errors recorded on other goroutines by dispatchErrorAsync, reported at the
	// widget's next tick.
	pendingErrs []error
}

// SetBinaryPath sets the path to the guest binary to run. The binary must be an Ebitengine program
// built with the "ebitenginevm" build tag, against the host's Ebitengine version. Changing the path
// launches the guest, and an empty path stops the running guest; setting the current value again has
// no effect.
func (e *Ebitengine) SetBinaryPath(path string) {
	e.binPath = path
}

// SetCommandArgs sets the command-line arguments the guest binary is launched with. They take effect at
// the next launch.
func (e *Ebitengine) SetCommandArgs(args []string) {
	e.commandArgs = slices.Clone(args)
}

// SetCommandEnv sets additional environment variables for the guest process, each in "key=value" form,
// appended to the host's environment. They take effect at the next launch.
func (e *Ebitengine) SetCommandEnv(env []string) {
	e.commandEnv = slices.Clone(env)
}

// SetTPS overrides the rate, in ticks per second, at which the guest is updated; 0 pauses it. Without
// SetTPS, the guest is paced at the rate its own game requests (reported by [Ebitengine.OnTPSRequested]).
func (e *Ebitengine) SetTPS(tps int) {
	e.tps = tps
	e.tpsSet = true
}

// AdvanceTicks requests n additional updates on the guest, applied at the widget's next tick while a
// guest is running. Combined with SetTPS(0), the widget's owner fully controls the guest's pace.
// n must not be negative.
func (e *Ebitengine) AdvanceTicks(n int) {
	if n < 0 {
		panic("ebitenginewidget: AdvanceTicks count must not be negative")
	}
	e.manualTicks += n
}

// SetInputForwardingEnabled sets whether the window's input is forwarded to the guest. It is enabled by
// default.
func (e *Ebitengine) SetInputForwardingEnabled(enabled bool) {
	e.inputForwardingDisabled = !enabled
}

// SetAudioEnabled sets whether the audio the guest plays is played on the host. It is enabled by
// default. While disabled, the guest's audio sources are not consumed, so the guest observes no audio
// playback progress.
func (e *Ebitengine) SetAudioEnabled(enabled bool) {
	e.audioDisabled = !enabled
}

// OnLaunched sets a handler called when a guest has connected and started running.
func (e *Ebitengine) OnLaunched(f func(context *guigui.Context)) {
	guigui.SetEventHandler(e, ebitengineEventLaunched, f)
}

// OnExited sets a handler called when the guest has terminated normally.
func (e *Ebitengine) OnExited(f func(context *guigui.Context)) {
	guigui.SetEventHandler(e, ebitengineEventExited, f)
}

// OnError sets a handler called when launching or driving the guest fails. Without a handler, failures
// are logged.
func (e *Ebitengine) OnError(f func(context *guigui.Context, err error)) {
	guigui.SetEventHandler(e, ebitengineEventError, f)
}

// dispatchError reports err to the OnError handler, or logs it when no handler is registered, so a
// failure is never dropped silently.
func (e *Ebitengine) dispatchError(err error) {
	if _, ok := guigui.DispatchEvent(e, ebitengineEventError, err); !ok {
		slog.Error(err.Error())
	}
}

// dispatchErrorAsync records err to be reported like dispatchError at the widget's next tick. Unlike
// dispatchError, it may be called from any goroutine.
func (e *Ebitengine) dispatchErrorAsync(err error) {
	e.pendingErrsMu.Lock()
	defer e.pendingErrsMu.Unlock()
	e.pendingErrs = append(e.pendingErrs, err)
}

// OnTPSRequested sets a handler called once per guest, after its first tick, with the ticks-per-second
// the guest's own game requests via [ebiten.SetTPS] ([ebiten.SyncWithFPS] resolved to the host's rate).
func (e *Ebitengine) OnTPSRequested(f func(context *guigui.Context, tps int)) {
	guigui.SetEventHandler(e, ebitengineEventTPSRequested, f)
}

// Session returns the underlying guest session for advanced control, or nil when no guest is running.
// The session must not be used after the guest exits or after [Ebitengine.Close].
func (e *Ebitengine) Session() *vmhost.GuestSession {
	if e.gp == nil {
		return nil
	}
	return e.gp.session
}

// IsRunning reports whether a guest is currently running.
func (e *Ebitengine) IsRunning() bool {
	return e.gp != nil
}

func (e *Ebitengine) Tick(context *guigui.Context, widgetBounds *guigui.WidgetBounds) error {
	// Report errors recorded on other goroutines. This runs even when the widget is closed, since the
	// reaper goroutines can fail after Close.
	e.pendingErrsMu.Lock()
	errs := e.pendingErrs
	e.pendingErrs = nil
	e.pendingErrsMu.Unlock()
	for _, err := range errs {
		e.dispatchError(err)
	}

	if e.state == ebitengineStateClosed {
		return nil
	}

	// Adopt an asynchronously-built guest once it is ready.
	select {
	case r := <-e.launchResultCh:
		e.state = ebitengineStateIdle
		if r.err != nil {
			e.dispatchError(r.err)
		} else {
			e.closeGuest()
			e.gp = r.gp
			e.screenSet = false
			e.requestedTPS = 0
			e.tpsReported = false
			e.audioRateWarned = false
			// A remainder accumulated for the previous guest must not tick the new one.
			e.tickAccum = 0
			guigui.DispatchEvent(e, ebitengineEventLaunched)
		}
	default:
	}

	// A change to the requested binary (re)launches, and a change to an empty path stops the current
	// guest. currentBinPath records the path as processed whether or not a launch started, so a guest
	// that has exited is not restarted and a failed attempt is reported once rather than retried.
	if e.binPath != e.currentBinPath && e.state == ebitengineStateIdle {
		if e.binPath == "" {
			e.closeGuest()
		} else {
			e.launch(e.binPath)
		}
		e.currentBinPath = e.binPath
	}

	if e.gp == nil {
		return nil
	}

	bounds := widgetBounds.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return nil
	}

	// The widget's bounds are in device-scale pixels; the application scale is applied by the widget
	// itself, as sibling widgets do when sizing themselves. The guest screen is bounds/AppScale physical
	// pixels — the guest renders at the device scale — and Draw scales it up to fill the bounds, so the
	// guest zooms with the application scale like the rest of the UI.
	appScale := context.AppScale()
	sw := max(1, int(math.Round(float64(bounds.Dx())/appScale)))
	sh := max(1, int(math.Round(float64(bounds.Dy())/appScale)))
	if e.guestScreen == nil || e.guestScreen.Bounds().Dx() != sw || e.guestScreen.Bounds().Dy() != sh {
		if e.guestScreen != nil {
			// The outgoing screen holds the last frame; Draw keeps presenting it until the guest
			// delivers a frame at the new size.
			if e.screenPresented {
				if e.prevGuestScreen != nil {
					e.prevGuestScreen.Deallocate()
				}
				e.prevGuestScreen = e.guestScreen
			} else {
				e.guestScreen.Deallocate()
			}
		}
		e.guestScreen = ebiten.NewImage(sw, sh)
		e.screenPresented = false
		e.screenSet = false
	}
	if !e.screenSet {
		if err := e.gp.session.SetOutsideScreen(e.guestScreen); err != nil {
			e.dispatchError(err)
			e.closeGuest()
			return nil
		}
		e.screenSet = true
	}

	if !e.inputForwardingDisabled {
		e.forwardInput(context, widgetBounds)
	}
	// updateTextInput runs even while input forwarding is disabled: the guest's text-input sessions
	// are still tracked, and a host IME session in progress is stopped rather than left composing.
	e.updateTextInput(context, widgetBounds)
	n := e.guestTickCount() + e.manualTicks
	e.manualTicks = 0
	e.gp.session.AdvanceTicks(n)

	// The session runs the guest on its own goroutine; a termination or error surfaces here.
	if err := e.gp.session.Err(); err != nil {
		if errors.Is(err, ebiten.Termination) {
			guigui.DispatchEvent(e, ebitengineEventExited)
		} else {
			e.dispatchError(err)
		}
		e.closeGuest()
		return nil
	}

	// Once the guest has processed its first tick, it has reported its requested TPS. Adopt it as the
	// default drive rate and report it, so the guest is paced as its own game intends.
	if !e.tpsReported && e.gp.session.ProcessedTicks() > 0 {
		tps := e.gp.session.RequestedTPS()
		if tps == ebiten.SyncWithFPS {
			// SyncWithFPS ties the guest's rate to the display's refresh rate. The guest is advanced from
			// this tick, so approximate that intent with the host's tick rate.
			tps = ebiten.TPS()
			if tps <= 0 {
				tps = ebiten.DefaultTPS
			}
		}
		e.requestedTPS = tps
		e.tpsReported = true
		guigui.DispatchEvent(e, ebitengineEventTPSRequested, tps)
	}

	if err := e.updateAudio(); err != nil {
		e.dispatchError(err)
		e.closeGuest()
		return nil
	}

	// The guest advanced, so a new frame is due: keep the widget repainting.
	if n > 0 {
		guigui.RequestRedraw(e)
	}
	return nil
}

func (e *Ebitengine) Draw(context *guigui.Context, widgetBounds *guigui.WidgetBounds, dst *ebiten.Image) {
	if e.state == ebitengineStateClosed || e.gp == nil || e.guestScreen == nil || !e.screenSet {
		return
	}
	e.gp.session.AdvanceFrame()
	if e.gp.session.CompositeFrame() {
		e.screenPresented = true
		if e.prevGuestScreen != nil {
			e.prevGuestScreen.Deallocate()
			e.prevGuestScreen = nil
		}
	}
	img := e.guestScreen
	if !e.screenPresented && e.prevGuestScreen != nil {
		img = e.prevGuestScreen
	}
	bounds := widgetBounds.Bounds()
	op := &ebiten.DrawImageOptions{}
	// The guest screen is bounds/AppScale pixels; scale it up to fill the bounds.
	sw := img.Bounds().Dx()
	sh := img.Bounds().Dy()
	if sw != bounds.Dx() || sh != bounds.Dy() {
		op.GeoM.Scale(float64(bounds.Dx())/float64(sw), float64(bounds.Dy())/float64(sh))
		op.Filter = ebiten.FilterPixelated
	}
	op.GeoM.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))
	dst.DrawImage(img, op)
}

// HandlePointingInput focuses the widget when it is clicked, so subsequent keyboard input is forwarded
// to the guest.
func (e *Ebitengine) HandlePointingInput(context *guigui.Context, widgetBounds *guigui.WidgetBounds) guigui.HandleInputResult {
	if e.state == ebitengineStateClosed || e.gp == nil {
		return guigui.HandleInputResult{}
	}
	if !widgetBounds.IsHitAtCursor() {
		return guigui.HandleInputResult{}
	}
	if slices.ContainsFunc([]ebiten.MouseButton{ebiten.MouseButtonLeft, ebiten.MouseButtonRight, ebiten.MouseButtonMiddle}, inpututil.IsMouseButtonJustPressed) {
		context.SetFocused(e, true)
		return guigui.HandleInputByWidget(e)
	}
	return guigui.HandleInputResult{}
}

// CursorShape returns the cursor shape the guest's game requests via [ebiten.SetCursorShape].
func (e *Ebitengine) CursorShape(context *guigui.Context, widgetBounds *guigui.WidgetBounds) (ebiten.CursorShapeType, bool) {
	if e.state == ebitengineStateClosed || e.gp == nil || e.inputForwardingDisabled {
		return 0, false
	}
	return e.gp.session.CursorShape(), true
}

// Close stops the guest and releases the widget's OS resources. Close is idempotent.
func (e *Ebitengine) Close() error {
	if e.state == ebitengineStateClosed {
		return nil
	}
	launching := e.state == ebitengineStateLaunching
	e.state = ebitengineStateClosed
	e.closeGuest()

	// A launch may be in flight; its goroutine always sends exactly one result, which the closed widget
	// never adopts. Reap it so a guest that has already connected does not outlive the widget. The
	// session was never composited, so closing it off the host's frame is safe.
	if launching {
		ch := e.launchResultCh
		go func() {
			r := <-ch
			if r.gp == nil {
				return
			}
			if err := r.gp.session.Close(); err != nil {
				e.dispatchErrorAsync(fmt.Errorf("ebitenginewidget: closing the unadopted guest: %w", err))
			}
			if err := r.gp.cmd.Wait(); err != nil {
				e.dispatchErrorAsync(fmt.Errorf("ebitenginewidget: waiting for the unadopted guest: %w", err))
			}
		}()
	}

	var err error
	if e.dir != "" {
		err = errors.Join(err, os.RemoveAll(e.dir))
		e.dir = ""
	}
	if e.guestScreen != nil {
		e.guestScreen.Deallocate()
		e.guestScreen = nil
	}
	if e.prevGuestScreen != nil {
		e.prevGuestScreen.Deallocate()
		e.prevGuestScreen = nil
	}
	return err
}

// effectiveTPS returns the rate the guest is driven at: an explicit [Ebitengine.SetTPS] override when
// set, otherwise the guest's own requested rate, falling back to [ebiten.DefaultTPS] until that is known.
func (e *Ebitengine) effectiveTPS() int {
	if e.tpsSet {
		return e.tps
	}
	if e.requestedTPS > 0 {
		return e.requestedTPS
	}
	return ebiten.DefaultTPS
}

// guestTickCount returns how many ticks to advance the guest this tick so it runs at the effective TPS
// on average, regardless of the host's tick rate.
func (e *Ebitengine) guestTickCount() int {
	tps := e.effectiveTPS()
	// A non-positive rate pauses the guest.
	if tps <= 0 {
		return 0
	}
	hostTPS := ebiten.TPS()
	if hostTPS <= 0 {
		// Guard against a non-positive rate (e.g. SyncWithFPS).
		hostTPS = ebiten.DefaultTPS
	}
	// Every hostTPS accumulated units make one tick, so hostTPS ticks (one second) yield tps ticks.
	e.tickAccum += tps
	n := e.tickAccum / hostTPS
	e.tickAccum %= hostTPS
	return n
}

// launch kicks off an asynchronous launch of binPath. Connecting runs in a goroutine so the UI stays
// responsive; the result is adopted in [Ebitengine.Tick].
func (e *Ebitengine) launch(binPath string) {
	if e.dir == "" {
		if err := e.setup(); err != nil {
			e.dispatchError(err)
			return
		}
	}
	e.state = ebitengineStateLaunching
	e.launchGen++

	// A socket path is never reused, so a lingering file from an earlier launch cannot make the listen
	// fail.
	sockPath := filepath.Join(e.dir, fmt.Sprintf("s%d", e.launchGen))
	options := &startGuestOptions{
		args: e.commandArgs,
		env:  e.commandEnv,
	}
	go func() {
		gp, err := startGuest(binPath, sockPath, options)
		e.launchResultCh <- launchResult{gp: gp, err: err}
	}()
}

// setup performs the one-time host setup: a temporary directory for the guests' sockets.
func (e *Ebitengine) setup() error {
	e.launchResultCh = make(chan launchResult, 1)

	// A short directory name keeps the Unix socket paths within sun_path (~104 bytes on macOS).
	dir, err := os.MkdirTemp("", "guigui-vm")
	if err != nil {
		return err
	}
	e.dir = dir
	return nil
}

// closeGuest stops the current guest, if any. Close releases the mirror images that [Ebitengine.Draw]
// composites, so it runs on the host's goroutine; reaping the process is left to a goroutine so a slow
// exit cannot stall a tick.
func (e *Ebitengine) closeGuest() {
	if e.gp == nil {
		return
	}
	gp := e.gp
	e.gp = nil
	e.screenSet = false

	e.closeAudioPlayers()
	e.audioStreams = slices.Delete(e.audioStreams, 0, len(e.audioStreams))

	// The forwarded presses belong to the guest being closed; the next guest starts with nothing held.
	clear(e.pressedKeys)
	clear(e.pressedMouseButtons)
	clear(e.forwardedTouches)

	// The text-input session being served belongs to the guest being closed.
	e.textInput = nil
	e.composerForwarder.Reset()

	if err := gp.session.Close(); err != nil {
		e.dispatchError(fmt.Errorf("ebitenginewidget: closing the guest: %w", err))
	}
	go func() {
		// Reaping happens off the tick and has no caller to return to, so log rather than discard. The
		// binary is caller-owned, so it is not removed here.
		if err := gp.cmd.Wait(); err != nil {
			e.dispatchErrorAsync(fmt.Errorf("ebitenginewidget: waiting for the guest: %w", err))
		}
	}()
}
