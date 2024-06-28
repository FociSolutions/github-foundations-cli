package listinput

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mode string

const (
	adding  mode = "adding"
	editing      = "editing"
)

type Model struct {
	state  mode
	prompt string
	values list.Model
	input  textinput.Model
}

func New(prompt string) Model {
	return Model{
		state:  adding,
		prompt: prompt,
		values: list.New(make([]list.Item, 0), itemDelegate{}, 0, 0),
		input:  textinput.New(),
	}
}

type item string

func (i item) FilterValue() string { return fmt.Sprint(i) }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprint(i)

	fn := lipgloss.NewStyle().PaddingLeft(4).Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color("170")).Render(strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.input.Width = msg.Width
		m.values.SetWidth(msg.Width)
		m.values.SetHeight(msg.Height - 5)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			insertCmd := m.values.InsertItem(len(m.values.Items()), item(m.input.Value()))
			m.input.Reset()
			return m, insertCmd
		case tea.KeyUp, tea.KeyDown:
			if m.state == editing {
				m.selectItem(msg.Type)
			}
		case tea.KeyTab:
			if m.state == editing {
				m.state = adding
				m.selectItem(msg.Type)
			} else if m.state == adding {
				m.state = editing
				m.input.SetValue("")
			}
		}
	}
	inputModel, inputUpdateCmd := m.input.Update(msg)
	valuesModel, valuesUpdateCmd := m.values.Update(msg)

	m.input = inputModel
	m.values = valuesModel

	return m, tea.Batch(inputUpdateCmd, valuesUpdateCmd)
}

func (m Model) View() string {
	return lipgloss.JoinVertical(lipgloss.Left, fmt.Sprintf("%s:", m.prompt), m.input.View(), m.values.View())
}

func (m Model) selectItem(keyPressed tea.KeyType) {
	index := m.values.Index()
	switch keyPressed {
	case tea.KeyDown:
		index = max(0, index-1)
	case tea.KeyUp:
		index = min(len(m.values.Items())-1, index+1)
	case tea.KeyTab:
		index = 0
	}
	m.values.Select(index)
	m.input.SetValue(m.values.SelectedItem().FilterValue())
}

func (m Model) Values() []string {
	items := m.values.Items()
	values := make([]string, len(items))
	for i, item := range items {
		values[i] = fmt.Sprint(item)
	}
	return values
}

func (m Model) Focus() {
	m.input.Focus()
}

func (m Model) Blur() {
	m.input.Blur()
}
