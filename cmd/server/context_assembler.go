package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	expdomain "github.com/0xsj/canopy-backend/internal/exploration/domain"
	seeddomain "github.com/0xsj/canopy-backend/internal/seed/domain"
	sessiondomain "github.com/0xsj/canopy-backend/internal/session/domain"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// seedFinder reads seed data for context assembly. Satisfied by postgres.SeedRepository.
type seedFinder interface {
	FindByID(ctx context.Context, id types.SeedID) (seeddomain.Seed, error)
}

// leafFinder reads leaf data for context assembly. Satisfied by postgres.LeafRepository.
type leafFinder interface {
	FindByID(ctx context.Context, id types.LeafID) (expdomain.Leaf, error)
	FindByIDs(ctx context.Context, ids []types.LeafID) ([]expdomain.Leaf, error)
}

// contextAssembler builds the full LLM message array based on session type
// and related domain data (seed, parent leaf). Replaces the noopContextAssembler.
type contextAssembler struct {
	seeds  seedFinder
	leaves leafFinder
	log    logger.Logger
}

func (a *contextAssembler) Assemble(ctx context.Context, session sessiondomain.Session) ([]sessiondomain.Message, error) {
	// Load seed context — every session has a seed.
	seed, err := a.seeds.FindByID(ctx, session.SeedID())
	if err != nil {
		return nil, fmt.Errorf("context assembler: load seed: %w", err)
	}

	// Load parent leaf if present.
	var parentLeaf *expdomain.Leaf
	if !session.ParentLeafID().IsZero() {
		leaf, err := a.leaves.FindByID(ctx, session.ParentLeafID())
		if err != nil {
			a.log.Warn("context assembler: parent leaf not found, continuing without it",
				logger.String("leaf_id", session.ParentLeafID().String()),
				logger.Err(err),
			)
		} else {
			parentLeaf = &leaf
		}
	}

	// Load source leaves for synthesis sessions.
	var sourceLeaves []expdomain.Leaf
	if ids := session.SourceLeafIDs(); len(ids) > 0 {
		sourceLeaves, err = a.leaves.FindByIDs(ctx, ids)
		if err != nil {
			a.log.Warn("context assembler: failed to load source leaves, continuing without them",
				logger.Err(err),
			)
			sourceLeaves = nil
		}
	}

	// Build system prompt based on session type.
	var systemPrompt string
	switch session.Type() {
	case sessiondomain.SessionExploration:
		systemPrompt = buildExplorationPrompt(seed, parentLeaf)
	case sessiondomain.SessionShaping:
		systemPrompt = buildShapingPrompt(seed, parentLeaf)
	case sessiondomain.SessionSynthesis:
		systemPrompt = buildSynthesisPrompt(seed, sourceLeaves)
	case sessiondomain.SessionDigest:
		systemPrompt = buildDigestPrompt(seed)
	default:
		systemPrompt = buildExplorationPrompt(seed, parentLeaf)
	}

	// Assemble: system message + conversation history.
	messages := make([]sessiondomain.Message, 0, 1+len(session.Messages()))
	messages = append(messages, sessiondomain.Message{
		Role:    "system",
		Content: systemPrompt,
	})
	messages = append(messages, session.Messages()...)

	return messages, nil
}

func buildExplorationPrompt(seed seeddomain.Seed, parentLeaf *expdomain.Leaf) string {
	var b strings.Builder

	b.WriteString("You are a collaborative thinking partner in Canopy, a platform for structured exploration of ideas.\n\n")
	b.WriteString("## Seed Topic\n")
	b.WriteString(fmt.Sprintf("**Title:** %s\n", seed.Title()))
	if seed.Description() != "" {
		b.WriteString(fmt.Sprintf("**Description:** %s\n", seed.Description()))
	}

	if constraints := formatConstraints(seed.Constraints()); constraints != "" {
		b.WriteString(fmt.Sprintf("\n## Constraints\n%s\n", constraints))
	}

	if parentLeaf != nil {
		b.WriteString("\n## Building On\n")
		b.WriteString(fmt.Sprintf("The user is branching from an existing idea:\n"))
		b.WriteString(fmt.Sprintf("**%s**: %s\n", parentLeaf.Title(), parentLeaf.Summary()))
		if len(parentLeaf.KeyPoints()) > 0 {
			b.WriteString("Key points:\n")
			for _, kp := range parentLeaf.KeyPoints() {
				b.WriteString(fmt.Sprintf("- %s\n", kp))
			}
		}
		if len(parentLeaf.OpenQuestions()) > 0 {
			b.WriteString("Open questions to explore:\n")
			for _, q := range parentLeaf.OpenQuestions() {
				b.WriteString(fmt.Sprintf("- %s\n", q))
			}
		}
	}

	b.WriteString("\n## Your Role\n")
	b.WriteString("Help the user explore this topic through conversation. Ask clarifying questions, ")
	b.WriteString("offer different perspectives, challenge assumptions, and help develop their thinking. ")
	b.WriteString("Keep responses focused and concise. The goal is to develop a well-formed idea that can ")
	b.WriteString("become a structured leaf (with title, summary, key points, and open questions).")

	return b.String()
}

func buildShapingPrompt(seed seeddomain.Seed, parentLeaf *expdomain.Leaf) string {
	var b strings.Builder

	b.WriteString("You are helping the user refine and structure their thinking into a well-formed leaf in Canopy.\n\n")
	b.WriteString("## Seed Topic\n")
	b.WriteString(fmt.Sprintf("**Title:** %s\n", seed.Title()))
	if seed.Description() != "" {
		b.WriteString(fmt.Sprintf("**Description:** %s\n", seed.Description()))
	}

	if parentLeaf != nil {
		b.WriteString(fmt.Sprintf("\n## Parent Idea\n**%s**: %s\n", parentLeaf.Title(), parentLeaf.Summary()))
	}

	b.WriteString("\n## Your Role\n")
	b.WriteString("The user has been exploring this topic and is now ready to structure their thinking. ")
	b.WriteString("Help them develop:\n\n")
	b.WriteString("1. **Title** — a clear, concise name for the idea\n")
	b.WriteString("2. **Summary** — a paragraph capturing the core insight\n")
	b.WriteString("3. **Key Points** — 3-5 bullet points of the most important aspects\n")
	b.WriteString("4. **Open Questions** — remaining questions worth exploring further\n")
	b.WriteString("5. **Tags** — relevant categories or themes\n\n")
	b.WriteString("Work iteratively with the user. Don't produce the final structure immediately — ")
	b.WriteString("help them refine each element through conversation.")

	return b.String()
}

func buildSynthesisPrompt(seed seeddomain.Seed, sourceLeaves []expdomain.Leaf) string {
	var b strings.Builder

	b.WriteString("You are helping synthesize multiple ideas into a coherent whole in Canopy.\n\n")
	b.WriteString("## Seed Topic\n")
	b.WriteString(fmt.Sprintf("**Title:** %s\n", seed.Title()))
	if seed.Description() != "" {
		b.WriteString(fmt.Sprintf("**Description:** %s\n", seed.Description()))
	}

	if len(sourceLeaves) > 0 {
		b.WriteString("\n## Source Ideas to Synthesize\n")
		for i, leaf := range sourceLeaves {
			b.WriteString(fmt.Sprintf("\n### Source %d: %s\n", i+1, leaf.Title()))
			if leaf.Summary() != "" {
				b.WriteString(fmt.Sprintf("**Summary:** %s\n", leaf.Summary()))
			}
			if len(leaf.KeyPoints()) > 0 {
				b.WriteString("**Key points:**\n")
				for _, kp := range leaf.KeyPoints() {
					b.WriteString(fmt.Sprintf("- %s\n", kp))
				}
			}
			if len(leaf.OpenQuestions()) > 0 {
				b.WriteString("**Open questions:**\n")
				for _, q := range leaf.OpenQuestions() {
					b.WriteString(fmt.Sprintf("- %s\n", q))
				}
			}
		}
	}

	b.WriteString("\n## Your Role\n")
	if len(sourceLeaves) > 0 {
		b.WriteString("The source ideas above have been selected for synthesis. Your job is to:\n\n")
	} else {
		b.WriteString("The user will share multiple ideas (leaves) that they want to combine. Your job is to:\n\n")
	}
	b.WriteString("1. Identify common themes and complementary insights across the sources\n")
	b.WriteString("2. Resolve any tensions or contradictions between ideas\n")
	b.WriteString("3. Produce a unified synthesis that preserves the strengths of each source\n")
	b.WriteString("4. Maintain attribution — note which source each insight came from\n")
	b.WriteString("5. Structure the result as a leaf (title, summary, key points, open questions, tags)\n\n")
	b.WriteString("Work with the user to refine the synthesis. Don't just merge mechanically — ")
	b.WriteString("find the deeper connections between the ideas.")

	return b.String()
}

func buildDigestPrompt(seed seeddomain.Seed) string {
	var b strings.Builder

	b.WriteString("You are summarizing recent activity and developments in a Canopy workspace.\n\n")
	b.WriteString("## Seed Topic\n")
	b.WriteString(fmt.Sprintf("**Title:** %s\n", seed.Title()))
	if seed.Description() != "" {
		b.WriteString(fmt.Sprintf("**Description:** %s\n", seed.Description()))
	}

	b.WriteString("\n## Your Role\n")
	b.WriteString("Create a concise digest of the thinking that has happened around this topic. ")
	b.WriteString("Highlight key developments, emerging patterns, and areas that need attention. ")
	b.WriteString("Keep it brief and actionable.")

	return b.String()
}

// formatConstraints renders seed constraints as readable text.
func formatConstraints(constraints map[string]any) string {
	if len(constraints) == 0 {
		return ""
	}

	// Try JSON for structured constraints.
	data, err := json.MarshalIndent(constraints, "", "  ")
	if err != nil {
		// Fallback to key-value list.
		var parts []string
		for k, v := range constraints {
			parts = append(parts, fmt.Sprintf("- **%s**: %v", k, v))
		}
		return strings.Join(parts, "\n")
	}
	return string(data)
}
