package repositoryset

import (
	"gh_foundations/cmd/gen/common"
	githubfoundations "gh_foundations/internal/pkg/types/github_foundations"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	width             int
	height            int
	completeQuestions bool
	questions         []common.IQuestion
	currentQuestion   int
	loadingSpinner    spinner.Model
	repositorySet     *githubfoundations.RepositorySetInput
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
		resizeCmds := make([]tea.Cmd, 0)
		for _, q := range m.questions {
			resizeCmds = append(resizeCmds, q.Update(msg))
		}
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
		default:
			return m, m.questions[m.currentQuestion].Update(msg)
		}
	}
	cmd := m.questions[m.currentQuestion].Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.width == 0 {
		return m.loadingSpinner.View()
	}

	if !m.completeQuestions {
		return m.questions[m.currentQuestion].View()
	}

	return "Done"
}
