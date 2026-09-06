package challenge

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math"

	"codeberg.org/megakuul/cloudjam/internal/oltp"
	"codeberg.org/megakuul/cloudjam/internal/provider"
	"codeberg.org/megakuul/cloudjam/pkg/api/v1/cloud"
	"codeberg.org/megakuul/cloudjam/pkg/api/v1/play"
	"codeberg.org/megakuul/cloudjam/pkg/challenge/api"
	extism "github.com/extism/go-sdk"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/megakuul/dynamitedb"
	"github.com/megakuul/lake"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Challenge is the actual statemachine that runs the challenge wasm plugin and implements the API.
type Challenge struct {
	logger *slog.Logger

	definition *oltp.Definition
	challenge  *oltp.Challenge
	team       *oltp.Team

	pluginCache *Cache

	oltp *dynamitedb.Bucket
	olap *lake.Bucket

	access    provider.AccessController
	assets    provider.AssetController
	resources provider.ResourceController

	cancelFunc context.CancelFunc
}

func New(
	logger *slog.Logger,
	definition *oltp.Definition,
	challenge *oltp.Challenge,
	team *oltp.Team,
	pluginCache *Cache,
	oltp *dynamitedb.Bucket,
	olap *lake.Bucket,
	access provider.AccessController,
	assets provider.AssetController,
	resource provider.ResourceController,
) *Challenge {
	return &Challenge{
		logger:      logger,
		definition:  definition,
		challenge:   challenge,
		team:        team,
		pluginCache: pluginCache,
		oltp:        oltp,
		olap:        olap,
		access:      access,
		assets:      assets,
		resources:   resource,
	}
}

// Start launches the challenge plugin and runs until the context expires.
func (c *Challenge) Start(ctx context.Context) (err error) {
	report := func(err error) {
		c.logger.Warn(err.Error())
	}
	challengeCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	c.cancelFunc = cancel

	plugin, err := c.pluginCache.Load(ctx, string(c.definition.Hash.Value()), func(ctx context.Context) (extism.Wasm, error) {
		definitionBinary, err := dynamitedb.Get(ctx, c.oltp, &oltp.DefinitionBinary{
			ProviderID:   dynamitedb.Key(c.definition.ProviderID.Value()),
			DefinitionID: dynamitedb.Key(c.definition.DefinitionID.Value()),
			Scope:        dynamitedb.Eq(c.definition.Scope.Value()),
		})
		if err != nil {
			return nil, err
		}
		wasmData := extism.WasmData{}
		switch cloud.CompressionMode(definitionBinary.Compression.Value()) {
		case cloud.CompressionMode_Zstd:
			wasmData.Data, err = zstd.DecodeTo(nil, definitionBinary.WASM.Value())
			if err != nil {
				return nil, fmt.Errorf("failed to zstd decode plugin: %w", err)
			}
		default:
			return nil, fmt.Errorf("unknown compression algorithm in challenge definition (%d)", definitionBinary.Compression.Value())
		}
		return wasmData, nil
	},
		RegisterInOutHost(api.CancelName, c.cancel, report),
		RegisterInOutHost(api.LogName, c.log, report),
		RegisterInOutHost(api.CreateMetaName, c.createMeta, report),
		RegisterInOutHost(api.UpdateMetaName, c.updateMeta, report),
		RegisterInOutHost(api.ReadScoreName, c.readScore, report),
		RegisterInOutHost(api.UpdateScoreName, c.updateScore, report),
		RegisterInOutHost(api.RegisterScoreName, c.registerScore, report),
		RegisterInOutHost(api.SubmitScoreName, c.submitScore, report),
		RegisterInOutHost(api.SendHTTPName, c.sendHTTP, report),
		RegisterOutHost(api.CreateAssetName, c.createAsset, report),
		RegisterInOutHost(api.UpdateAssetName, c.updateAsset, report),
		RegisterInOutHost(api.CreatePermissionName, c.createPermission, report),
		RegisterInOutHost(api.UpdatePermissionName, c.updatePermission, report),
		RegisterInOutHost(api.CreateGuardrailName, c.createGuardrail, report),
		RegisterInOutHost(api.UpdateGuardrailName, c.updateGuardrail, report),
		RegisterInOutHost(api.CreateResourceName, c.createResource, report),
		RegisterInOutHost(api.ReadResourceName, c.readResource, report),
		RegisterInOutHost(api.UpdateResourceName, c.updateResource, report),
		RegisterInOutHost(api.DeleteResourceName, c.deleteResource, report),
		RegisterInOutHost(api.ListResourceName, c.listResource, report),
	)
	if err != nil {
		return fmt.Errorf("loading plugin: %v", err)
	}
	defer plugin.Close(ctx)

	_, _, err = plugin.CallWithContext(challengeCtx, "_start", nil)
	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("starting plugin: %v", err)
	}
	return nil
}

func (c *Challenge) cancel(ctx context.Context, input *api.CancelInput) (*api.CancelOutput, error) {
	c.cancelFunc()
	err := dynamitedb.Update(ctx, c.oltp, &oltp.Challenge{
		GameID:      dynamitedb.Key(c.challenge.GameID.Value()),
		ChallengeID: dynamitedb.Key(c.challenge.ChallengeID.Value()),
		Error:       dynamitedb.Set(input.Error),
	})
	c.logger.Log(ctx, slog.LevelError, fmt.Sprintf("challenge cancelled: %q", input.DetailError))
	return &api.CancelOutput{}, err
}

func (c *Challenge) log(ctx context.Context, input *api.LogInput) (*api.LogOutput, error) {
	c.logger.Log(ctx, input.Severity, input.Message)
	return &api.LogOutput{}, nil
}

func (c *Challenge) createMeta(ctx context.Context, input *api.CreateMetaInput) (*api.CreateMetaOutput, error) {
	if err := ValidateDiagrams(input.Diagrams); err != nil {
		return nil, err
	}
	err := dynamitedb.Update(ctx, c.oltp, &oltp.Challenge{
		GameID:      dynamitedb.Key(c.challenge.GameID.Value()),
		ChallengeID: dynamitedb.Key(c.challenge.ChallengeID.Value()),
		Title:       dynamitedb.Set(input.Title),
		Description: dynamitedb.Set(input.Descriptions),
		Clues:       dynamitedb.Set(input.Clues),
		CluePrices:  dynamitedb.Emplace(input.CluePrices),
		Assets:      dynamitedb.Set(input.Assets),
		Diagrams:    dynamitedb.Set(input.Diagrams),
		Ready:       dynamitedb.Set(input.Ready),
	})
	return &api.CreateMetaOutput{}, err
}

func (c *Challenge) updateMeta(ctx context.Context, input *api.UpdateMetaInput) (*api.UpdateMetaOutput, error) {
	if len(input.AdditionalDiagrams) > 0 {
		challenge, err := dynamitedb.Get(ctx, c.oltp, &oltp.Challenge{
			GameID:      dynamitedb.Key(c.challenge.GameID.Value()),
			ChallengeID: dynamitedb.Key(c.challenge.ChallengeID.Value()),
		})
		if err != nil {
			return nil, err
		}
		diagrams := challenge.Diagrams.Value()
		if diagrams == nil {
			diagrams = map[string][]byte{}
		}
		for name, diagram := range input.AdditionalDiagrams {
			diagrams[name] = diagram
		}
		if err := ValidateDiagrams(diagrams); err != nil {
			return nil, err
		}
	}
	var ready dynamitedb.DataField[bool]
	if input.Ready != nil {
		ready = dynamitedb.Set(*input.Ready)
	}
	err := dynamitedb.Update(ctx, c.oltp, &oltp.Challenge{
		GameID:      dynamitedb.Key(c.challenge.GameID.Value()),
		ChallengeID: dynamitedb.Key(c.challenge.ChallengeID.Value()),
		Description: dynamitedb.Append(input.AdditionalDescriptions...),
		Clues:       dynamitedb.Emplace(input.AdditionalClues),
		CluePrices:  dynamitedb.Emplace(input.AdditionalCluePrices),
		Assets:      dynamitedb.Emplace(input.AdditionalAssets),
		Diagrams:    dynamitedb.Emplace(input.AdditionalDiagrams),
		Ready:       ready,
	})
	return &api.UpdateMetaOutput{}, err
}

func (c *Challenge) readScore(ctx context.Context, input *api.ReadScoreInput) (*api.ReadScoreOutput, error) {
	team, err := dynamitedb.Get(ctx, c.oltp, &oltp.Team{
		GameID: dynamitedb.Key(c.team.GameID.Value()),
		TeamID: dynamitedb.Key(c.team.TeamID.Value()),
	})
	if err != nil {
		return nil, err
	}
	challenge, err := dynamitedb.Get(ctx, c.oltp, &oltp.Challenge{
		GameID:      dynamitedb.Key(c.challenge.GameID.Value()),
		ChallengeID: dynamitedb.Key(c.challenge.ChallengeID.Value()),
	})
	if err != nil {
		return nil, err
	}
	scores := make(map[api.ScoreType]api.Score, len(challenge.Scores.Value()))
	for scoreType, score := range challenge.Scores.Value() {
		if score == nil {
			continue
		}
		scores[scoreType] = api.Score{Value: score.Value, Maximum: score.Maximum}
	}
	output := &api.ReadScoreOutput{Score: team.Score.Value(), Scores: scores}
	if input.Type != api.ScoreTypeUnspecified {
		output.Score = 0
		if score, ok := scores[input.Type]; ok {
			output.Score = score.Value
			output.Maximum = score.Maximum
		}
		return output, nil
	}
	for _, score := range scores {
		output.Maximum += score.Maximum
	}
	return output, nil
}

func (c *Challenge) updateScore(ctx context.Context, input *api.UpdateScoreInput) (*api.UpdateScoreOutput, error) {
	err := dynamitedb.Update(ctx, c.oltp, &oltp.Team{
		GameID: dynamitedb.Key(c.team.GameID.Value()),
		TeamID: dynamitedb.Key(c.team.TeamID.Value()),
		Score:  dynamitedb.Increment(input.Increment),
	})
	if err != nil {
		return nil, err
	}
	err = dynamitedb.Update(ctx, c.oltp, &oltp.Challenge{
		GameID:      dynamitedb.Key(c.challenge.GameID.Value()),
		ChallengeID: dynamitedb.Key(c.challenge.ChallengeID.Value()),
		ScoreEvents: dynamitedb.Append(&play.ScoreEvent{
			Timestamp: timestamppb.Now(),
			Text:      input.Reason,
			Change:    input.Increment,
		}),
	})
	return &api.UpdateScoreOutput{}, err
}

func (c *Challenge) registerScore(ctx context.Context, input *api.RegisterScoreInput) (*api.RegisterScoreOutput, error) {
	if input.Name == "" || input.Type == api.ScoreTypeUnspecified {
		return nil, fmt.Errorf("score name and type cannot be empty")
	}
	if math.IsNaN(input.Maximum) || math.IsInf(input.Maximum, 0) || input.Maximum <= 0 {
		return nil, fmt.Errorf("maximum score must be positive and finite")
	}

	challenge, err := dynamitedb.Get(ctx, c.oltp, &oltp.Challenge{
		GameID:      dynamitedb.Key(c.challenge.GameID.Value()),
		ChallengeID: dynamitedb.Key(c.challenge.ChallengeID.Value()),
	})
	if err != nil {
		return nil, err
	}
	previous := challenge.ScoreItems.Value()[input.Name]
	if previous != nil && previous.Type == input.Type && previous.Maximum == input.Maximum {
		return &api.RegisterScoreOutput{}, nil
	}
	if previous != nil && previous.Score > input.Maximum {
		return nil, fmt.Errorf("maximum score cannot be lower than its current score")
	}

	scores := challenge.Scores.Value()
	updates := map[play.ScoreType]*play.Score{}
	if previous != nil {
		previousTotal := scores[previous.Type]
		if previousTotal == nil {
			previousTotal = &play.Score{}
		}
		updates[previous.Type] = &play.Score{
			Type:    previous.Type,
			Value:   previousTotal.Value - previous.Score,
			Maximum: previousTotal.Maximum - previous.Maximum,
		}
	}
	current := scores[input.Type]
	if updated, ok := updates[input.Type]; ok {
		current = updated
	}
	if current == nil {
		current = &play.Score{}
	}
	item := &oltp.ScoreItem{Type: input.Type, Maximum: input.Maximum}
	if previous != nil {
		item.Score = previous.Score
		item.Reason = previous.Reason
	}
	updates[input.Type] = &play.Score{
		Type:    input.Type,
		Value:   current.Value + item.Score,
		Maximum: current.Maximum + input.Maximum,
	}
	err = dynamitedb.Update(ctx, c.oltp, &oltp.Challenge{
		GameID:      dynamitedb.Key(c.challenge.GameID.Value()),
		ChallengeID: dynamitedb.Key(c.challenge.ChallengeID.Value()),
		Scores:      dynamitedb.Emplace(updates),
		ScoreItems:  dynamitedb.Emplace(map[string]*oltp.ScoreItem{input.Name: item}),
	})
	return &api.RegisterScoreOutput{}, err
}

func (c *Challenge) submitScore(ctx context.Context, input *api.SubmitScoreInput) (*api.SubmitScoreOutput, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("score name cannot be empty")
	}
	if input.Type == api.ScoreTypeUnspecified {
		return nil, fmt.Errorf("score type cannot be empty")
	}
	if math.IsNaN(input.Score) || math.IsInf(input.Score, 0) || math.IsNaN(input.Maximum) || math.IsInf(input.Maximum, 0) {
		return nil, fmt.Errorf("score values must be finite")
	}
	if input.Maximum <= 0 || input.Score < 0 || input.Score > input.Maximum {
		return nil, fmt.Errorf("score must be between zero and its positive maximum")
	}

	challenge, err := dynamitedb.Get(ctx, c.oltp, &oltp.Challenge{
		GameID:      dynamitedb.Key(c.challenge.GameID.Value()),
		ChallengeID: dynamitedb.Key(c.challenge.ChallengeID.Value()),
	})
	if err != nil {
		return nil, err
	}
	items := challenge.ScoreItems.Value()
	previous := items[input.Name]
	if !input.Accumulate && previous != nil && previous.Type == input.Type && previous.Score == input.Score && previous.Maximum == input.Maximum && previous.Reason == input.Reason {
		return &api.SubmitScoreOutput{}, nil
	}
	score, maximum := input.Score, input.Maximum
	if input.Accumulate && previous != nil && previous.Type == input.Type {
		score += previous.Score
		maximum += previous.Maximum
	}

	scores := challenge.Scores.Value()
	updates := map[play.ScoreType]*play.Score{}
	if previous != nil {
		previousTotal := scores[previous.Type]
		if previousTotal == nil {
			previousTotal = &play.Score{}
		}
		updates[previous.Type] = &play.Score{
			Type:    previous.Type,
			Value:   previousTotal.Value - previous.Score,
			Maximum: previousTotal.Maximum - previous.Maximum,
		}
	}
	current := scores[input.Type]
	if updated, ok := updates[input.Type]; ok {
		current = updated
	}
	if current == nil {
		current = &play.Score{}
	}
	updates[input.Type] = &play.Score{
		Type:    input.Type,
		Value:   current.Value + score,
		Maximum: current.Maximum + maximum,
	}

	delta := score
	if previous != nil {
		delta -= previous.Score
	}
	if delta != 0 {
		if err := dynamitedb.Update(ctx, c.oltp, &oltp.Team{
			GameID: dynamitedb.Key(c.team.GameID.Value()),
			TeamID: dynamitedb.Key(c.team.TeamID.Value()),
			Score:  dynamitedb.Increment(delta),
		}); err != nil {
			return nil, err
		}
	}
	err = dynamitedb.Update(ctx, c.oltp, &oltp.Challenge{
		GameID:      dynamitedb.Key(c.challenge.GameID.Value()),
		ChallengeID: dynamitedb.Key(c.challenge.ChallengeID.Value()),
		Scores:      dynamitedb.Emplace(updates),
		ScoreItems: dynamitedb.Emplace(map[string]*oltp.ScoreItem{
			input.Name: {
				Type:    input.Type,
				Score:   score,
				Maximum: maximum,
				Reason:  input.Reason,
			},
		}),
		ScoreEvents: dynamitedb.Append(&play.ScoreEvent{
			Timestamp: timestamppb.Now(),
			Text:      input.Name,
			Change:    delta,
			Type:      input.Type,
			Score:     input.Score,
			Maximum:   input.Maximum,
			Reason:    input.Reason,
		}),
	})
	return &api.SubmitScoreOutput{}, err
}

func (c *Challenge) createAsset(ctx context.Context, input api.CreateAssetInput) (*api.CreateAssetOutput, error) {
	if len(input) > 50_000_000 {
		return nil, fmt.Errorf("assets larger then 50 MB are not supported")
	}
	name := uuid.NewString()
	url, err := c.assets.Create(ctx, name, bytes.NewReader(input))
	if err != nil {
		return nil, err
	}
	return &api.CreateAssetOutput{
		Name: name,
		URL:  url,
	}, nil
}

func (c *Challenge) updateAsset(ctx context.Context, input *api.UpdateAssetInput) (*api.UpdateAssetOutput, error) {
	url, err := c.assets.Update(ctx, input.OldName, input.NewName)
	if err != nil {
		return nil, err
	}
	return &api.UpdateAssetOutput{
		NewURL: url,
	}, nil
}

func (c *Challenge) createPermission(ctx context.Context, input *api.CreatePermissionInput) (*api.CreatePermissionOutput, error) {
	err := c.access.CreatePermission(ctx, input.Permission)
	return &api.CreatePermissionOutput{}, err
}

func (c *Challenge) updatePermission(ctx context.Context, input *api.UpdatePermissionInput) (*api.UpdatePermissionOutput, error) {
	err := c.access.UpdatePermission(ctx, input.Permission)
	return &api.UpdatePermissionOutput{}, err
}

func (c *Challenge) createGuardrail(ctx context.Context, input *api.CreateGuardrailInput) (*api.CreateGuardrailOutput, error) {
	err := c.access.CreateGuardrail(ctx, input.Guardrail)
	return &api.CreateGuardrailOutput{}, err
}

func (c *Challenge) updateGuardrail(ctx context.Context, input *api.UpdateGuardrailInput) (*api.UpdateGuardrailOutput, error) {
	err := c.access.UpdateGuardrail(ctx, input.Guardrail)
	return &api.UpdateGuardrailOutput{}, err
}

func (c *Challenge) createResource(ctx context.Context, input *api.CreateResourceInput) (*api.CreateResourceOutput, error) {
	id, err := c.resources.Create(ctx, input.Type, input.Desired)
	return &api.CreateResourceOutput{Identifier: id}, err
}

func (c *Challenge) readResource(ctx context.Context, input *api.ReadResourceInput) (*api.ReadResourceOutput, error) {
	state, err := c.resources.Read(ctx, input.Type, input.Identifier)
	return &api.ReadResourceOutput{State: state}, err
}

func (c *Challenge) updateResource(ctx context.Context, input *api.UpdateResourceInput) (*api.UpdateResourceOutput, error) {
	err := c.resources.Update(ctx, input.Type, input.Identifier, input.Patch)
	return &api.UpdateResourceOutput{}, err
}

func (c *Challenge) deleteResource(ctx context.Context, input *api.DeleteResourceInput) (*api.DeleteResourceOutput, error) {
	err := c.resources.Delete(ctx, input.Type, input.Identifier)
	return &api.DeleteResourceOutput{}, err
}

func (c *Challenge) listResource(ctx context.Context, input *api.ListResourceInput) (*api.ListResourceOutput, error) {
	resources, err := c.resources.List(ctx, input.Type, input.ResourceModel)
	return &api.ListResourceOutput{Resources: resources}, err
}
