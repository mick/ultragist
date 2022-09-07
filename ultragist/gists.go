package ultragist

import (
	"fmt"
)

func GetGistId(gistId string) {

	fmt.Printf("gist by gistId for %s \n", gistId)
}

func UpdateGist(gist Gist) {

}

func CreateGist(gist Gist) {

}

type Gist struct {
	GistId  string `json:"userId"` // used in the url
	Title   string `json:"title"`  // future use
	Viewers string `json:"viewers"`
	Editors string `json:"editors"`
}
