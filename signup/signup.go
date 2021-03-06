package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"

	"github.com/dchest/captcha"
	"github.com/gorilla/mux"
)

var baseUrl string = "http://sharelatex"
var c chan string = make(chan string)
var capt map[string]string = make(map[string]string)

func isLoggedIn(c *http.Client) bool {
	resp, _ := c.Get(baseUrl + "/admin/register")
	if resp.StatusCode == 302 {
		return false
	} else {
		return true
	}
}

func login(c *http.Client) {
	if isLoggedIn(c) {
		return
	}

	re := regexp.MustCompile(`window\.csrfToken = "(.*?)"`)
	resp, _ := c.Get(baseUrl + "/login")
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	csrf := re.FindStringSubmatch(string(body))[1]
	resp, _ = c.PostForm(baseUrl+"/login", url.Values{
		"email":    {"pallavagarwal07@gmail.com"},
		"password": {"lambdacalculus"},
		"_csrf":    {csrf},
	})
	body, _ = ioutil.ReadAll(resp.Body)

	if !isLoggedIn(c) {
		panic("Login Failed")
	}
}

func getClient() *http.Client {
	cookieJar, err := cookiejar.New(nil)

	client := &http.Client{
		Jar: cookieJar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New("net/http: use last response")
		},
	}

	if err != nil {
		panic("Error creating Client")
	}

	return client
}

func sendRegistrationMail(client *http.Client, email string) {
	login(client)
	re := regexp.MustCompile(`window\.csrfToken = "(.*?)"`)
	resp, _ := client.Get(baseUrl + "/admin/register")
	body, _ := ioutil.ReadAll(resp.Body)
	csrf := re.FindStringSubmatch(string(body))[1]
	resp, _ = client.PostForm(baseUrl+"/admin/register", url.Values{
		"email": {email},
		"_csrf": {csrf},
	})
}

func isValidIITKMail(email string) bool {
	re := regexp.MustCompile(`^[A-Za-z0-9._%+-]+@iitk.ac.in$`)
	email = strings.TrimSpace(email)
	return re.MatchString(email)
}

func queueHandler(c chan string) {
	client := getClient()
	for true {
		email := <-c
		sendRegistrationMail(client, email)
	}
}

func serveRegister(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	email := r.Form.Get("email")
	retid := r.Form.Get("_csrf")
	cresp := r.Form.Get("captcha")
	str, _ := ioutil.ReadFile("register.html")
	bootstrap, _ := ioutil.ReadFile("bootstrap.min.css")
	css, _ := ioutil.ReadFile("style.css")
	captcha_id := []byte(captcha.New())
	captcha_url := []byte("/register/download/" + string(captcha_id) + ".png")

	str = bytes.Replace(str,
		[]byte("REPLACE_THIS_WITH_BOOTSTRAP"), bootstrap, 1)
	str = bytes.Replace(str,
		[]byte("REPLACE_THIS_WITH_CSS"), css, 1)
	str = bytes.Replace(str,
		[]byte("CAPTCHA"), captcha_id, 1)
	str = bytes.Replace(str,
		[]byte("IMAGE_SRC"), captcha_url, 1)

	if email != "" {
		if retid == "" || cresp == "" {
			str = bytes.Replace(str,
				[]byte(`<!--MESSAGE_HERE-->`),
				[]byte(`<div class="alert alert-danger" role="alert">
					Captcha Response not given by user.</div>`), 1)
		} else if ok := captcha.VerifyString(retid, cresp); !ok {
			str = bytes.Replace(str,
				[]byte(`<!--MESSAGE_HERE-->`),
				[]byte(`<div class="alert alert-danger" role="alert">
					Incorrect captcha response given by user.</div>`), 1)
			captcha.Reload(retid)
		} else if !isValidIITKMail(email) {
			str = bytes.Replace(str,
				[]byte(`<!--MESSAGE_HERE-->`),
				[]byte(`<div class="alert alert-danger" role="alert">
					Only IITK email addresses are allowed</div>`), 1)
			captcha.Reload(retid)
		} else {
			c <- email
			str = bytes.Replace(str,
				[]byte(`<!--MESSAGE_HERE-->`),
				[]byte(`<div class="alert alert-success" role="alert">
					Email Sent</div>`), 1)
			captcha.Reload(retid)
		}
	}
	fmt.Fprintf(w, "%s", str)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/register", serveRegister)
	r.Handle("/register/download/{id}", captcha.Server(300, 100))

	go queueHandler(c)
	http.Handle("/", r)
	http.ListenAndServe(":3001", nil)
}
