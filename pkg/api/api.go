package api

type Group struct {
	Name     string   `json:"name"`
	Contents Contents `json:"contents"`
}

type Contents struct {
	Refs     []string `json:"refs"`
	Items    []string `json:"items"`
	Requires []string `json:"requires"`
}
