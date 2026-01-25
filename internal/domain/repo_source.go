package domain

// RepoSource represents parsed repository source information
type RepoSource struct {
	Branch   string
	IsRemote bool
	Owner    string
	Path     string
	Repo     string
}
