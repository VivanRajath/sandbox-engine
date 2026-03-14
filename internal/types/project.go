package types

type Project struct {

	Framework string

	Entrypoint string

	Dependencies []string

	Port int
}