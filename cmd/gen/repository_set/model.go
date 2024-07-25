package repositoryset

import (
	"gh_foundations/cmd/gen/common"
	githubfoundations "gh_foundations/internal/pkg/types/github_foundations"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

type model struct {
	zClickedStart     int
	width             int
	height            int
	completeQuestions bool
	questions         []common.IQuestion
	currentQuestion   int
	loadingSpinner    spinner.Model
	repositorySet     *githubfoundations.RepositorySetInput
	viewport          viewport.Model
}

var viewportKeyBindings viewport.KeyMap = viewport.KeyMap{
	PageDown: key.NewBinding(
		key.WithKeys("pgdown"),
		key.WithHelp("pgdn", "page down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("pgup", "page up"),
	),
	HalfPageUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("ctrl+u", "½ page up"),
	),
	HalfPageDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "½ page down"),
	),
	Up: key.NewBinding(
		key.WithKeys("ctrl+up"),
		key.WithHelp("ctrl+↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("ctrl+down"),
		key.WithHelp("ctrl+↓", "down"),
	),
}

func initialModel() model {
	m := model{
		loadingSpinner:    spinner.New(),
		completeQuestions: false,
		questions:         []common.IQuestion{},
		currentQuestion:   0,
		repositorySet:     new(githubfoundations.RepositorySetInput),
	}
	if terraformerStateFile == "" {
		m.questions = []common.IQuestion{
			common.NewSelectQuestion(
				"Select the visibility of the repository",
				[]string{
					"public",
					"private",
				},
			),
			common.NewCompositeQuestion(
				[]common.CompositeQuestionEntry{
					{
						Key: "q1",
						Question: common.NewTextQuestion(
							"Enter some text",
							"test",
						),
					},

					{
						Key: "q2",
						Question: common.NewListQuestion(
							"Enter the name(s) of something",
						),
					},
					{
						Key: "q3",
						Question: common.NewKeyValueListQuestion(
							"Enter some key values",
						),
					},
					{
						Key: "q4",
						Question: common.NewSelectQuestion(
							"Select something",
							[]string{
								"Something 1",
								"Something 2",
							},
						),
					},
					{
						Key: "q5",
						Question: common.NewSelectQuestion(
							"Select something",
							[]string{
								"Something 1",
								"Something 2",
							},
						),
					},
					{
						Key: "q6",
						Question: common.NewSelectQuestion(
							"Select something",
							[]string{
								"Something 1",
								"Something 2",
							},
						),
					},
					{
						Key: "q7",
						Question: common.NewSelectQuestion(
							"Select something END",
							[]string{
								"Something 1",
								"Something 2",
							},
						),
					},
				},
			),
			common.NewTextQuestion(
				"Enter the name of the repository",
				"",
			),
			common.NewTextQuestion(
				"Enter the description for the repository",
				"",
			),
			common.NewTextQuestion(
				"Enter the default branch for the repository",
				"main",
			),
			common.NewListQuestion(
				"Enter the name(s) of any protected branches",
			),
			common.NewKeyValueListQuestion(
				"Enter custom team permissions for the repository",
			),
			common.NewKeyValueListQuestion(
				"Enter custom user permissions for the repository",
			),
			common.NewSelectQuestion(
				"Enable Github Advance Security",
				[]string{
					"true",
					"false",
				},
			),
			common.NewSelectQuestion(
				"Enable vulnerability alerts",
				[]string{
					"true",
					"false",
				},
			),
			common.NewListQuestion(
				"Add Topics",
			),
			common.NewTextQuestion(
				"Enter the homepage for the repository",
				"",
			),
			common.NewSelectQuestion(
				"Delete head branches on merge",
				[]string{
					"true",
					"false",
				},
			),
			common.NewSelectQuestion(
				"Require web commit signoff",
				[]string{
					"true",
					"false",
				},
			),
			common.NewSelectQuestion(
				"Enable Dependabot security updates",
				[]string{
					"true",
					"false",
				},
			),
			common.NewSelectQuestion(
				"Allow auto merge",
				[]string{
					"true",
					"false",
				},
			),
			common.NewTextQuestion(
				"Enter the name of a license template",
				"",
			),
		}
	}
	return m
}

func (m model) Init() tea.Cmd {
	return m.loadingSpinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport = viewport.New(m.width, m.height)
		m.viewport.KeyMap = viewportKeyBindings
		resizeCmds := make([]tea.Cmd, 0)
		for _, q := range m.questions {
			q.SetDimensions(msg.Width, msg.Height)
			resizeCmds = append(resizeCmds, q.Update(msg))
		}
		m.viewport.SetContent(m.questions[0].View())
		return m, tea.Batch(resizeCmds...)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyShiftLeft, tea.KeyShiftRight:
			m.questions[m.currentQuestion].Blur()
			if msg.Type == tea.KeyShiftLeft {
				m.currentQuestion = max(0, m.currentQuestion-1)
			} else if msg.Type == tea.KeyShiftRight {
				m.currentQuestion = min(len(m.questions)-1, m.currentQuestion+1)
			}
			m.questions[m.currentQuestion].Focus()
		}
	}

	questionUpdateCmd := m.questions[m.currentQuestion].Update(msg)
	m.viewport.SetContent(m.questions[m.currentQuestion].View())

	viewportModel, viewportUpdateCmd := m.viewport.Update(msg)
	m.viewport = viewportModel
	return m, tea.Batch(viewportUpdateCmd, questionUpdateCmd)
}

func (m model) View() string {
	if m.width == 0 {
		return m.loadingSpinner.View()
	}
	if !m.completeQuestions {
		return zone.Scan(m.viewport.View())
	}

	return "Done"
}
