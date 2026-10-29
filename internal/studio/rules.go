package studio

import (
	"net/http"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func (a *api) rules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"directives":  deck.Directives,
		"layouts":     deck.Layouts,
		"transitions": deck.Transitions,
		"metaKeys":    deck.MetaKeys,
	})
}
