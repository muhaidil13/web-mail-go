package main

import (
	"final-project/data"
	"fmt"
	"net/http"
	"text/template"
	"time"
)

var pathTotemplate = "cmd/web/templates"

type TemplateData struct {
	StringMap     map[string]string
	IntMap        map[string]int
	FloatMap      map[string]float64
	Data          map[string]any
	Flash         string
	Warning       string
	Error         string
	Authenticated bool
	Now           time.Time
	User          *data.User
}

func (app *Config) Render(w http.ResponseWriter, r *http.Request, t string, td *TemplateData) {
	partials := []string{
		fmt.Sprintf("%s/base.layout.gohtml", pathTotemplate),
		fmt.Sprintf("%s/header.partial.gohtml", pathTotemplate),
		fmt.Sprintf("%s/navbar.partial.gohtml", pathTotemplate),
		fmt.Sprintf("%s/footer.partial.gohtml", pathTotemplate),
		fmt.Sprintf("%s/alerts.partial.gohtml", pathTotemplate),
	}
	var templateslice []string
	templateslice = append(templateslice, fmt.Sprintf("%s/%s", pathTotemplate, t))

	for _, val := range partials {
		templateslice = append(templateslice, val)
	}

	if td == nil {
		td = &TemplateData{}
	}

	tmpl, err := template.ParseFiles(templateslice...)
	if err != nil {
		app.ErrorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, app.AddDefaultData(td, r))
	if err != nil {
		app.ErrorLog.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (app *Config) AddDefaultData(td *TemplateData, r *http.Request) *TemplateData {
	td.Flash = app.Session.PopString(r.Context(), "flash")
	td.Warning = app.Session.PopString(r.Context(), "warning")
	td.Error = app.Session.PopString(r.Context(), "error")
	if app.IsAuthenticated(r) {
		td.Authenticated = true
		user, ok := app.Session.Get(r.Context(), "user").(data.User)
		if !ok {
			app.ErrorLog.Println("Can't get user from the session")
		} else {
			td.User = &user
		}

	}
	td.Now = time.Now()
	return td
}

func (app *Config) IsAuthenticated(r *http.Request) bool {
	return app.Session.Exists(r.Context(), "userID")

}
