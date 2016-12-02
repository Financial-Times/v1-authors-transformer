package authors

//model aligned with v2-authors-transformer
type author struct {
	UUID                   string                 `json:"uuid"`
	PrefLabel              string                 `json:"prefLabel"`
	Type                   string                 `json:"type"`
	AlternativeIdentifiers alternativeIdentifiers `json:"alternativeIdentifiers,omitempty"`
	Aliases                []string               `json:"aliases,omitempty"`
}

type alternativeIdentifiers struct {
	TME   []string `json:"TME,omitempty"`
	Uuids []string `json:"uuids,omitempty"`
}

type authorLink struct {
	APIURL string `json:"apiUrl"`
}

type authorUUID struct {
	UUID string `json:"ID"`
}
