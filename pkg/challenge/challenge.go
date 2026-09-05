//go:build wasip1

// Package challenge is the plugin sdk.
package challenge

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge/api"
)

// DEVELOPER NOTICE: this code is specifically designed for the cloudjam plugin lifetime.
// It contains goroutine leaks, functions that disregard context and other usually bad practices.
// Please do not program normal go code like this. This is an exception that works due to the reduced scope and the short lifetime of plugins.

// Event defines a custom update in the scenario.
// Example: If the user reaches 500 points, start shutting down EC2 instances and add a check that awards points if enough are available.
type Event struct {
	Every   time.Duration // throttles duration (zero evaluates on every loop iteration (which is defined by Challenge.Interval))
	Event   func(ctx context.Context, s *Scenario) error
	Trigger func() (bool, error)

	last time.Time
}

// Check defines an entity that rewards points to the user if he solves part of the challenge.
// Example: If the user enabled
type Check struct {
	Points  float64              // points awarded if the check is positive (can also be negative).
	Every   time.Duration        // throttles duration (zero evaluates on every loop iteration (which is defined by Challenge.Interval))
	Repeat  bool                 // should the points be awarded on every evaluation?
	Live    bool                 // typed checks continue to track their result after passing.
	Type    ScoreType            // optional score category; typed checks submit their current score and maximum.
	Reason  string               // explanation shown with the result after the challenge ends.
	Trigger func() (bool, error) // condition that tells if the check is successful.

	last       time.Time
	lastResult *bool
	retired    bool
}

const (
	MaxDiagramSize  = api.MaxDiagramSize
	MaxDiagramCount = api.MaxDiagramCount
	MaxHTTPBodySize = api.MaxHTTPBodySize
)

type ScoreType = api.ScoreType

const (
	ScoreTypeUnspecified = api.ScoreTypeUnspecified
	ScoreTypeDesign      = api.ScoreTypeDesign
	ScoreTypeOperational = api.ScoreTypeOperational
)

type HTTPRequest = api.SendHTTPInput
type HTTPResponse = api.SendHTTPOutput

// NewDesignCheck creates a one-shot configuration check in the design score category.
func NewDesignCheck(points float64, reason string, trigger func() (bool, error)) Check {
	return NewScoreCheck(ScoreTypeDesign, points, false, reason, trigger)
}

// NewOperationalCheck creates a live check whose points follow its current result.
func NewOperationalCheck(points float64, reason string, trigger func() (bool, error)) Check {
	return NewScoreCheck(ScoreTypeOperational, points, true, reason, trigger)
}

// NewScoreCheck creates a typed check for a score category.
func NewScoreCheck(scoreType ScoreType, points float64, live bool, reason string, trigger func() (bool, error)) Check {
	return Check{Points: points, Live: live, Type: scoreType, Reason: reason, Trigger: trigger}
}

// NewHTTPRequest creates an HTTP request with an initialized header map and a ten second timeout.
func NewHTTPRequest(method, url string, body []byte) HTTPRequest {
	return HTTPRequest{
		Method:        method,
		URL:           url,
		Headers:       map[string][]string{},
		Body:          body,
		TimeoutMillis: 10_000,
	}
}

// Scenario is the main challenge subject, per challenge there should only be one scenario.
type Scenario struct {
	interval time.Duration // round check speed, defaults to 10s

	bootstrap func(s *Scenario) error

	// init params exist to collect all metadata in the initial builder
	// and then perform one single CreateMeta() call, after init is true, updates are incrementally.
	initTitle        string
	initDescriptions []string
	initAssets       map[string]string
	initDiagrams     map[string][]byte
	initClues        map[string]string
	initCluePrices   map[string]float64
	initPermission   string
	initGuardrail    string
	init             atomic.Bool

	checksLock sync.RWMutex
	// by pointer: evaluateChecks writes last and retired back onto the check it
	// just ran, and a map of values would hand it a copy to throw away.
	checks map[string]*Check

	eventsLock   sync.RWMutex // also used for activeEvents
	events       map[string]Event
	activeEvents map[string]context.CancelFunc
}

// New creates the scenario for your plugin.
func New(title string, interval time.Duration, boostrap func(s *Scenario) error) *Scenario {
	return &Scenario{
		bootstrap:      boostrap,
		initTitle:      title,
		interval:       max(10*time.Second, interval),
		init:           atomic.Bool{},
		initAssets:     map[string]string{},
		initDiagrams:   map[string][]byte{},
		initClues:      map[string]string{},
		initCluePrices: map[string]float64{},
		checks:         map[string]*Check{},
		events:         map[string]Event{},
		activeEvents:   map[string]context.CancelFunc{},
	}
}

// SendHTTP asks the challenge host to send an HTTP request and returns its response.
func (c *Scenario) SendHTTP(request HTTPRequest) (HTTPResponse, error) {
	return api.SendHTTP(request)
}

func (c *Scenario) ReportProgress(message string) error {
	_, err := api.UpdateScore(api.UpdateScoreInput{Reason: message})
	return err
}

// AddDiagram adds a small JPEG infrastructure diagram to the challenge metadata.
func (c *Scenario) AddDiagram(name string, diagram []byte) *Scenario {
	if !c.init.Load() {
		c.initDiagrams[name] = diagram
	} else {
		if _, err := api.UpdateMeta(api.UpdateMetaInput{
			AdditionalDiagrams: map[string][]byte{name: diagram},
		}); err != nil {
			_, err := api.Cancel(api.CancelInput{
				Error:       "diagram is defect",
				DetailError: fmt.Sprintf("failed to update diagram metadata '%s': %v", name, err),
			})
			if err != nil {
				slog.Error(fmt.Sprintf("failed to cancel: %v", err))
			}
		}
	}
	return c
}

// AddAsset uploads the provided asset with a specified name to the providers asset storage.
// The user will be able to retrieve this asset.
func (c *Scenario) AddAsset(name string, asset []byte) *Scenario {
	assetMeta, err := api.CreateAsset(asset)
	if err != nil {
		_, err := api.Cancel(api.CancelInput{
			Error:       "asset creation mechanism is defect",
			DetailError: fmt.Sprintf("failed to create asset '%s': %v", name, err),
		})
		if err != nil {
			slog.Error(fmt.Sprintf("failed to cancel: %v", err))
		}
		return c
	}
	newAssetMeta, err := api.UpdateAsset(api.UpdateAssetInput{
		OldName: assetMeta.Name,
		NewName: name,
	})
	if err != nil {
		_, err := api.Cancel(api.CancelInput{
			Error:       "asset update mechanism is defect",
			DetailError: fmt.Sprintf("failed to update asset '%s': %v", name, err),
		})
		if err != nil {
			slog.Error(fmt.Sprintf("failed to cancel: %v", err))
		}
		return c
	}
	if !c.init.Load() {
		c.initAssets[newAssetMeta.NewURL] = name
	} else {
		if _, err := api.UpdateMeta(api.UpdateMetaInput{
			AdditionalAssets: map[string]string{
				newAssetMeta.NewURL: name,
			},
		}); err != nil {
			_, err := api.Cancel(api.CancelInput{
				Error:       "metadata update mechanism is defect",
				DetailError: fmt.Sprintf("failed to update asset metadata '%s': %v", name, err),
			})
			if err != nil {
				slog.Error(fmt.Sprintf("failed to cancel: %v", err))
			}
		}
	}
	return c
}

// AddDescription adds a description text to the scenario.
func (c *Scenario) AddDescription(description string) *Scenario {
	if !c.init.Load() {
		c.initDescriptions = append(c.initDescriptions, description)
	} else {
		if _, err := api.UpdateMeta(api.UpdateMetaInput{
			AdditionalDescriptions: []string{description},
		}); err != nil {
			_, err := api.Cancel(api.CancelInput{
				Error:       "description is defect",
				DetailError: fmt.Sprintf("failed to update description metadata: %v", err),
			})
			if err != nil {
				slog.Error(fmt.Sprintf("failed to cancel: %v", err))
			}
		}
	}
	return c
}

// AddClue adds a clue to the scenario (hint is what the user always sees, secret is what he gets when uncovering the hint).
func (c *Scenario) AddClue(hint, secret string, price float64) *Scenario {
	if !c.init.Load() {
		c.initClues[hint] = secret
		c.initCluePrices[hint] = price
	} else {
		if _, err := api.UpdateMeta(api.UpdateMetaInput{
			AdditionalClues: map[string]string{
				hint: secret,
			},
			AdditionalCluePrices: map[string]float64{
				hint: price,
			},
		}); err != nil {
			_, err := api.Cancel(api.CancelInput{
				Error:       "clue system is defect",
				DetailError: fmt.Sprintf("failed to add clue '%s': %v", hint, err),
			})
			if err != nil {
				slog.Error(fmt.Sprintf("failed to cancel: %v", err))
			}
		}
	}
	return c
}

// SetPermission configures the users initial permissions for this challenge.
// May be escalated by user as process of the challenge (for absolute restriction use SetGuardrail).
func (c *Scenario) SetPermission(doc fmt.Stringer) *Scenario {
	if !c.init.Load() {
		c.initPermission = doc.String()
	} else {
		if _, err := api.UpdatePermission(api.UpdatePermissionInput{
			Permission: doc.String(),
		}); err != nil {
			_, err := api.Cancel(api.CancelInput{
				Error:       "permission system is defect",
				DetailError: fmt.Sprintf("failed to set permissions: %v", err),
			})
			if err != nil {
				slog.Error(fmt.Sprintf("failed to cancel: %v", err))
			}
		}
	}
	return c
}

// SetGuardrail configures the users definite permissions guardrail for this challenge.
func (c *Scenario) SetGuardrail(doc fmt.Stringer) *Scenario {
	if !c.init.Load() {
		c.initGuardrail = doc.String()
	} else {
		if _, err := api.UpdateGuardrail(api.UpdateGuardrailInput{
			Guardrail: doc.String(),
		}); err != nil {
			_, err := api.Cancel(api.CancelInput{
				Error:       "guardrail system is defect",
				DetailError: fmt.Sprintf("failed to set guardrails: %v", err),
			})
			if err != nil {
				slog.Error(fmt.Sprintf("failed to cancel: %v", err))
			}
		}
	}
	return c
}

// AddCheck adds a check to the next cycle.
func (c *Scenario) AddCheck(name string, check Check) *Scenario {
	if check.Type != ScoreTypeUnspecified {
		if _, err := api.RegisterScore(api.RegisterScoreInput{
			Name:    name,
			Type:    check.Type,
			Maximum: check.Points,
		}); err != nil {
			_, err := api.Cancel(api.CancelInput{
				Error:       "score metadata is defect",
				DetailError: fmt.Sprintf("failed to register score '%s': %v", name, err),
			})
			if err != nil {
				slog.Error(fmt.Sprintf("failed to cancel: %v", err))
			}
			return c
		}
	}
	c.checksLock.Lock()
	defer c.checksLock.Unlock()
	c.checks[name] = &check
	return c
}

// RemoveCheck removes the check in the next cycle.
func (c *Scenario) RemoveCheck(name string) *Scenario {
	c.checksLock.Lock()
	defer c.checksLock.Unlock()
	delete(c.checks, name)
	return c
}

// AddEvent adds a event to the next cycle.
func (c *Scenario) AddEvent(name string, event Event) *Scenario {
	c.eventsLock.Lock()
	defer c.eventsLock.Unlock()
	c.events[name] = event
	return c
}

// RemoveEvent removes the event in the next cycle.
func (c *Scenario) RemoveEvent(name string, event Event) *Scenario {
	c.eventsLock.Lock()
	defer c.eventsLock.Unlock()
	delete(c.events, name)
	if cancel, ok := c.activeEvents[name]; ok {
		cancel()
	}
	delete(c.activeEvents, name)
	return c
}

// Starts the specified scenario inside your wasm plugin.
// Start may only be called once (otherwise it will panic).
func (c *Scenario) Start() {
	if !c.init.CompareAndSwap(false, true) {
		panic("scenario cannot be started twice!")
	}
	if _, err := api.CreateMeta(api.CreateMetaInput{
		Title:        c.initTitle,
		Descriptions: c.initDescriptions,
		Clues:        c.initClues,
		CluePrices:   c.initCluePrices,
		Assets:       c.initAssets,
		Diagrams:     c.initDiagrams,
		Ready:        false,
	}); err != nil {
		_, err := api.Cancel(api.CancelInput{
			Error:       "metadata is defect",
			DetailError: fmt.Sprintf("failed to create metadata: %v", err),
		})
		if err != nil {
			slog.Error(fmt.Sprintf("failed to cancel: %v", err))
		}
		return
	}

	if c.initPermission == "" || c.initGuardrail == "" {
		panic("no permission specified for this scenario!")
	}
	if _, err := api.CreatePermission(api.CreatePermissionInput{
		Permission: c.initPermission,
	}); err != nil {
		_, err := api.Cancel(api.CancelInput{
			Error:       "permissions are defect",
			DetailError: fmt.Sprintf("failed to create permissions: %v", err),
		})
		if err != nil {
			slog.Error(fmt.Sprintf("failed to cancel: %v", err))
		}
		return
	}
	if _, err := api.CreateGuardrail(api.CreateGuardrailInput{
		Guardrail: c.initGuardrail,
	}); err != nil {
		_, err := api.Cancel(api.CancelInput{
			Error:       "guardrails are defect",
			DetailError: fmt.Sprintf("failed to create guardrails: %v", err),
		})
		if err != nil {
			slog.Error(fmt.Sprintf("failed to cancel: %v", err))
		}
		return
	}

	if err := c.bootstrap(c); err != nil {
		_, err := api.Cancel(api.CancelInput{
			Error:       "challenge infrastructure is defect",
			DetailError: fmt.Sprintf("failed to bootstrap: %v", err),
		})
		if err != nil {
			slog.Error(fmt.Sprintf("failed to cancel: %v", err))
		}
		return
	}
	if _, err := api.UpdateMeta(api.UpdateMetaInput{
		Ready: new(true),
	}); err != nil {
		_, err := api.Cancel(api.CancelInput{
			Error:       "challenge metadata is defect",
			DetailError: fmt.Sprintf("failed to set challenge to ready: %v", err),
		})
		if err != nil {
			slog.Error(fmt.Sprintf("failed to cancel: %v", err))
		}
		return
	}

	for {
		c.evaluateChecks()
		c.orchestrateEvents()

		time.Sleep(c.interval)
	}
}

// orchestrateEvents checks all event triggers and starts / stops them accordingly.
func (c *Scenario) orchestrateEvents() {
	c.eventsLock.Lock()
	defer c.eventsLock.Unlock()
	for name, event := range c.events {
		active, err := event.Trigger()
		if err != nil {
			slog.Error(fmt.Sprintf("event start trigger %q: %v", name, err))
			continue
		}
		if cancel, ok := c.activeEvents[name]; ok {
			if !active {
				cancel()
			}
		} else {
			if active {
				ctx, cancel := context.WithCancel(context.Background())
				c.activeEvents[name] = cancel
				go func() {
					if err := event.Event(ctx, c); err != nil {
						slog.Error(fmt.Sprintf("event %q: %v", name, err))
					}
				}()
			}
		}
	}
}

// evaluateChecks evaluates all checks and awards points accordingly.
func (c *Scenario) evaluateChecks() {
	c.checksLock.Lock()
	defer c.checksLock.Unlock()
	now := time.Now()
	for name, check := range c.checks {
		if check.retired || check.Trigger == nil {
			continue
		}
		if !check.last.IsZero() && now.Sub(check.last) < check.Every {
			continue
		}
		check.last = now

		passed, err := check.Trigger()
		if err != nil {
			if check.Type != ScoreTypeUnspecified {
				reason := err.Error()
				if check.Reason != "" {
					reason = fmt.Sprintf("%s: %v", check.Reason, err)
				}
				if _, submitErr := api.SubmitScore(api.SubmitScoreInput{
					Name:    name,
					Type:    check.Type,
					Maximum: check.Points,
					Reason:  reason,
				}); submitErr != nil {
					slog.Error(fmt.Sprintf("check %q: submit failure: %v", name, submitErr))
				} else {
					check.lastResult = nil
				}
			}
			slog.Error(fmt.Sprintf("check %q: %v", name, err))
			continue
		}
		if check.Type != ScoreTypeUnspecified {
			if check.lastResult != nil && *check.lastResult == passed {
				continue
			}
			score := 0.0
			if passed {
				score = check.Points
			}
			if _, err := api.SubmitScore(api.SubmitScoreInput{
				Name:    name,
				Type:    check.Type,
				Score:   score,
				Maximum: check.Points,
				Reason:  check.Reason,
			}); err != nil {
				slog.Error(fmt.Sprintf("check %q: submit: %v", name, err))
				continue
			}
			check.lastResult = new(passed)
			if passed && !check.Live {
				check.retired = true
			}
			continue
		}
		if !passed {
			continue
		}
		if _, err := api.UpdateScore(api.UpdateScoreInput{
			Reason:    name,
			Increment: check.Points,
		}); err != nil {
			// Leave it live so the award is attempted again next round.
			slog.Error(fmt.Sprintf("check %q: award: %v", name, err))
			continue
		}
		if !check.Repeat {
			check.retired = true
		}
	}
}
