package controller

import (
	"fmt"
	"github.com/caarlos0/httperr"
	"github.com/gorilla/mux"
	"html/template"
	"io/fs"
	"net/http"
	"strarcharts/internal/cache"
	"strarcharts/internal/github"
)

const (
	CHART_WIDTH  = 1024
	CHART_HEIGHT = 400
)

func GetRepo(fsys fs.FS, gh *github.GitHub, cache *cache.Redis, version string) http.Handler {
	repositoryTemplate, err := template.ParseFS(fsys, base, repository)

	if err != nil {
		panic(err)
	}
	indexTemplate, err := template.ParseFS(fsys, base, index)
	if err != nil {
		panic(err)
	}

	return httperr.NewF(func(w http.ResponseWriter, r *http.Request) error {
		name := fmt.Sprintf(
			"%s/%s",
			mux.Vars(r)["owner"],
		)
		details, err := gh.RepoDetails(r.Context(), name)
		if err != nil {
			return indexTemplate.Execute(w, map[string]error{
				"Error": err,
			})
		}
		return repositoryTemplate.Execute(w, map[string]interface{}{
			"Version": version,
			"Details": details,
		})
	})
}
