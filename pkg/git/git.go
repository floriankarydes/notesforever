package git

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v55/github"
	"github.com/keybase/go-keychain"
	"github.com/pkg/errors"
)

type Repo struct {
	dir   string
	url   string
	token string
}

const DirPerm = 0755

// Pull Git repository at dir. If dir is empty, clone repository. If url is empty, create GitHub repository using Base(dir) as name.
func Open(dir, url string) (*Repo, error) {
	var err error

	// Get token.
	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		token, err = getGhTokenFromKeychain()
		if err != nil {
			return nil, err
		}
	}

	r := &Repo{
		dir:   dir,
		url:   url,
		token: token,
	}

	if err = r.Pull(); err == nil {
		return r, nil
	}
	log.Printf("failed to pull repo: %s; try to clone", err.Error())

	if err = r.create(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Repo) Dir() string {
	return r.dir
}

func (r *Repo) URL() string {
	return r.url
}

// Clean all changes Git repository.
func (r *Repo) Clean() error {
	gitRepo, err := git.PlainOpen(r.dir)
	if err != nil {
		return err
	}
	w, err := gitRepo.Worktree()
	if err != nil {
		return err
	}
	if err := w.Clean(&git.CleanOptions{Dir: true}); err != nil {
		return err
	}
	return nil
}

// Pull all changes Git repository.
func (r *Repo) Pull() error {
	gitRepo, err := git.PlainOpen(r.dir)
	if err != nil {
		return err
	}
	w, err := gitRepo.Worktree()
	if err != nil {
		return err
	}
	err = w.Pull(&git.PullOptions{Auth: r.auth()})
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}
	if err != nil {
		return err
	}
	ref, err := gitRepo.Head()
	if err != nil {
		return err
	}
	commit, err := gitRepo.CommitObject(ref.Hash())
	if err != nil {
		return err
	}

	fmt.Println(commit)
	return nil
}

// Push all changes Git repository.
func (r *Repo) Push() error {
	// Opens an already existing repository.
	gitRepo, err := git.PlainOpen(r.dir)
	if err != nil {
		return err
	}
	w, err := gitRepo.Worktree()
	if err != nil {
		return err
	}

	// Commit all files.
	_, err = w.Add(".")
	if err != nil {
		return err
	}
	commit, err := w.Commit(time.Now().String(), &git.CommitOptions{})
	if err != nil {
		return err
	}
	obj, err := gitRepo.CommitObject(commit)
	if err != nil {
		return err
	}

	// Push to remote.
	if err := gitRepo.Push(&git.PushOptions{Auth: r.auth()}); err != nil {
		return err
	}

	fmt.Println(obj)
	return nil
}

// Create & clone Git repository.
func (r *Repo) create() error {
	var err error

	var name, org string
	if r.url != "" {
		// Try to clone now if URL is defined.
		if err = r.clone(); err == nil {
			return nil
		}
		log.Printf("failed to clone repo: %s; try to create", err.Error())
		name = filepath.Base(r.url)
		org = filepath.Base(filepath.Dir(r.url))
	} else {
		name = filepath.Base(r.dir)
		org = ""
	}

	// Connect to GitHub.
	client := github.NewClient(nil).WithAuthToken(r.token)
	ctx := context.Background()
	ghUser, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return err
	}

	// Clone if repository already exists.
	ghRepo, _, err := client.Repositories.Get(ctx, *ghUser.Login, name)
	if err == nil {
		r.url = *ghRepo.CloneURL
		return r.clone()
	}
	log.Printf("failed to get repo: %s", err)

	// Create repository if it does not exist.
	private := true
	autoInit := true
	ghRepo = &github.Repository{Name: &name, Private: &private, AutoInit: &autoInit}
	ghRepo, _, err = client.Repositories.Create(ctx, org, ghRepo)
	if err != nil {
		return err
	}
	r.url = *ghRepo.CloneURL

	// Clone repository.
	return r.clone()
}

// Clone Git repository.
func (r *Repo) clone() error {
	if err := os.Rename(r.dir, r.saveDir()); err != nil {
		return errors.Wrap(err, "failed to save existing directory")
	}
	_, err := git.PlainClone(r.dir, false, &git.CloneOptions{
		Auth:     r.auth(),
		URL:      r.url,
		Progress: os.Stdout,
	})
	if err != nil {
		return errors.Wrap(err, "failed to clone repository")
	}
	return nil
}

func (r *Repo) saveDir() string {
	return "notesforever_GitBackup_" + time.Now().Format("20060102150405")
}

func (r *Repo) auth() *http.BasicAuth {
	return &http.BasicAuth{
		Username: "abc123", // yes, this can be anything except an empty string
		Password: r.token,
	}
}

func getGhTokenFromKeychain() (string, error) {
	service := "git:https://github.com"
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(service)
	query.SetAccount(user.Username)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)
	results, err := keychain.QueryItem(query)
	if err != nil {
		return "", err
	}
	if len(results) != 1 {
		return "", errors.New("several token found")
	}
	return string(results[0].Data), nil
}
