// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package ebitenginewidget

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"slices"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

// guestProcess bundles a running guest with the resources needed to tear it down.
type guestProcess struct {
	session *vmhost.GuestSession
	cmd     *exec.Cmd

	// audioStreamsMu guards newAudioStreams.
	audioStreamsMu sync.Mutex

	// newAudioStreams holds the streams appendNewAudioStream recorded (on the session goroutine) and
	// takeNewAudioStreams has not yet drained (on the host goroutine).
	newAudioStreams []*vmhost.GuestAudioStream
}

// appendNewAudioStream records a new guest audio stream. It is the session's OnAudioStream handler, so
// it runs on the session goroutine and must not block or read the stream.
func (gp *guestProcess) appendNewAudioStream(s *vmhost.GuestAudioStream) {
	gp.audioStreamsMu.Lock()
	defer gp.audioStreamsMu.Unlock()
	gp.newAudioStreams = append(gp.newAudioStreams, s)
}

// takeNewAudioStreams drains the streams recorded since the last call, appending them to dst.
func (gp *guestProcess) takeNewAudioStreams(dst []*vmhost.GuestAudioStream) []*vmhost.GuestAudioStream {
	gp.audioStreamsMu.Lock()
	defer gp.audioStreamsMu.Unlock()
	dst = append(dst, gp.newAudioStreams...)
	gp.newAudioStreams = slices.Delete(gp.newAudioStreams, 0, len(gp.newAudioStreams))
	return dst
}

// launchResult is the outcome of an asynchronous launch.
type launchResult struct {
	gp  *guestProcess
	err error
}

// startGuestOptions represents options for startGuest.
type startGuestOptions struct {
	// args is the command-line arguments the guest binary is launched with.
	args []string

	// env is the additional environment variables for the guest process, in "key=value" form, appended
	// to the host's environment.
	env []string
}

// startGuest launches the guest binary at binPath pointed at the host's endpoint and returns a handle
// once it has connected. It is safe to call off the main goroutine; only the returned session's
// screen-touching methods (SetOutsideScreen, CompositeFrame, Close) must run on the host frame.
func startGuest(listener net.Listener, binPath, endpoint string, options *startGuestOptions) (gp *guestProcess, err error) {
	cmd := exec.Command(binPath, options.args...)
	// The endpoint comes last, so the widget's endpoint wins over an env entry of the same name.
	cmd.Env = append(os.Environ(), options.env...)
	cmd.Env = append(cmd.Env, "EBITENGINE_VM_ENDPOINT="+endpoint)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("ebitenginewidget: starting %s failed: %w", binPath, err)
	}
	defer func() {
		// The process outlives this function only on success.
		if err != nil {
			err = errors.Join(err, cmd.Process.Kill(), cmd.Wait())
		}
	}()

	// Both *net.UnixListener and *net.TCPListener provide SetDeadline.
	if err := listener.(interface{ SetDeadline(time.Time) error }).SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return nil, err
	}
	conn, err := listener.Accept()
	if err != nil {
		return nil, &GuestNotConnectedError{
			BinaryPath: binPath,
			Err:        err,
		}
	}
	defer func() {
		// The connection outlives this function only on success (the session takes ownership).
		if err != nil {
			err = errors.Join(err, conn.Close())
		}
	}()

	// The handlers below capture gp, so build it before the session; its session field is filled in once
	// NewGuestSession returns.
	gp = &guestProcess{cmd: cmd}
	session, err := vmhost.NewGuestSession(conn, &vmhost.NewGuestSessionOptions{
		// Bound how long a guest may stop responding mid-operation (a wedged Update, a dead
		// connection), so the wedge surfaces as an error from Err instead of stalling the session
		// forever.
		IdleTimeout: 30 * time.Second,

		// Record each new guest audio stream for updateAudio to play on the host frame.
		OnAudioStream: gp.appendNewAudioStream,

		// Mirror the vibrations the guest requests onto the host. The guest's gamepad IDs match the host's,
		// because the host forwards its own gamepads to the guest. Both functions are concurrent-safe, so
		// running them on the session goroutine is fine.
		OnGamepadVibration: func(v vmhost.GamepadVibration) {
			ebiten.VibrateGamepad(v.GamepadID, &ebiten.VibrateGamepadOptions{
				Duration:        v.Duration,
				StrongMagnitude: v.StrongMagnitude,
				WeakMagnitude:   v.WeakMagnitude,
			})
		},
		OnVibration: func(v vmhost.Vibration) {
			ebiten.Vibrate(&ebiten.VibrateOptions{
				Duration:  v.Duration,
				Magnitude: v.Magnitude,
			})
		},
	})
	if err != nil {
		return nil, err
	}
	gp.session = session
	return gp, nil
}
