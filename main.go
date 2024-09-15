package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type NPMPackageInfo struct {
	DistTags struct {
		Latest string `json:"latest"`
	} `json:"dist-tags"`
}

type Dependency struct {
	Name           string
	CurrentVersion string
	LatestVersion  string
	IsDev          bool
}

type model struct {
	table table.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	default:
		m.table, cmd = m.table.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	return m.table.View() + "\nPress q for exit\n"
}

func main() {
	dependencies, err := getDependencies()
	if err != nil {
		fmt.Println("Erro:", err)
		return
	}

	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Current Version", Width: 15},
		{Title: "Updated Version", Width: 15},
		{Title: "Type", Width: 10},
	}

	rows := []table.Row{}
	for _, dep := range dependencies {
		depType := "Produção"
		if dep.IsDev {
			depType = "Dev"
		}
		rows = append(rows, table.Row{dep.Name, dep.CurrentVersion, dep.LatestVersion, depType})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(dependencies)),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("39")). // Cor azul para o cabeçalho
		Background(lipgloss.Color("236")) // Fundo cinza escuro para o cabeçalho

	t.SetStyles(s)

	m := model{t}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Erro ao executar programa:", err)
		os.Exit(1)
	}
}

func getDependencies() ([]Dependency, error) {
	file, err := os.ReadFile("package.json")
	if err != nil {
		return nil, fmt.Errorf("erro ao ler o arquivo package.json: %v", err)
	}

	var pkg PackageJSON
	err = json.Unmarshal(file, &pkg)
	if err != nil {
		return nil, fmt.Errorf("erro ao analisar o JSON: %v", err)
	}

	dependencies := []Dependency{}

	// Processar dependências de produção
	for dep, version := range pkg.Dependencies {
		dependency, err := processDependency(dep, version, false)
		if err != nil {
			fmt.Printf("Erro ao processar dependência %s: %v\n", dep, err)
			continue
		}
		dependencies = append(dependencies, dependency)
	}

	// Processar devDependencies
	for dep, version := range pkg.DevDependencies {
		dependency, err := processDependency(dep, version, true)
		if err != nil {
			fmt.Printf("Erro ao processar devDependency %s: %v\n", dep, err)
			continue
		}
		dependencies = append(dependencies, dependency)
	}

	return dependencies, nil
}

func processDependency(name, version string, isDev bool) (Dependency, error) {
	latestVersion, err := getLatestVersion(name)
	if err != nil {
		return Dependency{}, err
	}

	currentVersion := strings.TrimPrefix(version, "^")
	return Dependency{
		Name:           name,
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		IsDev:          isDev,
	}, nil
}

func getLatestVersion(packageName string) (string, error) {
	url := fmt.Sprintf("https://registry.npmjs.org/%s", packageName)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var npmInfo NPMPackageInfo
	err = json.Unmarshal(body, &npmInfo)
	if err != nil {
		return "", err
	}

	return npmInfo.DistTags.Latest, nil
}
