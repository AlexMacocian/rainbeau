package internal

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	infoIcon    = "dialog-information"
	successIcon = "dialog-positive"
	errorIcon   = "dialog-error"
)

var progressFrames = []string{
	"\u283E",
	"\u2837",
	"\u282F",
	"\u281F",
	"\u283B",
	"\u283D",
}

const progressFrameInterval = 150 * time.Millisecond

func NotifyInfo(message string) {
	sendNotification("Rainbeau", message, infoIcon)
}

func NotifySuccess(message string) {
	sendNotification("Rainbeau", message, successIcon)
}

func NotifyError(message string) {
	sendNotification("Rainbeau", message, errorIcon)
}

func sendNotification(title string, message string, icon string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_ = exec.CommandContext(ctx, "notify-send", "-i", icon, title, message).Run()

	fmt.Printf("[%s] %s: %s\n", icon, title, message)
}

type ProgressNotification struct {
	title       string
	baseMessage atomic.Value
	icon        string
	cancel      context.CancelFunc
	done        chan struct{}
	closeOnce   sync.Once
	id          atomic.Uint32
}

func StartProgressNotification(message string) *ProgressNotification {
	return StartProgressNotificationWithIcon(message, infoIcon)
}

func StartProgressNotificationWithIcon(message string, icon string) *ProgressNotification {
	ctx, cancel := context.WithCancel(context.Background())
	progress := &ProgressNotification{
		title:  "Rainbeau",
		icon:   icon,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	progress.baseMessage.Store(message)
	progress.id.Store(progress.sendFrame(progressFrames[0], false))
	fmt.Printf("[%s] %s: %s\n", icon, "Rainbeau", message)

	go progress.animate(ctx)

	return progress
}

func (p *ProgressNotification) UpdateMessage(message string) {
	p.baseMessage.Store(message)
}

func (p *ProgressNotification) animate(ctx context.Context) {
	defer close(p.done)

	ticker := time.NewTicker(progressFrameInterval)
	defer ticker.Stop()

	frameIndex := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			frameIndex = (frameIndex + 1) % len(progressFrames)
			p.sendFrame(progressFrames[frameIndex], true)
		}
	}
}

func (p *ProgressNotification) sendFrame(frame string, replace bool) uint32 {
	args := []string{
		"-i", p.icon,
		"-t", "0",
	}

	currentID := p.id.Load()
	if replace && currentID != 0 {
		args = append(args, "-r", strconv.FormatUint(uint64(currentID), 10))
	} else {
		args = append(args, "-p")
	}

	args = append(args, p.title, fmt.Sprintf("%s %s", frame, p.message()))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "notify-send", args...).Output()
	if err != nil {
		return currentID
	}

	if !replace {
		parsed, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 32)
		if err == nil {
			return uint32(parsed)
		}
	}

	return currentID
}

func (p *ProgressNotification) Close() {
	p.closeOnce.Do(func() {
		p.cancel()

		select {
		case <-p.done:
		case <-time.After(500 * time.Millisecond):
		}

		currentID := p.id.Load()
		if currentID == 0 {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_ = exec.CommandContext(
			ctx,
			"notify-send",
			"-i", p.icon,
			"-r", strconv.FormatUint(uint64(currentID), 10),
			"-t", "1",
			p.title,
			p.message(),
		).Run()
	})
}

func (p *ProgressNotification) message() string {
	value := p.baseMessage.Load()
	message, _ := value.(string)
	return message
}
