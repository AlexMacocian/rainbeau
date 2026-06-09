package main

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

func notifyInfo(title string, message string) {
	sendNotification(title, message, infoIcon)
}

func notifySuccess(title string, message string) {
	sendNotification(title, message, successIcon)
}

func notifyError(title string, message string) {
	sendNotification(title, message, errorIcon)
}

func sendNotification(title string, message string, icon string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_ = exec.CommandContext(ctx, "notify-send", "-i", icon, title, message).Run()

	fmt.Printf("[%s] %s: %s\n", icon, title, message)
}

type progressNotification struct {
	title       string
	baseMessage atomic.Value
	icon        string
	cancel      context.CancelFunc
	done        chan struct{}
	closeOnce   sync.Once
	id          atomic.Uint32
}

func startProgressNotification(title string, message string) *progressNotification {
	return startProgressNotificationWithIcon(title, message, infoIcon)
}

func startProgressNotificationWithIcon(title string, message string, icon string) *progressNotification {
	ctx, cancel := context.WithCancel(context.Background())
	progress := &progressNotification{
		title:  title,
		icon:   icon,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	progress.baseMessage.Store(message)
	progress.id.Store(progress.sendFrame(progressFrames[0], false))
	fmt.Printf("[%s] %s: %s\n", icon, title, message)

	go progress.animate(ctx)

	return progress
}

func (p *progressNotification) updateMessage(message string) {
	p.baseMessage.Store(message)
}

func (p *progressNotification) animate(ctx context.Context) {
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

func (p *progressNotification) sendFrame(frame string, replace bool) uint32 {
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

func (p *progressNotification) close() {
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

func (p *progressNotification) message() string {
	value := p.baseMessage.Load()
	message, _ := value.(string)
	return message
}
