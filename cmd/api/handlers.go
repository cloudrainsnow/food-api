package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/food/internal/data"
	"github.com/go-chi/chi/v5"
	"github.com/mozillazg/go-slugify"
)

var staticPath = "./static/"

// jsonResponse is the type used for generic JSON responses
type jsonResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type envelope map[string]interface{}

// Login is the handler used to attempt to log a user into the api
func (app *application) Login(w http.ResponseWriter, r *http.Request) {
	type credentials struct {
		UserName string `json:"email"`
		Password string `json:"password"`
	}

	var creds credentials
	var payload jsonResponse

	err := app.readJSON(w, r, &creds)
	if err != nil {
		app.errorLog.Println(err)
		payload.Error = true
		payload.Message = "invalid json supplied, or json missing entirely"
		_ = app.writeJSON(w, http.StatusBadRequest, payload)
	}

	// look up the user by email
	user, err := app.models.User.GetByEmail(creds.UserName)
	if err != nil {
		app.errorJSON(w, errors.New("invalid username/password"))
		return
	}

	// validate the user's password
	validPassword, err := user.PasswordMatches(creds.Password)
	if err != nil || !validPassword {
		app.errorJSON(w, errors.New("invalid username/password"))
		return
	}

	// make sure user is active
	if user.Active == 0 {
		app.errorJSON(w, errors.New("user is not active"))
		return
	}

	// we have a valid user, so generate a token
	token, err := app.models.Token.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// save it to the database
	err = app.models.Token.Insert(*token, *user)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// send back a response
	payload = jsonResponse{
		Error:   false,
		Message: "logged in",
		Data:    envelope{"token": token, "user": user},
	}

	err = app.writeJSON(w, http.StatusOK, payload)
	if err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) Logout(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Token string `json:"token"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, errors.New("invalid json"))
		return
	}

	err = app.models.Token.DeleteByToken(requestPayload.Token)
	if err != nil {
		app.errorJSON(w, errors.New("invalid json"))
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "logged out",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

// AllUsers is the handler which lists all users. Note that this
// handler should be protected in the routes file, and require that
// the user have a valid token
func (app *application) AllUsers(w http.ResponseWriter, r *http.Request) {
	var users data.User
	all, err := users.GetAll()
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "success",
		Data:    envelope{"users": all},
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// EditUser saves a new user, or updates a user, in the database
func (app *application) EditUser(w http.ResponseWriter, r *http.Request) {
	var user data.User
	err := app.readJSON(w, r, &user)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if user.ID == 0 {
		// add user
		if _, err := app.models.User.Insert(user); err != nil {
			app.errorJSON(w, err)
			return
		}
	} else {
		// editing user
		u, err := app.models.User.GetOne(user.ID)
		if err != nil {
			app.errorJSON(w, err)
			return
		}

		u.Email = user.Email
		u.FirstName = user.FirstName
		u.LastName = user.LastName
		u.Active = user.Active

		if err := u.Update(); err != nil {
			app.errorJSON(w, err)
			return
		}

		// if password != string, update password
		if user.Password != "" {
			err := u.ResetPassword(user.Password)
			if err != nil {
				app.errorJSON(w, err)
				return
			}
		}
	}

	payload := jsonResponse{
		Error:   false,
		Message: "Changes saved",
	}

	_ = app.writeJSON(w, http.StatusAccepted, payload)
}

// GetUser returns one user as JSON
func (app *application) GetUser(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	user, err := app.models.User.GetOne(userID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	_ = app.writeJSON(w, http.StatusOK, user)
}

// DeleteUser delets a user from the users table by the id given in the supplied JSON file
func (app *application) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		ID int `json:"id"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	err = app.models.User.DeleteByID(requestPayload.ID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "User deleted",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

// LogUserOutAndSetInactive sets the user specified by the id value in the supplied JSON
// to inactive, and deletes any tokens associated with that user id from the tokens table
// in the database
func (app *application) LogUserOutAndSetInactive(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	user, err := app.models.User.GetOne(userID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	user.Active = 0
	err = user.Update()
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// delete tokens for user
	err = app.models.Token.DeleteTokensForUser(userID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "user logged out and set to inactive",
	}

	_ = app.writeJSON(w, http.StatusAccepted, payload)
}

// ValidateToken accepts a JSON payload with a plain text token, and returns
// true if that token is valid, or false if it is not, as a JSON response
func (app *application) ValidateToken(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Token string `json:"token"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	valid := false
	valid, _ = app.models.Token.ValidToken(requestPayload.Token)

	payload := jsonResponse{
		Error: false,
		Data:  valid,
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

// AllFoods returns all foods as JSON
func (app *application) AllFoods(w http.ResponseWriter, r *http.Request) {
	foods, err := app.models.Food.GetAll()
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "success",
		Data:    envelope{"foods": foods},
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// OneFood returns one food as JSON, by slug
func (app *application) OneFood(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	food, err := app.models.Food.GetOneBySlug(slug)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error: false,
		Data:  food,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// CountriesAll returns a list of all countries consisting of country id and country name, as JSON
func (app *application) CountriesAll(w http.ResponseWriter, r *http.Request) {
	all, err := app.models.Country.All()
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	type selectData struct {
		Value int    `json:"value"`
		Text  string `json:"text"`
	}

	var results []selectData

	for _, x := range all {
		country := selectData{
			Value: x.ID,
			Text:  x.CountryName,
		}

		results = append(results, country)
	}

	payload := jsonResponse{
		Error: false,
		Data:  results,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// EditFood accepts Food as JSON and makes DB calls to either insert or update Food
func (app *application) EditFood(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		ID           int    `json:"id"`
		KnownAs      string `json:"known_as"`
		CountryID    int    `json:"country_id"`
		MakeYear     int    `json:"make_year"`
		Description  string `json:"description"`
		SampleBase64 string `json:"sample"`
		TasteIDs     []int  `json:"taste_ids"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	food := data.Food{
		ID:          requestPayload.ID,
		KnownAs:     requestPayload.KnownAs,
		CountryID:   requestPayload.CountryID,
		MakeYear:    requestPayload.MakeYear,
		Description: requestPayload.Description,
		Slug:        slugify.Slugify(requestPayload.KnownAs),
		TasteIDs:    requestPayload.TasteIDs,
	}

	if len(requestPayload.SampleBase64) > 0 {
		// we have a sample
		decoded, err := base64.StdEncoding.DecodeString(requestPayload.SampleBase64)
		if err != nil {
			app.errorJSON(w, err)
			return
		}

		// write image to /static/samples
		if err := os.WriteFile(fmt.Sprintf("%s/samples/%s.jpg", staticPath, food.Slug), decoded, 0666); err != nil {
			app.errorJSON(w, err)
			return
		}
	}

	if food.ID == 0 {
		// adding a food
		_, err := app.models.Food.Insert(food)
		if err != nil {
			app.errorJSON(w, err)
			return
		}
	} else {
		// updating a food
		err := food.Update()
		if err != nil {
			app.errorJSON(w, err)
			return
		}
	}

	payload := jsonResponse{
		Error:   false,
		Message: "Changes saved",
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

// FoodByID returns one food as JSON, by ID
func (app *application) FoodByID(w http.ResponseWriter, r *http.Request) {
	foodID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	food, err := app.models.Food.GetOneById(foodID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error: false,
		Data:  food,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// DeleteFood accepts an ID and calls DB to delete one Food
func (app *application) DeleteFood(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		ID int `json:"id"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	err = app.models.Food.DeleteByID(requestPayload.ID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "Food deleted",
	}

	app.writeJSON(w, http.StatusOK, payload)

}
