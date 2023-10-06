package git

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v55/github"
	"github.com/keybase/go-keychain"
)

type Repo struct {
	dir string
	url string
}

const DirPerm = 0755

// Pull Git repository at dir. If dir is empty, clone repository. If url is empty, create GitHub repository using Base(dir) as name.
func New(dir, url string) (*Repo, error) {
	r := &Repo{
		dir: dir,
		url: url,
	}

	if err := r.Pull(); err == nil {
		return r, nil
	} else {
		log.Printf("cannot pull, trying to clone; err: %s", err.Error())
	}

	if err := r.clone(); err == nil {
		return r, nil
	} else {
		log.Printf("cannot clone, trying to create; err: %s", err.Error())
	}

	if err := r.create(); err != nil {
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
	if err = w.Pull(&git.PullOptions{RemoteName: "origin"}); err != nil {
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
	fmt.Println(obj)

	// Push to remote.
	if err := gitRepo.Push(&git.PushOptions{}); err != nil {
		return err
	}

	return nil
}

// Clone Git repository.
func (r *Repo) clone() error {
	_, err := git.PlainClone(r.dir, false, &git.CloneOptions{
		URL:      r.url,
		Progress: os.Stdout,
	})
	return err
}

// Create & clone Git repository.
func (r *Repo) create() error {

	// Connect to GitHub.
	token, err := getGhTokenFromKeychain()
	if err != nil {
		return err
	}
	client := github.NewClient(nil).WithAuthToken(token)

	// Create repository.
	var name, org string
	if r.url != "" {
		name = filepath.Base(r.dir)
		org = filepath.Base(filepath.Dir(r.dir))
	} else {
		name = filepath.Base(r.dir)
		org = ""
	}
	private := true
	desc := "NotesForever backup repository"
	autoInit := true
	ghRepo := &github.Repository{Name: &name, Private: &private, Description: &desc, AutoInit: &autoInit}
	ctx := context.Background()
	ghRepo, _, err = client.Repositories.Create(ctx, org, ghRepo)
	if err != nil {
		return err
	}
	r.url = *ghRepo.CloneURL

	// Clone repository.
	if err := r.clone(); err != nil {
		return err
	}

	return nil
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
	//query.SetAccessGroup(accessGroup)
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
