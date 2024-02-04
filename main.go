package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lotusdblabs/lotusdb/v2"
)

var db *lotusdb.DB

type model struct {
	Choices     []string
	Cursor      int
	Selected    map[int]struct{}
	CurrentView string
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

	if m.CurrentView == "list" {
		updateList(&m, msg)
	} else if m.CurrentView == "add" {
		updateAdd(&m, msg)
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	modelJSON, err := m.toJSON()
	if err != nil {
		panic(err)
	}
	db.Put([]byte("model"), modelJSON, nil)

	if m.CurrentView == "" {
		m.CurrentView = "list"
	}

	return m, nil
}

func updateAdd(m *model, msg tea.Msg) tea.Model {
	m.textInput.Focus()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.CurrentView = "list"
			return m
		case "enter":
			m.Choices = append(m.Choices, m.textInput.Value())
			m.textInput.Reset()
			// move cursor to last element (newly added choice)
			m.Cursor = len(m.Choices) - 1
			m.CurrentView = "list"
			return m
		case "esc":
			m.textInput.Reset()
		}
	}

	return m
}

func updateList(m *model, msg tea.Msg) tea.Model {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.CurrentView = "add"
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
			// remove selected choice from choices slice
			m.Choices = append(m.Choices[:m.Cursor], m.Choices[m.Cursor+1:]...)
			// move the cursor up once if it would dissapear
			if m.Cursor == len(m.Choices) {
				m.Cursor--
			}
			delete(m.Selected, m.Cursor)
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
	s := "DO IT!\n\n"

	for i, choice := range m.Choices {
		cursor := "  "

		if m.Cursor == i {
			cursor = "> "
		}

		cursor += choice

		if _, ok := m.Selected[i]; ok {
			cursor += " x"
		} else {
			cursor += "  "
		}

		cursor += "\n"

		s += cursor
	}

	return s
}

func addView(m model) string {
	return m.textInput.View()
}

func (m model) View() string {
	switch m.CurrentView {
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
		return *restoredModel
	}

	return model{
		Choices:     []string{"Do a thing", "do another thing"},
		Selected:    make(map[int]struct{}),
		CurrentView: "list",
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
