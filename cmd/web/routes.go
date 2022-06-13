package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *Config) routes() http.Handler {
	mux := chi.NewRouter()

	//recoverer biar panic auto recover sendiri
	mux.Use(middleware.Recoverer)
	
	mux.Get("/", app.Handler)

	return mux
}