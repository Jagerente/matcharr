package domain

type Preset struct {
	ID          string
	Title       map[string]string
	Description map[string]string
	Icon        string
	MediaType   MediaType
	Source      string
	Endpoint    string
	Params      map[string]string
	Order       int
	Hidden      bool
}
