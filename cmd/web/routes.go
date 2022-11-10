package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *Config) routes() http.Handler {
	mux := chi.NewRouter()

	// Set Middlware
	mux.Use(middleware.Recoverer)

	mux.Use(app.SessionLoad)

	// create Routes home
	mux.Get("/", app.HomePage)

	// Login
	mux.Get("/login", app.LoginPage)
	mux.Post("/login", app.PostLoginPage)

	mux.Get("/logout", app.Logout)

	// register
	mux.Get("/register", app.RegisterPage)
	mux.Post("/register", app.PostRegisterPage)

	// activate

	mux.Get("/activate", app.ActivateAccounts)

	mux.Mount("/members", app.AuthRouter())

	return mux

}

func (app *Config) AuthRouter() http.Handler {
	mux := chi.NewRouter()
	mux.Use(app.Auth)

	mux.Get("/plans", app.ChooseSubcription)

	mux.Get("/subscribe", app.SubsribeToPlan)
	return mux

}
