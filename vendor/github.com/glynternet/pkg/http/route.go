package http

// Route holds all values for registering with the Router
type Route struct {
	Name       string
	Method     string
	Pattern    string
	AppHandler AppJSONHandler
}
