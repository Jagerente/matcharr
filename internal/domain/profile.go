package domain

type QualityProfile struct {
	ID   int
	Name string
}

type QualityProfiles struct {
	Profiles  []QualityProfile
	DefaultID int
}
