package main

import (
	"fmt"
	"gosubs/data"
	"html/template"
	"net/http"
)

func (app *Config) HomePage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "home.page.gohtml", nil)
}

func (app *Config) LoginPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "login.page.gohtml", nil)
}

func (app *Config) HandleLogin(w http.ResponseWriter, r *http.Request) {
	_ = app.Session.RenewToken(r.Context())

	err := r.ParseForm()
	if err != nil {
		app.ErrorLog.Println(err)
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")

	user, err := app.Models.User.GetByEmail(email)
	if err != nil {
		msg := Message {
			To: email,
			Subject: "Failed log in attempt",
			Data: "No user named " + email,
		} 
		app.sendMail(msg)

		app.Session.Put(r.Context(), "error", "invalid credentials!")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	//check password
	validPassword, err := user.PasswordMatches(password)
	if err != nil {
		app.Session.Put(r.Context(), "error", "invalid credentials!")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if !validPassword {
		msg := Message {
			To: email,
			Subject: "Failed log in attempt",
			Data: "Invalid login attempt",
		} 
		app.sendMail(msg)

		app.Session.Put(r.Context(), "error", "wrong password!")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	app.Session.Put(r.Context(), "userID", user.ID)
	app.Session.Put(r.Context(), "user", user) //dihandle pake gob.Register() di initSession

	app.Session.Put(r.Context(), "flash", "Successful login!")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *Config) Logout(w http.ResponseWriter, r *http.Request) {
	//hapus session
	_ = app.Session.Destroy(r.Context())
	_ = app.Session.RenewToken(r.Context())

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (app *Config) RegisterPage(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, "register.page.gohtml", nil)
}

func (app *Config) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.InfoLog.Println(err)
		return
	}

	u := data.User{
		Email: r.Form.Get("email"),
		FirstName: r.Form.Get("first-name"),
		LastName: r.Form.Get("last-name"),
		Password: r.Form.Get("password"),
		Active: 0,
		IsAdmin: 0,
	}

	_, err := u.Insert(u)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Unable to create user")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	//send email activation
	url := fmt.Sprintf("http://localhost/activate?email=%s", u.Email)
	signedUrl := GenerateTokenFromString(url)
	app.InfoLog.Println("Generated Token:", signedUrl)

	msg := Message{
		To: u.Email,
		Subject: "Activate Your Account",
		Template: "confirmation-email",
		Data: template.HTML(signedUrl),
	}

	app.sendMail(msg)

	app.Session.Put(r.Context(), "flash", "Confirmation has been sent to your email")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
	
}

func (app *Config) ActivateAccount(w http.ResponseWriter, r *http.Request) {
	//validate url
	url := r.RequestURI
	testURL := fmt.Sprintf("http://localhost%s", url)

	if ok := VerifyToken(testURL); !ok {
		app.Session.Put(r.Context(), "error", "Invalid token")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user, err := app.Models.User.GetByEmail(r.URL.Query().Get("email"))
	if err != nil {
		app.Session.Put(r.Context(), "error", "No user found!")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user.Active = 1
	err = user.Update()
	if err != nil {
		app.Session.Put(r.Context(), "error", "Unable to update user")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	app.Session.Put(r.Context(), "flash", "Account activated!!")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
