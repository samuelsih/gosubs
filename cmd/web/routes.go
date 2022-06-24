package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *Config) routes() http.Handler {
	mux := chi.NewRouter()

	mux.Use(middleware.Recoverer) //recoverer biar panic auto recover sendiri
	mux.Use(app.SessionLoad)

	mux.Get("/", app.HomePage)
	mux.Get("/login", app.LoginPage)
	mux.Post("/login", app.HandleLogin)
	mux.Get("/logout", app.Logout)
	mux.Get("/register", app.RegisterPage)
	mux.Post("/register", app.HandleRegister)
	mux.Get("/activate", app.ActivateAccount)
	mux.Get("/plans", app.ChooseSubscription)

	mux.Mount("/members", app.authRoutes())

	return mux
}

func (app *Config) authRoutes() http.Handler {
	mux := chi.NewRouter()

	mux.Get("/plans", app.ChooseSubscription)
	mux.Get("/subscribe", app.SubscribeToPlan)

	return mux
}
