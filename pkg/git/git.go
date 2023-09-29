package git

import "log"

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
	defer r.enableLfs()

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

// Discard all changes Git repository.
func (r *Repo) Discard() error {
	panic("not implemented")
}

// Pull all changes Git repository.
func (r *Repo) Pull() error {
	panic("not implemented")
}

// Push all changes Git repository.
func (r *Repo) Push() error {
	panic("not implemented")
}

// Clone Git repository.
func (r *Repo) clone() error {
	//TODO
	panic("not implemented")
}

// Create & clone Git repository.
func (r *Repo) create() error {
	//TODO
	panic("not implemented")
}

// Set Git LFS.
func (r *Repo) enableLfs() error {
	if r == nil {
		return nil
	}
	//TODO
	panic("not implemented")
}
