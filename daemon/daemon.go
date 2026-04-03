package daemon

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cheung/conchtalk-dlc/relay"
	"github.com/cheung/conchtalk-dlc/skills"
	"github.com/cheung/conchtalk-dlc/tools"
)

type Daemon struct {
	client   *relay.Client
	registry *tools.Registry
	skills   []relay.SkillDefinition
	sem      chan struct{}
}

// HandleMessage implements relay.MessageHandler.
func (d *Daemon) HandleMessage(msg relay.IncomingMessage) {
	switch msg.Type {
	case "tool_call":
		go d.executeTool(msg)
	case "status":
		log.Printf("[daemon] client status: %s", msg.Client)
	default:
		log.Printf("[daemon] unknown message type: %s", msg.Type)
	}
}

func Run(token, server string) error {
	d := &Daemon{
		registry: tools.NewRegistry(),
		skills:   skills.Load(),
		sem:      make(chan struct{}, 16),
	}

	d.client = relay.NewClient(server, token, d)
	d.client.OnConnect = d.SendCapabilities

	// Handle shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("[daemon] shutting down...")
		d.client.Stop()
		cancel()
	}()

	go d.client.Run()

	<-ctx.Done()
	return nil
}

func (d *Daemon) executeTool(msg relay.IncomingMessage) {
	// Acquire semaphore slot
	select {
	case d.sem <- struct{}{}:
		defer func() { <-d.sem }()
	default:
		if err := d.client.Send(relay.OutgoingMessage{
			Type:  "tool_error",
			ID:    msg.ID,
			Error: "too many concurrent calls",
		}); err != nil {
			log.Printf("[daemon] send failed: %v", err)
		}
		return
	}

	tool, err := d.registry.Get(msg.Tool)
	if err != nil {
		if err := d.client.Send(relay.OutgoingMessage{
			Type:  "tool_error",
			ID:    msg.ID,
			Error: err.Error(),
		}); err != nil {
			log.Printf("[daemon] send failed: %v", err)
		}
		return
	}

	var args map[string]interface{}
	if err := json.Unmarshal(msg.Arguments, &args); err != nil {
		if err := d.client.Send(relay.OutgoingMessage{
			Type:  "tool_error",
			ID:    msg.ID,
			Error: "invalid arguments: " + err.Error(),
		}); err != nil {
			log.Printf("[daemon] send failed: %v", err)
		}
		return
	}

	streamCb := func(stream string, data string) {
		if err := d.client.Send(relay.OutgoingMessage{
			Type:   "tool_output",
			ID:     msg.ID,
			Stream: stream,
			Data:   data,
		}); err != nil {
			log.Printf("[daemon] send failed: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	result := tool.Execute(ctx, args, streamCb)

	if result.Error != "" {
		if err := d.client.Send(relay.OutgoingMessage{
			Type:  "tool_error",
			ID:    msg.ID,
			Error: result.Error,
		}); err != nil {
			log.Printf("[daemon] send failed: %v", err)
		}
		return
	}

	exitCode := result.ExitCode
	if err := d.client.Send(relay.OutgoingMessage{
		Type:     "tool_done",
		ID:       msg.ID,
		ExitCode: &exitCode,
		Output:   result.Output,
	}); err != nil {
		log.Printf("[daemon] send failed: %v", err)
	}
}

func (d *Daemon) SendCapabilities() {
	if err := d.client.Send(relay.OutgoingMessage{
		Type:   "capabilities",
		Tools:  d.registry.Definitions(),
		Skills: d.skills,
	}); err != nil {
		log.Printf("[daemon] send failed: %v", err)
	}
}
