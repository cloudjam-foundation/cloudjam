package plugin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"codeberg.org/megakuul/cloudjam/internal/challenge"
	"codeberg.org/megakuul/cloudjam/internal/provider"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/api"
	"github.com/google/uuid"
)

type localProvider struct {
	score      float64
	scores     map[api.ScoreType]api.Score
	scoreItems map[string]api.SubmitScoreInput
	diagrams   map[string][]byte
	access     provider.AccessController
	assets     provider.AssetController
	resources  provider.ResourceController
}

func (p *localProvider) sendHTTP(ctx context.Context, in *api.SendHTTPInput) (*api.SendHTTPOutput, error) {
	if len(in.Body) > api.MaxHTTPBodySize {
		return nil, fmt.Errorf("http request bodies larger than %d bytes are not supported", api.MaxHTTPBodySize)
	}
	endpoint, err := url.Parse(in.URL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return nil, fmt.Errorf("unsupported url scheme %q", endpoint.Scheme)
	}
	if endpoint.Host == "" {
		return nil, fmt.Errorf("url has no host")
	}

	timeout := 10 * time.Second
	if in.TimeoutMillis > 0 {
		timeout = time.Duration(min(in.TimeoutMillis, (30*time.Second).Milliseconds())) * time.Millisecond
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, in.Method, endpoint.String(), bytes.NewReader(in.Body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	request.Header = http.Header(in.Headers).Clone()

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, api.MaxHTTPBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if len(body) > api.MaxHTTPBodySize {
		return nil, fmt.Errorf("http response bodies larger than %d bytes are not supported", api.MaxHTTPBodySize)
	}
	return &api.SendHTTPOutput{
		StatusCode: response.StatusCode,
		Headers:    map[string][]string(response.Header.Clone()),
		Body:       body,
	}, nil
}

func (p *localProvider) cancel(ctx context.Context, in *api.CancelInput) (*api.CancelOutput, error) {
	slog.Error(fmt.Sprintf("plugin cancelled %q: %q", in.Error, in.DetailError), "source", "plugin")
	return &api.CancelOutput{}, nil
}

func (p *localProvider) log(ctx context.Context, in *api.LogInput) (*api.LogOutput, error) {
	slog.Log(ctx, in.Severity, in.Message, "source", "plugin")
	return &api.LogOutput{}, nil
}

func (p *localProvider) createMeta(ctx context.Context, in *api.CreateMetaInput) (*api.CreateMetaOutput, error) {
	if err := challenge.ValidateDiagrams(in.Diagrams); err != nil {
		return nil, err
	}
	p.diagrams = in.Diagrams
	slog.Info("challenge metadata created", "title", in.Title)
	slog.Info(strings.Join(in.Descriptions, "\n\n"))
	for id, clue := range in.Clues {
		slog.Info(clue, "clue", id)
	}
	for id, asset := range in.Assets {
		slog.Info(asset, "asset", id)
	}
	for name, diagram := range in.Diagrams {
		slog.Info("infrastructure diagram added", "name", name, "size", len(diagram))
	}
	return &api.CreateMetaOutput{}, nil
}

func (p *localProvider) updateMeta(ctx context.Context, in *api.UpdateMetaInput) (*api.UpdateMetaOutput, error) {
	diagrams := make(map[string][]byte, len(p.diagrams)+len(in.AdditionalDiagrams))
	for name, diagram := range p.diagrams {
		diagrams[name] = diagram
	}
	for name, diagram := range in.AdditionalDiagrams {
		diagrams[name] = diagram
	}
	if err := challenge.ValidateDiagrams(diagrams); err != nil {
		return nil, err
	}
	p.diagrams = diagrams
	slog.Info("challenge metadata updated")
	slog.Info(strings.Join(in.AdditionalDescriptions, "\n\n"))
	for id, clue := range in.AdditionalClues {
		slog.Info(clue, "clue", id)
	}
	for id, asset := range in.AdditionalAssets {
		slog.Info(asset, "asset", id)
	}
	for name, diagram := range in.AdditionalDiagrams {
		slog.Info("infrastructure diagram added", "name", name, "size", len(diagram))
	}
	return &api.UpdateMetaOutput{}, nil
}

func (p *localProvider) readScore(ctx context.Context, in *api.ReadScoreInput) (*api.ReadScoreOutput, error) {
	output := &api.ReadScoreOutput{Score: p.score, Scores: p.scores}
	if in.Type != api.ScoreTypeUnspecified {
		output.Score = 0
		if score, ok := p.scores[in.Type]; ok {
			output.Score = score.Value
			output.Maximum = score.Maximum
		}
		return output, nil
	}
	for _, score := range p.scores {
		output.Maximum += score.Maximum
	}
	return output, nil
}

func (p *localProvider) updateScore(ctx context.Context, in *api.UpdateScoreInput) (*api.UpdateScoreOutput, error) {
	p.score += in.Increment
	slog.Info(fmt.Sprintf("%+g points — %s", in.Increment, in.Reason), "score", p.score)
	return &api.UpdateScoreOutput{}, nil
}

func (p *localProvider) registerScore(ctx context.Context, in *api.RegisterScoreInput) (*api.RegisterScoreOutput, error) {
	if in.Name == "" || in.Type == api.ScoreTypeUnspecified || math.IsNaN(in.Maximum) || math.IsInf(in.Maximum, 0) || in.Maximum <= 0 {
		return nil, fmt.Errorf("invalid score registration")
	}
	previous, exists := p.scoreItems[in.Name]
	if exists && previous.Type == in.Type && previous.Maximum == in.Maximum {
		return &api.RegisterScoreOutput{}, nil
	}
	if exists && previous.Score > in.Maximum {
		return nil, fmt.Errorf("maximum score cannot be lower than its current score")
	}
	if exists {
		total := p.scores[previous.Type]
		total.Value -= previous.Score
		total.Maximum -= previous.Maximum
		p.scores[previous.Type] = total
	}
	total := p.scores[in.Type]
	total.Value += previous.Score
	total.Maximum += in.Maximum
	p.scores[in.Type] = total
	previous.Name = in.Name
	previous.Type = in.Type
	previous.Maximum = in.Maximum
	p.scoreItems[in.Name] = previous
	return &api.RegisterScoreOutput{}, nil
}

func (p *localProvider) submitScore(ctx context.Context, in *api.SubmitScoreInput) (*api.SubmitScoreOutput, error) {
	if in.Name == "" || in.Type == api.ScoreTypeUnspecified || math.IsNaN(in.Score) || math.IsInf(in.Score, 0) || math.IsNaN(in.Maximum) || math.IsInf(in.Maximum, 0) || in.Maximum <= 0 || in.Score < 0 || in.Score > in.Maximum {
		return nil, fmt.Errorf("invalid score submission")
	}
	previous, exists := p.scoreItems[in.Name]
	if !in.Accumulate && exists && previous.Type == in.Type && previous.Score == in.Score && previous.Maximum == in.Maximum && previous.Reason == in.Reason {
		return &api.SubmitScoreOutput{}, nil
	}
	item := *in
	if in.Accumulate && exists && previous.Type == in.Type {
		item.Score += previous.Score
		item.Maximum += previous.Maximum
	}
	if exists {
		total := p.scores[previous.Type]
		total.Value -= previous.Score
		total.Maximum -= previous.Maximum
		p.scores[previous.Type] = total
	}
	total := p.scores[in.Type]
	total.Value += item.Score
	total.Maximum += item.Maximum
	p.scores[in.Type] = total
	delta := item.Score - previous.Score
	p.score += delta
	p.scoreItems[in.Name] = item
	slog.Info(fmt.Sprintf("%+g points — %s", delta, in.Name), "type", in.Type, "score", in.Score, "maximum", in.Maximum, "reason", in.Reason)
	return &api.SubmitScoreOutput{}, nil
}

func (p *localProvider) createAsset(ctx context.Context, in api.CreateAssetInput) (*api.CreateAssetOutput, error) {
	if len(in) > 50_000_000 {
		return nil, fmt.Errorf("assets larger then 50 MB are not supported")
	}
	name := uuid.NewString()
	url, err := p.assets.Create(ctx, name, bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	return &api.CreateAssetOutput{
		Name: name,
		URL:  url,
	}, nil
}

func (p *localProvider) updateAsset(ctx context.Context, in *api.UpdateAssetInput) (*api.UpdateAssetOutput, error) {
	url, err := p.assets.Update(ctx, in.OldName, in.NewName)
	if err != nil {
		return nil, err
	}
	return &api.UpdateAssetOutput{
		NewURL: url,
	}, nil
}

func (p *localProvider) createPermission(ctx context.Context, in *api.CreatePermissionInput) (*api.CreatePermissionOutput, error) {
	slog.Info("creating permission")
	err := p.access.CreatePermission(ctx, in.Permission)
	return &api.CreatePermissionOutput{}, err
}

func (p *localProvider) updatePermission(ctx context.Context, in *api.UpdatePermissionInput) (*api.UpdatePermissionOutput, error) {
	slog.Info("updating permission")
	err := p.access.UpdatePermission(ctx, in.Permission)
	return &api.UpdatePermissionOutput{}, err
}

func (p *localProvider) createGuardrail(ctx context.Context, in *api.CreateGuardrailInput) (*api.CreateGuardrailOutput, error) {
	slog.Info("creating guardrail")
	err := p.access.CreateGuardrail(ctx, in.Guardrail)
	return &api.CreateGuardrailOutput{}, err
}

func (p *localProvider) updateGuardrail(ctx context.Context, in *api.UpdateGuardrailInput) (*api.UpdateGuardrailOutput, error) {
	slog.Info("creating guardrail")
	err := p.access.UpdateGuardrail(ctx, in.Guardrail)
	return &api.UpdateGuardrailOutput{}, err
}

func (p *localProvider) createResource(ctx context.Context, in *api.CreateResourceInput) (*api.CreateResourceOutput, error) {
	slog.Info("creating resource", "type", in.Type)
	identifier, err := p.resources.Create(ctx, in.Type, in.Desired)
	if err != nil {
		return nil, err
	}
	slog.Info("created resource", "type", in.Type, "identifier", identifier)
	return &api.CreateResourceOutput{Identifier: identifier}, nil
}

func (p *localProvider) readResource(ctx context.Context, in *api.ReadResourceInput) (*api.ReadResourceOutput, error) {
	state, err := p.resources.Read(ctx, in.Type, in.Identifier)
	return &api.ReadResourceOutput{State: state}, err
}

func (p *localProvider) updateResource(ctx context.Context, in *api.UpdateResourceInput) (*api.UpdateResourceOutput, error) {
	slog.Info("updating resource", "type", in.Type, "identifier", in.Identifier)
	return &api.UpdateResourceOutput{}, p.resources.Update(ctx, in.Type, in.Identifier, in.Patch)
}

func (p *localProvider) deleteResource(ctx context.Context, in *api.DeleteResourceInput) (*api.DeleteResourceOutput, error) {
	slog.Info("deleting resource", "type", in.Type, "identifier", in.Identifier)
	return &api.DeleteResourceOutput{}, p.resources.Delete(ctx, in.Type, in.Identifier)
}

func (p *localProvider) listResource(ctx context.Context, in *api.ListResourceInput) (*api.ListResourceOutput, error) {
	resources, err := p.resources.List(ctx, in.Type, in.ResourceModel)
	return &api.ListResourceOutput{Resources: resources}, err
}
