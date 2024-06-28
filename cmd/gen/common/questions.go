package common

import (
	"encoding/json"
	"fmt"
	listinput "gh_foundations/cmd/gen/common/listInput"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type IQuestion interface {
	GetAnswer() string
	View() string
	Update(msg tea.Msg) tea.Cmd
	Focus()
	Blur()
}

type TextQuestion struct {
	prompt string
	model  textinput.Model
}

func NewTextQuestion(prompt string, defaultValue string) *TextQuestion {
	m := textinput.New()
	m.SetValue(defaultValue)
	return &TextQuestion{
		prompt: prompt,
		model:  m,
	}
}

func (t *TextQuestion) GetAnswer() string {
	return t.model.Value()
}

func (t *TextQuestion) View() string {
	return lipgloss.JoinVertical(lipgloss.Center, fmt.Sprintf("%s:", t.prompt), t.model.View())
}

func (t *TextQuestion) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.model.Width = msg.Width
	}
	m, cmd := t.model.Update(msg)
	t.model = m
	return cmd
}

func (t *TextQuestion) Focus() {
	t.model.Focus()
}

func (t *TextQuestion) Blur() {
	t.model.Blur()
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
			return lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")).Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type SelectQuestion struct {
	prompt string
	model  list.Model
}

func NewSelectQuestion(prompt string, selection []string) *SelectQuestion {
	items := make([]list.Item, len(selection))
	for i, s := range selection {
		items[i] = item(s)
	}
	return &SelectQuestion{
		prompt: prompt,
		model:  list.New(items, itemDelegate{}, 0, 0),
	}
}

func (s *SelectQuestion) GetAnswer() string {
	return s.model.SelectedItem().FilterValue()
}

func (s *SelectQuestion) View() string {
	return lipgloss.JoinVertical(lipgloss.Center, fmt.Sprintf("%s:", s.prompt), s.model.View())
}

func (s *SelectQuestion) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.model.SetWidth(msg.Width)
		s.model.SetHeight(msg.Height - 1)
	}
	m, cmd := s.model.Update(msg)
	s.model = m
	return cmd
}

func (s *SelectQuestion) Focus() {
}

func (s *SelectQuestion) Blur() {
}

type ListQuestion struct {
	prompt string
	model  listinput.Model
}

func NewListQuestion(prompt string) *ListQuestion {
	return &ListQuestion{
		prompt: prompt,
		model:  listinput.New(prompt),
	}
}

func (l *ListQuestion) GetAnswer() string {
	values := l.model.Values()
	bytes, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func (l *ListQuestion) View() string {
	return l.model.View()
}

func (l *ListQuestion) Update(msg tea.Msg) tea.Cmd {
	m, cmd := l.model.Update(msg)
	l.model = m
	return cmd
}

func (l *ListQuestion) Focus() {
	l.model.Focus()
}

func (l *ListQuestion) Blur() {
	l.model.Blur()
}
