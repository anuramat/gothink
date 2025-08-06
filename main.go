package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ThoughtData struct {
	Thought            string  `json:"thought"`
	ThoughtNumber      int     `json:"thoughtNumber"`
	TotalThoughts      int     `json:"totalThoughts"`
	NextThoughtNeeded  bool    `json:"nextThoughtNeeded"`
	IsRevision         *bool   `json:"isRevision,omitempty"`
	RevisesThought     *int    `json:"revisesThought,omitempty"`
	BranchFromThought  *int    `json:"branchFromThought,omitempty"`
	BranchId           *string `json:"branchId,omitempty"`
	NeedsMoreThoughts  *bool   `json:"needsMoreThoughts,omitempty"`
}

type SequentialThinkingServer struct {
	thoughtHistory         []ThoughtData
	branches               map[string][]ThoughtData
	disableThoughtLogging  bool
}

func NewSequentialThinkingServer() *SequentialThinkingServer {
	return &SequentialThinkingServer{
		thoughtHistory:        make([]ThoughtData, 0),
		branches:              make(map[string][]ThoughtData),
		disableThoughtLogging: strings.ToLower(os.Getenv("DISABLE_THOUGHT_LOGGING")) == "true",
	}
}

func (s *SequentialThinkingServer) validateThoughtData(args map[string]any) (*ThoughtData, error) {
	data := &ThoughtData{}
	
	thought, ok := args["thought"].(string)
	if !ok || thought == "" {
		return nil, fmt.Errorf("invalid thought: must be a string")
	}
	data.Thought = thought

	if val, ok := args["thoughtNumber"]; !ok {
		return nil, fmt.Errorf("invalid thoughtNumber: must be a number")
	} else if num, ok := val.(float64); ok {
		data.ThoughtNumber = int(num)
	} else {
		return nil, fmt.Errorf("invalid thoughtNumber: must be a number")
	}

	if val, ok := args["totalThoughts"]; !ok {
		return nil, fmt.Errorf("invalid totalThoughts: must be a number")
	} else if num, ok := val.(float64); ok {
		data.TotalThoughts = int(num)
	} else {
		return nil, fmt.Errorf("invalid totalThoughts: must be a number")
	}

	if val, ok := args["nextThoughtNeeded"]; !ok {
		return nil, fmt.Errorf("invalid nextThoughtNeeded: must be a boolean")
	} else if b, ok := val.(bool); ok {
		data.NextThoughtNeeded = b
	} else {
		return nil, fmt.Errorf("invalid nextThoughtNeeded: must be a boolean")
	}

	if val, ok := args["isRevision"]; ok {
		if b, ok := val.(bool); ok {
			data.IsRevision = &b
		}
	}

	if val, ok := args["revisesThought"]; ok {
		if num, ok := val.(float64); ok {
			thought := int(num)
			data.RevisesThought = &thought
		}
	}

	if val, ok := args["branchFromThought"]; ok {
		if num, ok := val.(float64); ok {
			thought := int(num)
			data.BranchFromThought = &thought
		}
	}

	if val, ok := args["branchId"]; ok {
		if s, ok := val.(string); ok {
			data.BranchId = &s
		}
	}

	if val, ok := args["needsMoreThoughts"]; ok {
		if b, ok := val.(bool); ok {
			data.NeedsMoreThoughts = &b
		}
	}

	return data, nil
}

func (s *SequentialThinkingServer) formatThought(data *ThoughtData) string {
	var prefix, context string

	if data.IsRevision != nil && *data.IsRevision {
		prefix = color.YellowString("🔄 Revision")
		if data.RevisesThought != nil {
			context = fmt.Sprintf(" (revising thought %d)", *data.RevisesThought)
		}
	} else if data.BranchFromThought != nil && data.BranchId != nil {
		prefix = color.GreenString("🌿 Branch")
		context = fmt.Sprintf(" (from thought %d, ID: %s)", *data.BranchFromThought, *data.BranchId)
	} else {
		prefix = color.BlueString("💭 Thought")
	}

	header := fmt.Sprintf("%s %d/%d%s", prefix, data.ThoughtNumber, data.TotalThoughts, context)
	border := strings.Repeat("─", maxLen(len(header), len(data.Thought)) + 4)

	return fmt.Sprintf("\n┌%s┐\n│ %s │\n├%s┤\n│ %-*s │\n└%s┘",
		border, header, border, len(border)-2, data.Thought, border)
}

func maxLen(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (s *SequentialThinkingServer) processThought(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	
	validatedInput, err := s.validateThoughtData(args)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if validatedInput.ThoughtNumber > validatedInput.TotalThoughts {
		validatedInput.TotalThoughts = validatedInput.ThoughtNumber
	}

	s.thoughtHistory = append(s.thoughtHistory, *validatedInput)

	if validatedInput.BranchFromThought != nil && validatedInput.BranchId != nil {
		branchId := *validatedInput.BranchId
		if s.branches[branchId] == nil {
			s.branches[branchId] = make([]ThoughtData, 0)
		}
		s.branches[branchId] = append(s.branches[branchId], *validatedInput)
	}

	if !s.disableThoughtLogging {
		formattedThought := s.formatThought(validatedInput)
		fmt.Fprintf(os.Stderr, "%s\n", formattedThought)
	}

	branches := make([]string, 0, len(s.branches))
	for k := range s.branches {
		branches = append(branches, k)
	}

	result := map[string]any{
		"thoughtNumber":       validatedInput.ThoughtNumber,
		"totalThoughts":       validatedInput.TotalThoughts,
		"nextThoughtNeeded":   validatedInput.NextThoughtNeeded,
		"branches":            branches,
		"thoughtHistoryLength": len(s.thoughtHistory),
	}

	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func main() {
	s := server.NewMCPServer(
		"sequential-thinking-server",
		"0.2.0",
	)

	thinkingServer := NewSequentialThinkingServer()

	tool := mcp.NewTool("sequentialthinking",
		mcp.WithDescription(`A detailed tool for dynamic and reflective problem-solving through thoughts.
This tool helps analyze problems through a flexible thinking process that can adapt and evolve.
Each thought can build on, question, or revise previous insights as understanding deepens.

When to use this tool:
- Breaking down complex problems into steps
- Planning and design with room for revision
- Analysis that might need course correction
- Problems where the full scope might not be clear initially
- Problems that require a multi-step solution
- Tasks that need to maintain context over multiple steps
- Situations where irrelevant information needs to be filtered out

Key features:
- You can adjust total_thoughts up or down as you progress
- You can question or revise previous thoughts
- You can add more thoughts even after reaching what seemed like the end
- You can express uncertainty and explore alternative approaches
- Not every thought needs to build linearly - you can branch or backtrack
- Generates a solution hypothesis
- Verifies the hypothesis based on the Chain of Thought steps
- Repeats the process until satisfied
- Provides a correct answer

Parameters explained:
- thought: Your current thinking step, which can include:
* Regular analytical steps
* Revisions of previous thoughts
* Questions about previous decisions
* Realizations about needing more analysis
* Changes in approach
* Hypothesis generation
* Hypothesis verification
- next_thought_needed: True if you need more thinking, even if at what seemed like the end
- thought_number: Current number in sequence (can go beyond initial total if needed)
- total_thoughts: Current estimate of thoughts needed (can be adjusted up/down)
- is_revision: A boolean indicating if this thought revises previous thinking
- revises_thought: If is_revision is true, which thought number is being reconsidered
- branch_from_thought: If branching, which thought number is the branching point
- branch_id: Identifier for the current branch (if any)
- needs_more_thoughts: If reaching end but realizing more thoughts needed

You should:
1. Start with an initial estimate of needed thoughts, but be ready to adjust
2. Feel free to question or revise previous thoughts
3. Don't hesitate to add more thoughts if needed, even at the "end"
4. Express uncertainty when present
5. Mark thoughts that revise previous thinking or branch into new paths
6. Ignore information that is irrelevant to the current step
7. Generate a solution hypothesis when appropriate
8. Verify the hypothesis based on the Chain of Thought steps
9. Repeat the process until satisfied with the solution
10. Provide a single, ideally correct answer as the final output
11. Only set next_thought_needed to false when truly done and a satisfactory answer is reached`),
		mcp.WithString("thought",
			mcp.Required(),
			mcp.Description("Your current thinking step"),
		),
		mcp.WithBoolean("nextThoughtNeeded",
			mcp.Required(),
			mcp.Description("Whether another thought step is needed"),
		),
		mcp.WithNumber("thoughtNumber",
			mcp.Required(),
			mcp.Description("Current thought number"),
		),
		mcp.WithNumber("totalThoughts",
			mcp.Required(),
			mcp.Description("Estimated total thoughts needed"),
		),
		mcp.WithBoolean("isRevision",
			mcp.Description("Whether this revises previous thinking"),
		),
		mcp.WithNumber("revisesThought",
			mcp.Description("Which thought is being reconsidered"),
		),
		mcp.WithNumber("branchFromThought",
			mcp.Description("Branching point thought number"),
		),
		mcp.WithString("branchId",
			mcp.Description("Branch identifier"),
		),
		mcp.WithBoolean("needsMoreThoughts",
			mcp.Description("If more thoughts are needed"),
		),
	)

	s.AddTool(tool, thinkingServer.processThought)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}