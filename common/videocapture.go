package common

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/grafana/xk6-browser/log"
)

// VideoCapturePersister defines the interface for persisting a video capture
type VideoCapturePersister interface {
	Persist(ctx context.Context, path string, data io.Reader) (err error)
}

type VideoFrame struct {
	Content   []byte
	Timestamp int64
}

// VideoFormat represents a video file format.
type VideoFormat string

// Valid video format options.
const (
	// VideoFormatWebM stores video as a series of jpeg files
	VideoFormatWebM VideoFormat = "webm"
)

// String returns the video format as a string
func (f VideoFormat) String() string {
	return f.String()
}

var videoFormatToID = map[string]VideoFormat{ //nolint:gochecknoglobals
	"webm": VideoFormatWebM,
}

type videocapture struct {
	ctx       context.Context
	logger    *log.Logger
	opts      VideoCaptureOptions
	persister VideoCapturePersister
	ffmpegCmd exec.Cmd
	ffmpegIn  io.WriteCloser
	ffmpegOut io.ReadCloser
	lastFrame VideoFrame
}

// creates a new videocapture for a session
func newVideoCapture(
	ctx context.Context,
	logger *log.Logger,
	opts VideoCaptureOptions,
	persister VideoCapturePersister,
) (*videocapture, error) {

	// construct command to start ffmpeg to convert series of images into a video
	// heavily inspired by puppeteer's screen recorder
	// https://github.com/puppeteer/puppeteer/blob/main/packages/puppeteer-core/src/node/ScreenRecorder.ts
	ffmpegCmd := exec.Command(
		"ffmpeg",
		// create video from sequence of images
		"-f", "image2pipe",
		// copy stream without conversion
		"-c:v", "png",
		// set frame rate
		"-framerate", fmt.Sprintf("%d", opts.FrameRate),
		// read from stdin
		"-i", "pipe:0",
		// set output format
		"-f", "webm",
		// set quality
		//"-crf", fmt.Sprintf("%d", opts.Quality),
		// optimize for speed
		"-deadline", "realtime", "-cpu-used", "8",
		// write to sdtout
		//"pipe:1",
		"-y",
		opts.Path, // FIXME: send to stdout
	)
	ffmpegCmd.Stderr = os.Stderr // FIXME: for debugging

	ffmpegIn, err := ffmpegCmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating ffmpeg stdin pipe: %w", err)
	}

	// ffmpegOut, err := ffmpegCmd.StdoutPipe()
	// if err != nil {
	// 	return nil, fmt.Errorf("creating ffmpeg stdout pipe: %w", err)
	// }

	err = ffmpegCmd.Start()
	if err != nil {
		return nil, fmt.Errorf("starting ffmpeg: %w", err)
	}

	return &videocapture{
		ctx:       ctx,
		logger:    logger,
		opts:      opts,
		persister: persister,
		ffmpegCmd: *ffmpegCmd,
		ffmpegIn:  ffmpegIn,
		//		ffmpegOut: ffmpegOut,
	}, nil
}

// HandleFrame sends the frame to the video stream
func (v *videocapture) handleFrame(ctx context.Context, frame *VideoFrame) error {
	// time between frames (in milliseconds)
	step := 1000 / v.opts.FrameRate

	//normalize frame timestamp to a multiple of the step
	timestamp := frame.Timestamp
	if timestamp%step != 0 {
		timestamp = ((timestamp + step) / step) * step
	}

	// repeat last frame to fill video until the current frame
	if v.lastFrame.Timestamp > 0 {
		for ts := v.lastFrame.Timestamp + step; ts < timestamp; ts += step {
			if _, err := v.ffmpegIn.Write(v.lastFrame.Content); err != nil {
				return fmt.Errorf("writing frame: %w", err)
			}
		}
	}

	if _, err := v.ffmpegIn.Write(frame.Content); err != nil {
		return fmt.Errorf("writing frame: %w", err)
	}

	v.lastFrame = VideoFrame{Timestamp: timestamp, Content: frame.Content}

	return nil
}

// Close stops the recording of the video capture
func (v *videocapture) Close(ctx context.Context) error {
	_ = v.ffmpegIn.Close()

	// if err := v.persister.Persist(ctx, v.opts.Path, v.ffmpegOut); err != nil {
	// 	return fmt.Errorf("creating video file: %w", err)
	// }

	return nil
}
