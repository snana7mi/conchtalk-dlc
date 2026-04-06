package daemon

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/snana7mi/conchtalk-dlc/acp"
	"github.com/snana7mi/conchtalk-dlc/relay"
	"github.com/snana7mi/conchtalk-dlc/skills"
	"github.com/snana7mi/conchtalk-dlc/tools"
)

type Daemon struct {
	client     *relay.Client
	registry   *tools.Registry
	skills     []relay.SkillDefinition
	sem        chan struct{}
	acpManager *acp.Manager
}

// HandleMessage implements relay.MessageHandler.
func (d *Daemon) HandleMessage(msg relay.IncomingMessage) {
	switch msg.Type {
	case "tool_call":
		go d.executeTool(msg)
	case "status":
		log.Printf("[daemon] client status: %s", msg.Client)
	case "acp_start":
		go d.handleACPStart(msg)
	case "acp_data":
		go d.handleACPData(msg)
	case "acp_close":
		d.acpManager.Close(msg.SessionID)
	default:
		log.Printf("[daemon] unknown message type: %s", msg.Type)
	}
}

func Run(token, server string) error {
	d := &Daemon{
		registry:   tools.NewRegistry(),
		skills:     skills.Load(),
		sem:        make(chan struct{}, 16),
		acpManager: acp.NewManager(),
	}
	defer d.acpManager.CloseAll()

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
	agents := acp.DetectAgents()
	if err := d.client.Send(relay.OutgoingMessage{
		Type:   "capabilities",
		Tools:  d.registry.Definitions(),
		Skills: d.skills,
		Agents: agents,
	}); err != nil {
		log.Printf("[daemon] send failed: %v", err)
	}
}

func (d *Daemon) handleACPStart(msg relay.IncomingMessage) {
	err := d.acpManager.Start(msg.SessionID, msg.Command, msg.Cwd, func(out relay.OutgoingMessage) error {
		return d.client.Send(out)
	})
	if err != nil {
		_ = d.client.Send(relay.OutgoingMessage{
			Type:      "acp_error",
			SessionID: msg.SessionID,
			Error:     err.Error(),
		})
		return
	}
	_ = d.client.Send(relay.OutgoingMessage{
		Type:      "acp_started",
		SessionID: msg.SessionID,
	})
}

func (d *Daemon) handleACPData(msg relay.IncomingMessage) {
	if err := d.acpManager.Send(msg.SessionID, msg.Data); err != nil {
		_ = d.client.Send(relay.OutgoingMessage{
			Type:      "acp_error",
			SessionID: msg.SessionID,
			Error:     err.Error(),
		})
	}
}
