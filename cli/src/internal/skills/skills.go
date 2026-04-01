package skills

import (
	"embed"

	internalversion "github.com/jongio/azd-app/cli/src/internal/version"
	"github.com/jongio/azd-core/copilotskills"
)

//go:embed azd-app/SKILL.md
var skillFS embed.FS

// InstallSkill installs the azd-app skill to ~/.copilot/skills/azd-app.
func InstallSkill() error {
	return copilotskills.Install("azd-app", internalversion.Version, skillFS, "azd-app")
}
