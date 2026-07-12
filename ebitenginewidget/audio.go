// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Guigui Authors

package ebitenginewidget

import (
	"fmt"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

// updateAudio plays each guest player on its own host player, so the guest's players stay unmixed.
func (e *Ebitengine) updateAudio() error {
	// Adopt the streams handed to the OnAudioStream handler since the last tick, and drop any the guest
	// has closed (a closed stream never plays again, so audioStreams would otherwise grow unbounded). A
	// finished-but-open stream is kept: a seek-and-replay reuses it and fires no new handler.
	e.audioStreams = e.gp.takeNewAudioStreams(e.audioStreams)
	e.audioStreams = slices.DeleteFunc(e.audioStreams, func(s *vmhost.GuestAudioStream) bool {
		return s.IsClosed()
	})

	if e.audioDisabled {
		// Stop the players started while audio was enabled. The streams stay recorded, so re-enabling
		// starts a fresh player for whatever is playing then.
		e.closeAudioPlayers()
		return nil
	}

	rate := e.gp.session.AudioSampleRate()
	if rate == 0 {
		// The guest has not produced audio yet, so its sample rate is unknown.
		return nil
	}
	if e.audioContext == nil {
		// Reuse an existing process-wide context (only one is allowed); create one only if none exists.
		if c := audio.CurrentContext(); c != nil {
			e.audioContext = c
		} else {
			e.audioContext = audio.NewContext(rate)
		}
	}
	if e.audioContext.SampleRate() != rate {
		// One audio context per process, so a guest at a different rate than the context cannot be played
		// without resampling; skip its audio.
		if !e.audioRateWarned {
			e.audioRateWarned = true
			e.dispatchError(&AudioSampleRateMismatchError{
				CurrentSampleRate: e.audioContext.SampleRate(),
				NewSampleRate:     rate,
			})
		}
		return nil
	}

	for _, stream := range e.audioStreams {
		hp := e.audioPlayers[stream]
		if hp == nil {
			// Start a host player only for a stream that is currently playing; a finished or paused stream
			// gets none, and a replayed one gets a fresh host player when it plays again.
			if !stream.IsPlaying() {
				continue
			}
			var err error
			hp, err = e.audioContext.NewPlayerF32(stream)
			if err != nil {
				return fmt.Errorf("ebitenginewidget: creating an audio player: %w", err)
			}
			// oto reads ahead this far, pulling the samples from the guest; keep it small for low latency
			// but large enough to cover a momentarily busy session.
			hp.SetBufferSize(time.Second / 20)
			hp.Play()
			if e.audioPlayers == nil {
				e.audioPlayers = map[*vmhost.GuestAudioStream]*audio.Player{}
			}
			e.audioPlayers[stream] = hp
		}
		// The forwarded samples are raw, so apply the guest player's volume on the host side.
		hp.SetVolume(stream.Volume())
	}
	// Close finished host players. A host player keeps playing until its guest stream reaches EOF and
	// the buffered tail has played out (or the stream is closed), so closing here happens only after the
	// stream has fully sounded. The stream stays in audioStreams so a replay can start a fresh host
	// player.
	for stream, hp := range e.audioPlayers {
		if !hp.IsPlaying() {
			if err := hp.Close(); err != nil {
				e.dispatchError(fmt.Errorf("ebitenginewidget: closing an audio player: %w", err))
			}
			delete(e.audioPlayers, stream)
		}
	}
	return nil
}

// closeAudioPlayers closes all the host audio players.
func (e *Ebitengine) closeAudioPlayers() {
	for stream, hp := range e.audioPlayers {
		if err := hp.Close(); err != nil {
			e.dispatchError(fmt.Errorf("ebitenginewidget: closing an audio player: %w", err))
		}
		delete(e.audioPlayers, stream)
	}
}
