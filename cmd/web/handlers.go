package main

import (
	"errors"
	"fmt"
	"gosubs/data"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/phpdave11/gofpdf"
	"github.com/phpdave11/gofpdf/contrib/gofpdi"
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


func(app *Config) ChooseSubscription(w http.ResponseWriter, r *http.Request) {
	if !app.Session.Exists(r.Context(), "userID") {
		app.Session.Put(r.Context(), "warning", "You must login before see this page")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	plans, err := app.Models.Plan.GetAll()
	if err != nil {
		app.ErrorLog.Println(err)
	}

	data := map[string]any{
		"plans": plans,
	}

	app.render(w, r, "plans.page.gohtml", &TemplateData{
		Data: data,
	})
}


func(app *Config) SubscribeToPlan(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	planId, err := strconv.Atoi(id)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Unknown type of id")
		http.Redirect(w, r, "/members/plans", http.StatusSeeOther)
		return
	}

	plan, err := app.Models.Plan.GetById(planId)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Unable to find plan")
		http.Redirect(w, r, "/members/plans", http.StatusSeeOther)
		return
	}

	user, ok := app.Session.Get(r.Context(), "user").(data.User)
	if !ok {
		app.Session.Put(r.Context(), "error", "Please log in first")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	app.Wg.Add(1)
	go func() {
		defer app.Wg.Done()
		invoice, err := app.getInvoice(user, plan)
		if err != nil {
			app.ErrorChan <- err
			return
		}

		msg := Message{
			To: user.Email,
			Subject: "Your invoice",
			Data: invoice,
			Template: "invoice",
		}

		app.sendMail(msg)
	}()

	app.Wg.Add(1)
	go func() {
		defer app.Wg.Done()

		pdf := app.generateManualPDF(user, plan)
		err := pdf.OutputFileAndClose(fmt.Sprintf("./tmp/%d_manual.pdf", user.ID))
		if err != nil {
			app.ErrorChan <- err
			return
		}

		msg := Message{
			To: user.Email,
			Subject: "Your Manual",
			Data: "Your user manual is attached",
			AttachmentMap: map[string]string{
				"Manual.pdf": fmt.Sprintf("./tmp/%d_manual.pdf", user.ID),
			},
		}
		
		app.sendMail(msg)

		//test error
		app.ErrorChan <- errors.New("some custom error")
	}()

	app.Wg.Wait()

	err = app.Models.Plan.SubscribeUserToPlan(user, *plan)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Error subscribing to that plan!")
		http.Redirect(w, r, "/members/plans", http.StatusSeeOther)
		return
	}

	currentUser, err := app.Models.User.GetOne(user.ID)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Error getting current user!")
		http.Redirect(w, r, "/members/plans", http.StatusSeeOther)
		return
	} 

	app.Session.Put(r.Context(), "user", currentUser)

	app.Session.Put(r.Context(), "flash", "Subscribed")
	http.Redirect(w, r, "/members/plans", http.StatusSeeOther)
}

func (app *Config) getInvoice(user data.User, plan *data.Plan) (string, error) {
	return plan.PlanAmountFormatted, nil
}

func (app *Config) generateManualPDF(user data.User, plan *data.Plan) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(10, 13, 10)

	importer := gofpdi.NewImporter()

	//simulasi kalo nge create nya lama
	time.Sleep(5 * time.Second)

	template := importer.ImportPage(pdf, "./pdf/manual.pdf", 1, "/MediaBox")
	pdf.AddPage()

	//215.9 --> biar ke tengah
	importer.UseImportedTemplate(pdf, template, 0, 0, 215.9, 0)

	pdf.SetX(75)
	pdf.SetY(150)

	pdf.SetFont("Arial", "", 12)
	pdf.MultiCell(0, 4, fmt.Sprintf("%s %s", user.FirstName, user.LastName), "", "C", false)
	pdf.Ln(5)
	pdf.MultiCell(0, 4, fmt.Sprintf("%s User Guide", plan.PlanName), "", "C", false)

	return pdf
}