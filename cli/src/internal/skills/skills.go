package skills

import (
	"embed"
	"errors"

	internalversion "github.com/jongio/azd-app/cli/src/internal/version"
	"github.com/jongio/azd-core/copilotskills"
)

//go:embed azd-app/SKILL.md
var skillFS embed.FS

//go:embed azd-app-onboard/SKILL.md
var onboardSkillFS embed.FS

// InstallSkills installs all copilot skills to ~/.copilot/skills/.
func InstallSkills() error {
	return errors.Join(
		copilotskills.Install("azd-app", internalversion.Version, skillFS, "azd-app"),
		copilotskills.Install("azd-app-onboard", internalversion.Version, onboardSkillFS, "azd-app-onboard"),
	)
}
