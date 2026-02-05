package prompter

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/reglet-dev/reglet-sdk/domain/entities"
)

// CliPrompter implements ports.Prompter for CLI environments.
type CliPrompter struct {
	in  io.Reader
	out io.Writer
}

// NewCliPrompter creates a new CliPrompter.
func NewCliPrompter(in io.Reader, out io.Writer) *CliPrompter {
	return &CliPrompter{in: in, out: out}
}

// IsInteractive checks if the input is a terminal.
func (p *CliPrompter) IsInteractive() bool {
	if f, ok := p.in.(*os.File); ok {
		stat, err := f.Stat()
		if err != nil {
			return false
		}
		return (stat.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

// PromptForCapability asks the user to grant a single capability.
func (p *CliPrompter) PromptForCapability(req entities.CapabilityRequest) (granted bool, always bool, err error) {
	_, _ = fmt.Fprintf(p.out, "Plugin Request: %s\n", req.Description)
	_, _ = fmt.Fprintf(p.out, "Risk: %s\n", req.RiskLevel)
	_, _ = fmt.Fprintf(p.out, "Allow? [y/n/always]: ")

	scanner := bufio.NewScanner(p.in)
	if scanner.Scan() {
		text := strings.ToLower(strings.TrimSpace(scanner.Text()))
		switch text {
		case "y", "yes":
			return true, false, nil
		case "a", "always":
			return true, true, nil
		case "n", "no":
			return false, false, nil
		default:
			return false, false, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, false, err
	}
	return false, false, io.EOF
}

// PromptForCapabilities prompts for multiple capabilities.
func (p *CliPrompter) PromptForCapabilities(reqs []entities.CapabilityRequest) (*entities.GrantSet, error) {
	if len(reqs) == 0 {
		return &entities.GrantSet{}, nil
	}

	_, _ = fmt.Fprintf(p.out, "Plugin requests the following capabilities:\n")
	for _, req := range reqs {
		// Display with rule details if available
		if reqRule, ok := req.Rule.(string); ok && reqRule != "" {
			_, _ = fmt.Fprintf(p.out, "- [%s] %s (%s)\n", req.RiskLevel, req.Description, reqRule)
		} else {
			_, _ = fmt.Fprintf(p.out, "- [%s] %s\n", req.RiskLevel, req.Description)
		}
	}
	_, _ = fmt.Fprintf(p.out, "Grant all? [y/n]: ")

	scanner := bufio.NewScanner(p.in)
	if scanner.Scan() {
		text := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if text == "y" || text == "yes" {
			gs := &entities.GrantSet{}
			for _, req := range reqs {
				addRuleToGrantSet(gs, req.Rule)
			}
			return gs, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Default deny
	return &entities.GrantSet{}, nil
}

func addRuleToGrantSet(gs *entities.GrantSet, rule interface{}) {
	temp := &entities.GrantSet{}
	switch r := rule.(type) {
	case *entities.NetworkCapability:
		temp.Network = r
	case *entities.FileSystemCapability:
		temp.FS = r
	case *entities.EnvironmentCapability:
		temp.Env = r
	case *entities.ExecCapability:
		temp.Exec = r
	case *entities.KeyValueCapability:
		temp.KV = r
	}
	gs.Merge(temp)
}

// FormatNonInteractiveError creates a helpful error.
func (p *CliPrompter) FormatNonInteractiveError(missing *entities.GrantSet) error {
	// TODO: Create a more detailed error message listing missing capabilities
	return fmt.Errorf("plugin requires capabilities in non-interactive mode. Please review and update grants")
}
