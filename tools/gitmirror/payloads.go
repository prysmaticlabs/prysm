package main

// ReleasePayload event from Github's API containing the
// minimal required information for the tool.
type ReleasePayload struct {
	Action     string     `json:"action"`
	Release    Release    `json:"release"`
	Repository Repository `json:"repository"`
}

// Release object with minimal information from Github's API.
type Release struct {
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish"`
	URL             string `json:"url"`
	Name            string `json:"name"`
}

// Repository object with minimal information from Github's API.
type Repository struct {
	Name string `json:"name"`
}
