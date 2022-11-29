package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
)

var (
	label     string
	path      string
	repo      *git.Repository
	doneStyle = lipgloss.NewStyle().Margin(1, 2)
	checkMark = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).SetString("✓")
)

type version struct {
	Major int
	Minor int
	Patch int
	Label string
}

func (v version) String() string {
	if v.Label != "" {
		return fmt.Sprintf("v%d.%d.%d-%s", v.Major, v.Minor, v.Patch, v.Label)
	}
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v version) isGreaterThan(other version) bool {
	if v.Major > other.Major {
		return true
	}
	if v.Major == other.Major && v.Minor > other.Minor {
		return true
	}
	if v.Major == other.Major && v.Minor == other.Minor && v.Patch > other.Patch {
		return true
	}
	return false
}

type model struct {
	currentVersion version
	newVersion     version
	annotation     textinput.Model
	commitMessage  string
	done           bool
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func initialModel(newVersion version) model {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	if r, err := git.PlainOpen(path); err != nil {
		log.Fatal().Err(err).Msg("error opening git repo")
		os.Exit(1)
	} else {
		repo = r
	}
	v, err := getLatestVersion(repo)
	if err != nil {
		log.Fatal().Err(err).Msg("error getting latest version")
		os.Exit(1)
	}

	commitMessage, err := getLastCommitMessage(repo)
	if err != nil {
		log.Fatal().Err(err).Msg("error getting latest commit message")
		os.Exit(1)
	}
	ti.Placeholder = commitMessage

	return model{
		currentVersion: *v,
		newVersion:     newVersion,
		annotation:     ti,
		commitMessage:  commitMessage,
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.KeyMsg:
		k := msg.String()
		switch k {
		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if err := updateVersion(repo, m.newVersion, m.annotation.Value()); err != nil {
				log.Error().Err(err).Msg("error updating version")

			} else {
				m.done = true
			}
			return m, tea.Quit
		case "right":
			if len(m.annotation.Value()) == 0 {
				m.annotation.SetValue(m.commitMessage)
			}
		default:
			m.annotation, cmd = m.annotation.Update(msg)
		}

	}
	return m, cmd
}

func (m model) View() string {
	if m.done {
		renderString := fmt.Sprintf("%s Updated version to %s", checkMark, m.newVersion.String())
		return doneStyle.Render(renderString)
	}

	return fmt.Sprintf("Current Version: %s\nNew Version: %s\n\nPlease supply an annotation\n%s",
		m.currentVersion.String(),
		m.newVersion.String(),
		m.annotation.View())
}

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("error getting current working directory")
	}
	patchVersionCmd.Flags().StringVarP(&label, "label", "l", "", "label")
	patchVersionCmd.Flags().StringVarP(&path, "path", "p", cwd, "path")

	minorVersionCmd.Flags().StringVarP(&path, "path", "p", cwd, "path")

	majorVersionCmd.Flags().StringVarP(&path, "path", "p", cwd, "path")

	rootCmd.AddCommand(patchVersionCmd)
	rootCmd.AddCommand(minorVersionCmd)
	rootCmd.AddCommand(majorVersionCmd)

}

var rootCmd = &cobra.Command{
	Use:   "git-version",
	Short: "Git Version Commands",
	Long:  `Commands for interacting with a Git Repo and managing version tags`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		log.Fatal().Msg("you need to specify a subcommand")
		os.Exit(1)
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var patchVersionCmd = &cobra.Command{
	Use:   "patch",
	Short: "Increment the last version number",
	Long:  `Create a new tag with the last version number incremented by 1`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		r, err := git.PlainOpen(path)
		if err != nil {
			log.Fatal().Err(err).Msg("error opening repo")
		}
		repo = r
		currentVersion, err := getLatestVersion(repo)
		if err != nil {
			log.Fatal().Err(err).Msg("error getting latest version")
		}
		newVersion := version{
			Major: currentVersion.Major,
			Minor: currentVersion.Minor,
			Patch: currentVersion.Patch + 1,
			Label: label,
		}
		p := tea.NewProgram(initialModel(newVersion))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	},
}

var minorVersionCmd = &cobra.Command{
	Use:   "minor",
	Short: "Increment the middle/minor version number",
	Long: `Create a new tag with the middle/minor version number incremented by 1
				and the last version number reset to 0`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		r, err := git.PlainOpen(path)
		if err != nil {
			log.Fatal().Err(err).Msg("error opening repo")
		}
		repo = r
		currentVersion, err := getLatestVersion(repo)
		if err != nil {
			log.Fatal().Err(err).Msg("error getting latest version")
		}
		newVersion := version{
			Major: currentVersion.Major,
			Minor: currentVersion.Minor + 1,
			Patch: 0,
		}
		p := tea.NewProgram(initialModel(newVersion))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	},
}

var majorVersionCmd = &cobra.Command{
	Use:   "major",
	Short: "Increment the first/major version number",
	Long: `Create a new tag with the first/major version number incremented by 1
				and the rest of the version numbers reset to 0`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		r, err := git.PlainOpen(path)
		if err != nil {
			log.Fatal().Err(err).Msg("error opening repo")
		}
		repo = r
		currentVersion, err := getLatestVersion(repo)
		if err != nil {
			log.Fatal().Err(err).Msg("error getting latest version")
		}
		newVersion := version{
			Major: currentVersion.Major + 1,
			Minor: 0,
			Patch: 0,
		}
		p := tea.NewProgram(initialModel(newVersion))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	},
}

func updateVersion(repo *git.Repository, v version, annotation string) error {
	ref, err := repo.Head()
	if err != nil {
		return err
	}
	opts := &git.CreateTagOptions{
		Message: annotation,
	}
	err = opts.Validate(repo, ref.Hash())
	if err != nil {
		return err
	}

	_, err = repo.CreateTag(v.String(), ref.Hash(), &git.CreateTagOptions{
		Message: annotation,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Created tag %s", v.String())
	return nil
}

func getLatestVersion(repo *git.Repository) (*version, error) {
	// get the current version
	var currentVersion version
	iter, err := repo.Tags()
	if err != nil {
		return nil, err
	}
	if err := iter.ForEach(func(ref *plumbing.Reference) error {
		h := ref.Hash()
		obj, err := repo.TagObject(h)
		switch err {
		case nil:
			// Tag object present
			tagNameParts := strings.Split(obj.Name, ".")
			if len(tagNameParts) == 3 {
				mjr, err := strconv.Atoi(strings.TrimPrefix(tagNameParts[0], "v"))
				if err != nil {
					return fmt.Errorf("error parsing major version: %w", err)
				}
				mnr, err := strconv.Atoi(tagNameParts[1])
				if err != nil {
					return fmt.Errorf("error parsing minor version: %w", err)
				}
				patchParts := strings.Split(tagNameParts[2], "-")
				patch, err := strconv.Atoi(patchParts[0])
				if err != nil {
					return fmt.Errorf("error parsing patch version: %w", err)
				}
				var lbl string
				if len(patchParts) > 1 {
					lbl = patchParts[1]
				}
				v := version{
					Major: mjr,
					Minor: mnr,
					Patch: patch,
					Label: lbl,
				}
				if v.isGreaterThan(currentVersion) {
					currentVersion = v
				}
			}
			return nil
		case plumbing.ErrObjectNotFound:
			// Not a tag object
			return nil
		default:
			// Some other error
			return err
		}
	}); err != nil {
		// Handle outer iterator error
		return nil, err

	}
	return &currentVersion, nil
}

func getLastCommitMessage(repo *git.Repository) (string, error) {
	ref, err := repo.Head()
	if err != nil {
		return "", err
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", err
	}
	return commit.Message, nil
}
