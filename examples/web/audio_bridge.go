package main

import (
	"encoding/binary"
	"io"
	"sync"

	meowcaller "github.com/purpshell/meowcaller"
)

// maxBufferedAudioBytes caps upload jitter buffering to ~600ms of 16 kHz mono s16le
// PCM, so a network stall doesn't build unbounded latency once the browser catches up.
const maxBufferedAudioBytes = meowcaller.FrameSamples * 2 * 10

// browserAudioSource is a meowcaller.AudioSource fed by raw s16le 16 kHz mono PCM
// pushed from the browser's microphone capture (see handleAudioOut). The call engine
// pulls a frame every 60ms on a fixed timer, so ReadFrame must never block or error on
// underrun — a starved buffer yields a frame of silence instead of stalling the call.
type browserAudioSource struct {
	mu     sync.Mutex
	buf    []byte
	closed bool
}

func newBrowserAudioSource() *browserAudioSource {
	return &browserAudioSource{}
}

// push appends browser-captured PCM, dropping the oldest bytes past the jitter cap.
func (s *browserAudioSource) push(pcm []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || len(pcm) == 0 {
		return
	}
	s.buf = append(s.buf, pcm...)
	if excess := len(s.buf) - maxBufferedAudioBytes; excess > 0 {
		s.buf = s.buf[excess:]
	}
}

func (s *browserAudioSource) ReadFrame() ([]float32, error) {
	const frameBytes = meowcaller.FrameSamples * 2
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, io.EOF
	}
	frame := make([]float32, meowcaller.FrameSamples)
	if len(s.buf) < frameBytes {
		return frame, nil // silence while the browser's buffer is starved
	}
	for i := range frame {
		frame[i] = float32(int16(binary.LittleEndian.Uint16(s.buf[2*i:]))) / 32768.0
	}
	s.buf = s.buf[frameBytes:]
	return frame, nil
}

func (s *browserAudioSource) Close() error {
	s.mu.Lock()
	s.closed = true
	s.buf = nil
	s.mu.Unlock()
	return nil
}

// newBrowserAudioSink returns an AudioSink that converts each decoded 16 kHz mono
// frame to s16le PCM and forwards it to the browser over the existing SSE stream, for
// playback via Web Audio (the "audio" event in the page script).
func newBrowserAudioSink(bridge *videoBridge) meowcaller.AudioSink {
	return meowcaller.SinkFunc(func(frame []float32) {
		pcm := make([]byte, len(frame)*2)
		for i, sample := range frame {
			switch {
			case sample > 1:
				sample = 1
			case sample < -1:
				sample = -1
			}
			var v int16
			if sample < 0 {
				v = int16(sample * 32768)
			} else {
				v = int16(sample * 32767)
			}
			binary.LittleEndian.PutUint16(pcm[2*i:], uint16(v))
		}
		bridge.WriteAudioFrame(pcm)
	})
}
