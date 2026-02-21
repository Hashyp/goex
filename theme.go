package main

import "github.com/charmbracelet/lipgloss"

type appTheme struct {
	name        string
	header      lipgloss.Color
	border      lipgloss.Color
	text        lipgloss.Color
	missing     lipgloss.Color
	highlightBG lipgloss.Color
	highlightFG lipgloss.Color
	selectedBG  lipgloss.Color
	selectedFG  lipgloss.Color
}

var (
	catppuccinMocha = appTheme{
		name:        "mocha",
		header:      lipgloss.Color("#74c7ec"),
		border:      lipgloss.Color("#585b70"),
		text:        lipgloss.Color("#cdd6f4"),
		missing:     lipgloss.Color("#f38ba8"),
		highlightBG: lipgloss.Color("#89b4fa"),
		highlightFG: lipgloss.Color("#1e1e2e"),
		selectedBG:  lipgloss.Color("#45475a"),
		selectedFG:  lipgloss.Color("#f5e0dc"),
	}

	catppuccinLatte = appTheme{
		name:        "latte",
		header:      lipgloss.Color("#209fb5"),
		border:      lipgloss.Color("#acb0be"),
		text:        lipgloss.Color("#4c4f69"),
		missing:     lipgloss.Color("#d20f39"),
		highlightBG: lipgloss.Color("#1e66f5"),
		highlightFG: lipgloss.Color("#eff1f5"),
		selectedBG:  lipgloss.Color("#ccd0da"),
		selectedFG:  lipgloss.Color("#1e66f5"),
	}

	catppuccinFrappe = appTheme{
		name:        "frappe",
		header:      lipgloss.Color("#85c1dc"),
		border:      lipgloss.Color("#737994"),
		text:        lipgloss.Color("#c6d0f5"),
		missing:     lipgloss.Color("#e78284"),
		highlightBG: lipgloss.Color("#8caaee"),
		highlightFG: lipgloss.Color("#303446"),
		selectedBG:  lipgloss.Color("#51576d"),
		selectedFG:  lipgloss.Color("#a5adce"),
	}

	catppuccinMacchiato = appTheme{
		name:        "macchiato",
		header:      lipgloss.Color("#91d7e3"),
		border:      lipgloss.Color("#6e738d"),
		text:        lipgloss.Color("#cad3f5"),
		missing:     lipgloss.Color("#ed8796"),
		highlightBG: lipgloss.Color("#8aadf4"),
		highlightFG: lipgloss.Color("#24273a"),
		selectedBG:  lipgloss.Color("#494d64"),
		selectedFG:  lipgloss.Color("#f4dbd6"),
	}

	nord = appTheme{
		name:        "nord",
		header:      lipgloss.Color("#88c0d0"),
		border:      lipgloss.Color("#4c566a"),
		text:        lipgloss.Color("#e5e9f0"),
		missing:     lipgloss.Color("#bf616a"),
		highlightBG: lipgloss.Color("#5e81ac"),
		highlightFG: lipgloss.Color("#eceff4"),
		selectedBG:  lipgloss.Color("#434c5e"),
		selectedFG:  lipgloss.Color("#88c0d0"),
	}

	themes = []appTheme{
		catppuccinMocha,
		catppuccinLatte,
		catppuccinFrappe,
		catppuccinMacchiato,
		nord,
	}
)

func nextThemeIndex(current int) int {
	if len(themes) == 0 {
		return 0
	}

	return (current + 1) % len(themes)
}
