package services

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type CompositionClip struct {
	Path       string
	Transition string
	Duration   float64
}

type VideoService struct {
	storage *StorageService
}

func NewVideoService(storage *StorageService) *VideoService {
	return &VideoService{storage: storage}
}

// StandardizeClip ensures a video is 1080p, 30fps, and has a consistent pixel format.
func (s *VideoService) StandardizeClip(ctx context.Context, inputPath, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputPath,
		"-vf", "scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2,setsar=1",
		"-r", "30",
		"-c:v", "libx264", "-preset", "fast", "-crf", "23",
		"-c:a", "aac", "-b:a", "128k",
		"-y", outputPath)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg standardization failed: %w (output: %s)", err, string(output))
	}
	return nil
}

// GetVideoDuration returns the duration of a video file in seconds using ffprobe.
func (s *VideoService) GetVideoDuration(ctx context.Context, path string) (float64, error) {
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var duration float64
	fmt.Sscanf(strings.TrimSpace(string(output)), "%f", &duration)
	return duration, nil
}

// ComposeVideos stitches multiple clips together with transitions.
func (s *VideoService) ComposeVideos(ctx context.Context, clips []CompositionClip, outputPath string) error {
	if len(clips) == 0 {
		return fmt.Errorf("no clips provided for composition")
	}
	if len(clips) == 1 {
		// Just copy or symlink if only one clip
		return s.StandardizeClip(ctx, clips[0].Path, outputPath)
	}

	// 1. Get durations for all clips to calculate offsets
	durations := make([]float64, len(clips))
	for i, clip := range clips {
		d, err := s.GetVideoDuration(ctx, clip.Path)
		if err != nil {
			return fmt.Errorf("failed to get duration for clip %d: %w", i, err)
		}
		durations[i] = d
	}

	// 2. Build the filter graph
	// For xfade, we chain them: ((c0, c1) -> v01, c2) -> v012, etc.
	filterGraph := ""
	lastOut := "v0"
	currentOffset := 1.0 // Start offset calculation

	// Initial inputs
	inputs := []string{}
	for i, clip := range clips {
		inputs = append(inputs, "-i", clip.Path)
		if i == 0 {
			filterGraph += fmt.Sprintf("[%d:v]settb=AVTB,setpts=PTS-STARTPTS[v0];", i)
		} else {
			filterGraph += fmt.Sprintf("[%d:v]settb=AVTB,setpts=PTS-STARTPTS[v%d];", i, i)
		}
	}

	currentOffset = durations[0]
	for i := 1; i < len(clips); i++ {
		transType := clips[i].Transition
		if transType == "" {
			transType = "fade"
		}
		transDuration := clips[i].Duration
		if transDuration == 0 {
			transDuration = 1.0 // Default 1s transition
		}

		offset := currentOffset - transDuration
		nextOut := fmt.Sprintf("vout%d", i)
		filterGraph += fmt.Sprintf("[%s][%s]xfade=transition=%s:duration=%f:offset=%f[%s];", 
			lastOut, fmt.Sprintf("v%d", i), transType, transDuration, offset, nextOut)
		
		lastOut = nextOut
		currentOffset = offset + durations[i]
	}

	// Audio handling (simple mix/concat for now - xfade for audio is crossfade filter)
	filterGraph += fmt.Sprintf("[%s]format=yuv420p[outv]", lastOut)

	args := append(inputs, "-filter_complex", filterGraph, "-map", "[outv]", "-c:v", "libx264", "-y", outputPath)
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	log.Printf("Running FFmpeg composition: ffmpeg %s", strings.Join(args, " "))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg composition failed: %w (output: %s)", err, string(output))
	}

	return nil
}
