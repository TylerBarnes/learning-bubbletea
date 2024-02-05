package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lotusdblabs/lotusdb/v2"
)

var db *lotusdb.DB

type model struct {
	Choices     []string
	Cursor      int
	Selected    map[int]struct{}
	currentView string
	textInput   textinput.Model
}

func (m *model) toJSON() ([]byte, error) {
	data, err := json.Marshal(m)

	if err != nil {
		return nil, err
	}

	return data, nil
}

func modelFromJSON(data []byte) (*model, error) {
	var m model
	err := json.Unmarshal(data, &m)

	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	if m.currentView == "list" {
		updateList(&m, msg)
	} else if m.currentView == "add" {
		updateAdd(&m, msg)
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	if len(m.Choices) < 1 {
		m.currentView = "add"
	}

	modelJSON, err := m.toJSON()
	if err != nil {
		panic(err)
	}
	db.Put([]byte("model"), modelJSON, nil)

	if m.currentView == "" {
		m.currentView = "list"
	}

	return m, nil
}

func updateAdd(m *model, msg tea.Msg) tea.Model {
	m.textInput.Focus()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.Choices = append(m.Choices, m.textInput.Value())
			m.textInput.Reset()
			delete(m.Selected, m.Cursor+1)
			// move cursor to last element (newly added choice)
			m.Cursor = len(m.Choices) - 1
			m.currentView = "list"
			return m
		case "esc":
			m.textInput.Reset()
			m.currentView = "list"
			return m

		}
	}

	return m
}

func updateList(m *model, msg tea.Msg) tea.Model {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "a":
			m.currentView = "add"
			return m
		case "k":
			if m.Cursor > 0 {
				m.Cursor--
			} else {
				m.Cursor = len(m.Choices) - 1
			}
		case "j":
			if m.Cursor < len(m.Choices)-1 {
				m.Cursor++
			} else {
				m.Cursor = 0
			}
		case "d":

			if len(m.Choices) > 0 {

				// remove selected choice from choices slice
				m.Choices = append(m.Choices[:m.Cursor], m.Choices[m.Cursor+1:]...)
				// move the cursor up once if it would dissapear
				if m.Cursor == len(m.Choices) {
					m.Cursor--
				}
				delete(m.Selected, m.Cursor)
			}
		case "enter", " ":
			_, ok := m.Selected[m.Cursor]
			if ok {
				delete(m.Selected, m.Cursor)
			} else {
				m.Selected[m.Cursor] = struct{}{}
			}
		}

	}

	return m
}

func listView(m model) string {
	s := "Tasks:\n\n"

	for i, choice := range m.Choices {
		cursor := ""

		item := strconv.Itoa(i+1) + ". " + choice

		if _, ok := m.Selected[i]; ok {
			style := lipgloss.NewStyle().Strikethrough(true)
			cursor += style.Render(item)
		} else {
			cursor += item
		}

		borderStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).PaddingRight(1).PaddingLeft(1)
		activeStyle := borderStyle.Copy().BorderForeground(lipgloss.Color("86"))
		inactiveStyle := borderStyle.Copy().BorderForeground(lipgloss.Color("12"))

		if m.Cursor == i {
			cursor = activeStyle.Render(cursor)
		} else {
			cursor = inactiveStyle.Render(cursor)
		}

		cursor += "\n"

		s += cursor
	}

	return s
}

func addView(m model) string {
	return "What's the task?\n\n" + m.textInput.View()
}

func (m model) View() string {
	switch m.currentView {
	case "add":
		return addView(m)
	case "list":
		return listView(m)

	}

	return listView(m)
}

func initialModel() model {
	storedModel, err := db.Get([]byte("model"))
	if err != nil && err != lotusdb.ErrKeyNotFound {
		panic(err)
	}

	ti := textinput.New()
	ti.Placeholder = "Do the thing"
	ti.CharLimit = 156
	ti.Width = 20

	if storedModel != nil {
		restoredModel, err := modelFromJSON(storedModel)
		if err != nil {
			panic(err)
		}
		restoredModel.textInput = ti
		restoredModel.currentView = "list"
		return *restoredModel
	}

	return model{
		Choices:     []string{"Do a thing", "do another thing"},
		Selected:    make(map[int]struct{}),
		currentView: "list",
		textInput:   ti,
	}
}

func main() {
	options := lotusdb.DefaultOptions
	options.DirPath = "exdb"
	ldb, err := lotusdb.Open(options)

	if err != nil {
		panic(err)
	}
	defer func() {
		_ = db.Close()
	}()

	db = ldb
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
