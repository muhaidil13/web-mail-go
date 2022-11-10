package main

import (
	"errors"
	"final-project/data"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/phpdave11/gofpdf"
	"github.com/phpdave11/gofpdf/contrib/gofpdi"
)

func (app *Config) HomePage(w http.ResponseWriter, r *http.Request) {
	app.Render(w, r, "home.page.gohtml", nil)
}
func (app *Config) LoginPage(w http.ResponseWriter, r *http.Request) {
	app.Render(w, r, "login.page.gohtml", nil)
}

func (app *Config) PostLoginPage(w http.ResponseWriter, r *http.Request) {
	_ = app.Session.RenewToken(r.Context())

	// parse form
	err := r.ParseForm()
	if err != nil {
		app.ErrorLog.Println(err)
	}

	// get email and password
	email := r.Form.Get("email")
	passsword := r.Form.Get("password")

	user, err := app.Models.User.GetByEmail(email)

	if err != nil {
		app.Session.Put(r.Context(), "error", "invalid credentials.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Cek password
	validpassword, err := user.PasswordMatches(passsword)
	if err != nil {
		app.Session.Put(r.Context(), "error", "invalid credentials.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if !validpassword {
		msg := Message{
			To:      email,
			Subject: "fail login in attempt",
			Data:    "invalid login attempt",
		}
		app.sendMail(msg)
		app.Session.Put(r.Context(), "error", "invalid credentials.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// login user success
	app.Session.Put(r.Context(), "userID", user.ID)
	app.Session.Put(r.Context(), "user", user)
	app.Session.Put(r.Context(), "flash", "Successful login")
	// redirect
	http.Redirect(w, r, "/", http.StatusSeeOther)

}
func (app *Config) Logout(w http.ResponseWriter, r *http.Request) {
	_ = app.Session.Destroy(r.Context())
	_ = app.Session.RenewToken(r.Context())

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (app *Config) RegisterPage(w http.ResponseWriter, r *http.Request) {
	app.Render(w, r, "register.page.gohtml", nil)
}

func (app *Config) PostRegisterPage(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.ErrorLog.Println(err)
	}
	// TODO-Validate data

	// create a user
	u := data.User{
		Email:     r.Form.Get("email"),
		FirstName: r.Form.Get("first-name"),
		LastName:  r.Form.Get("last-name"),
		Password:  r.Form.Get("password"),
		Active:    0,
		IsAdmin:   0,
	}
	_, err = u.Insert(u)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Unable to Create a user")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}
	// send an activation email
	url := fmt.Sprintf("http://localhost/activate?email=%s", u.Email)
	signurl := GenerateTokenFromString(url)
	app.InfoLog.Println(signurl)
	msg := Message{
		To:       u.Email,
		Subject:  "Activate Your Accounts",
		Template: "confirmation-email",
		Data:     template.HTML(signurl),
	}
	app.sendMail(msg)
	app.Session.Put(r.Context(), "flash", "Confirm in Your email")
	http.Redirect(w, r, "/login", http.StatusSeeOther)

}

func (app *Config) ActivateAccounts(w http.ResponseWriter, r *http.Request) {
	// 	validate url
	url := r.RequestURI
	app.InfoLog.Println(url)
	testUrl := fmt.Sprintf("http://localhost%s", url)
	okay := VerifyToken(testUrl)

	if !okay {
		app.Session.Put(r.Context(), "error", "Invalid Token")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Activate acount
	u, err := app.Models.User.GetByEmail(r.URL.Query().Get("email"))
	if err != nil {
		app.Session.Put(r.Context(), "error", "User Not Found")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	u.Active = 1
	err = u.Update()
	if err != nil {
		app.Session.Put(r.Context(), "error", "Unable to update users")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	app.Session.Put(r.Context(), "flash", "You can login now")
	http.Redirect(w, r, "/login", http.StatusSeeOther)

	// generate invoice

	// send an email with attachment

	// send an email with invioce attachment

	// subcribe the user to an account
}

func (app *Config) ChooseSubcription(w http.ResponseWriter, r *http.Request) {
	// sudah dihandle oleh middleware
	plan, err := app.Models.Plan.GetAll()
	if err != nil {
		app.ErrorLog.Println(err)
		return
	}
	dataMap := make(map[string]any)
	dataMap["plans"] = plan
	app.Render(w, r, "plans.page.gohtml", &TemplateData{
		Data: dataMap,
	})

}

func (app *Config) SubsribeToPlan(w http.ResponseWriter, r *http.Request) {
	// get id of the plan that is choosen
	id := r.URL.Query().Get("id")
	planID, _ := strconv.Atoi(id)

	// get plan from the database
	plan, err := app.Models.Plan.GetOne(planID)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Unable to update plan")
		http.Redirect(w, r, "/members/plans", http.StatusSeeOther)
		return
	}

	// get user from the session
	user, ok := app.Session.Get(r.Context(), "user").(data.User)
	if !ok {
		app.Session.Put(r.Context(), "error", "Unable to update users")
		http.Redirect(w, r, "/members/plans", http.StatusSeeOther)
		return
	}

	app.Wait.Add(1)
	go func() {
		defer app.Wait.Done()

		invoice, err := app.getInvoice(user, plan)
		if err != nil {
			// send to the channel
			app.ErrorChan <- err
		}
		msg := Message{
			To:       user.Email,
			Subject:  "Your Invoice",
			Data:     invoice,
			Template: "invoice",
		}
		app.sendMail(msg)
	}()

	// generate a manual
	app.Wait.Add(1)
	go func() {
		defer app.Wait.Done()

		pdf := app.generateManual(user, plan)
		err = pdf.OutputFileAndClose(fmt.Sprintf("./tmp/%d_manual.pdf", user.ID))
		if err != nil {
			app.ErrorChan <- err
			return
		}
		msg := Message{
			To:      user.Email,
			Subject: "Your Manual",
			Data:    "Your User Manual Attached",
			AttachmentMap: map[string]string{
				"Manual.pdf": fmt.Sprintf("./tmp/%d_manual.pdf", user.ID),
			},
		}

		app.sendMail(msg)

		// test app error chan

		app.ErrorChan <- errors.New("Some Error")
	}()
	// subcribe the user to an account
	err = app.Models.Plan.SubscribeUserToPlan(user, *plan)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Error To Subcribe a plan")
		http.Redirect(w, r, "/members/plan", http.StatusSeeOther)
		return
	}

	u, err := app.Models.User.GetOne(user.ID)
	if err != nil {
		app.Session.Put(r.Context(), "error", "Error To Get user ")
		http.Redirect(w, r, "/members/plan", http.StatusSeeOther)
		return
	}

	app.Session.Put(r.Context(), "user", u)

	// redicret
	app.Session.Put(r.Context(), "flash", "Subscribe")
	http.Redirect(w, r, "/members/plans", http.StatusSeeOther)
}

func (app *Config) generateManual(user data.User, plan *data.Plan) *gofpdf.Fpdf {
	pdf := gofpdf.New("p", "mm", "Letter", "")
	pdf.SetMargins(10, 13, 10)

	importer := gofpdi.NewImporter()

	time.Sleep(5 * time.Second)

	t := importer.ImportPage(pdf, "./pdf/manual.pdf", 1, "/MediaBox")
	pdf.AddPage()

	importer.UseImportedTemplate(pdf, t, 0, 0, 215.9, 0)

	pdf.SetX(75)
	pdf.SetY(150)

	pdf.SetFont("Arial", "", 12)

	pdf.MultiCell(0, 4, fmt.Sprintf("%s %s", user.FirstName, user.LastName), "", "C", false)

	pdf.Ln(5)

	pdf.MultiCell(0, 4, fmt.Sprintf("%s User Guide", plan.PlanName), "", "C", false)

	return pdf

}

func (app *Config) getInvoice(u data.User, plan *data.Plan) (string, error) {
	return plan.PlanAmountFormatted, nil
}
