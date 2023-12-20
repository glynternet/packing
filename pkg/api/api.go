package api

type Group struct {
	Name     string   `json:"name"`
	Contents Contents `json:"contents"`
}

type Contents struct {
	GroupKeys []string `json:"group_keys"`
	Items     []string `json:"items"`
}
