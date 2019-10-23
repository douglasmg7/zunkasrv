package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/julienschmidt/httprouter"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

/************************************************************************************************
* Templates
************************************************************************************************/
// Geral.
var tmplMaster, tmplIndex, tmplDeniedAccess *template.Template

// Misc.
var tmplMessage *template.Template

// Aldo.
var tmplAldoProducts, tmplAldoProduct, tmplAldoCategorySel, tmplAldoCategoryUse, tmplAldoCategoryAll *template.Template

// Allnations.
var tmplAllnationsProducts, tmplAllnationsConfig *template.Template

// Auth.
var tmplAuthSignup, tmplAuthSignin, tmplPasswordRecovery, tmplPasswordReset *template.Template

// Student.
var tmplStudent, tmplAllStudent, tmplNewStudent *template.Template

var production bool
var port string

// Db.
var dbApp *sql.DB
var dbAldo *sqlx.DB
var dbAppFile, dbAldoFile string

// list.
var listPath string

var err error

// User.
var tmplUserAdd *template.Template
var tmplUserAccount *template.Template
var tmplUserChangeName *template.Template
var tmplUserChangeEmail *template.Template
var tmplUserChangeMobile *template.Template
var tmplUserChangePassword *template.Template
var tmplUserChangeRG *template.Template
var tmplUserChangeCPF *template.Template
var tmplUserDeleteAccount *template.Template

// Sessions from each user.
var sessions = Sessions{
	mapUserID:      map[string]int{},
	mapSessionData: map[int]*SessionData{},
}

func init() {
	// Config path.
	cfgPath := os.Getenv("ZUNKAPATH")
	if cfgPath == "" {
		panic("Path to config.toml must be dfined on enviroment variable ZUNKAPATH")
	}

	// Config.
	viper.AddConfigPath(cfgPath)
	viper.SetConfigName("config")
	viper.SetDefault("all.logDir", "log")
	viper.SetDefault("all.dbDir", "db")
	viper.SetDefault("all.listDir", "list")
	viper.SetDefault("all.env", "development")
	viper.BindEnv("all.env", "ZUNKAENV")
	err = viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Error reading config file: %s \n", err))
	}

	// Port.
	port = viper.GetString("zunkasrv.port")

	// Path.
	logPath := path.Join(cfgPath, viper.GetString("all.logDir"))
	dbPath := path.Join(cfgPath, viper.GetString("all.dbDir"))
	listPath = path.Join(cfgPath, viper.GetString("all.listDir"))

	// Create log path.
	os.MkdirAll(logPath, os.ModePerm)

	// Db files.
	dbAppFile = path.Join(dbPath, viper.GetString("zunkasrv.dbFileName"))
	dbAldoFile = path.Join(dbPath, viper.GetString("aldowsc.dbFileName"))

	// Log file.
	logFile, err := os.OpenFile(path.Join(logPath, viper.GetString("zunkasrv.logFileName")), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	// Log configuration.
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	log.SetFlags(log.Ldate | log.Lmicroseconds)
	// log.SetFlags(log.Ldate | log.Lmicroseconds | log.Lshortfile)
	// log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	// log.SetFlags(log.LstdFlags | log.Ldate | log.Lshortfile)
	// log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	// production or development mode

	// Env mode.
	if viper.GetString("all.env") == "production" {
		production = true
		log.Println("Running in production mode")
	} else {
		log.Println("Running in development mode")
	}

	/************************************************************************************************
	* Load templates
	************************************************************************************************/
	// Geral.
	tmplMaster = template.Must(template.ParseGlob("templates/master/*"))
	tmplIndex = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/index.tpl"))
	tmplDeniedAccess = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/misc/deniedAccess.tpl"))
	// Misc.
	tmplMessage = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/misc/message.tpl"))
	// Aldo.
	tmplAldoProduct = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/aldo/aldoProduct.tmpl"))
	tmplAldoProducts = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/aldo/aldoProducts.tmpl"))
	tmplAldoCategorySel = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/aldo/aldoCategorySel.tmpl"))
	tmplAldoCategoryUse = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/aldo/aldoCategoryUse.tmpl"))
	tmplAldoCategoryAll = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/aldo/aldoCategoryAll.tmpl"))
	// Allnations.
	tmplAllnationsProducts = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/allnations/allnationsProducts.tpl"))
	tmplAllnationsConfig = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/allnations/allnationsConfig.tpl"))
	// Auth.
	tmplAuthSignup = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/auth/signup.tpl"))
	tmplAuthSignin = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/auth/signin.tpl"))
	tmplPasswordRecovery = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/auth/passwordRecovery.tpl"))
	tmplPasswordReset = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/auth/passwordReset.tpl"))
	// Student.
	tmplStudent = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/student.tpl"))
	tmplAllStudent = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/allStudent.tpl"))
	tmplNewStudent = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/newStudent.tpl"))
	// User.
	tmplUserAdd = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/userAdd.tpl"))
	tmplUserAccount = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/user/userAccount.tpl"))
	tmplUserChangeName = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/user/userChangeName.tpl"))
	tmplUserChangeEmail = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/user/userChangeEmail.tpl"))
	tmplUserChangeMobile = template.Must(template.Must(tmplMaster.Clone()).ParseFiles("templates/user/userChangeMobile.tpl"))

	// debug templates
	// for _, tplItem := range tmplAll["user_add"].Templates() {
	// 	log.Println(tplItem.Name())
	// }
}

func main() {
	// Start data base.
	dbApp, err = sql.Open("sqlite3", dbAppFile)
	if err != nil {
		log.Fatal(fmt.Errorf("Error open db %v: %v", dbAppFile, err))
	}
	defer dbApp.Close()
	err = dbApp.Ping()
	if err != nil {
		log.Fatal(err)
	}

	dbAldo = sqlx.MustConnect("sqlite3", dbAldoFile)
	defer dbAldo.Close()

	// Init router.
	router := httprouter.New()
	router.GET("/ns/favicon.ico", faviconHandler)
	router.GET("/ns/", getSession(indexHandler))

	// Clean the session cache.
	router.GET("/ns/clean-sessions", checkPermission(cleanSessionsHandler, "admin"))

	// Aldo.
	router.GET("/ns/aldo/products", checkPermission(aldoProductsHandler, "admin"))
	router.GET("/ns/aldo/product/:code", checkPermission(aldoProductHandler, "admin"))
	// Create product on zunka server.
	router.POST("/ns/aldo/product/:code", checkPermission(aldoProductHandlerPost, "admin"))
	// Product removed from site, so remove his reference from the site system.
	router.DELETE("/ns/aldo/product/mongodb_id/:code", checkApiAuthorization(aldoProductMongodbIdHandlerDelete))
	router.GET("/ns/aldo/category/sel", checkPermission(aldoCategSelHandler, "admin"))
	router.POST("/ns/aldo/category/sel", checkPermission(aldoCategSelHandlerPost, "admin"))
	router.GET("/ns/aldo/category/use", checkPermission(aldoCategUseHandler, "admin"))
	router.GET("/ns/aldo/category/all", checkPermission(aldoCategAllHandler, "admin"))

	// Allnations.
	router.GET("/allnations/products", checkPermission(allnationsProductsHandler, "admin"))
	router.GET("/allnations/config", checkPermission(allnationsConfigHandler, "admin"))

	// Auth - signup.
	router.GET("/ns/auth/signup", confirmNoLogged(authSignupHandler))
	router.POST("/ns/auth/signup", confirmNoLogged(authSignupHandlerPost))
	router.GET("/ns/auth/signup/confirmation/:uuid", confirmNoLogged(authSignupConfirmationHandler))

	// Auth - signin/signout.
	router.GET("/ns/auth/signin", confirmNoLogged(authSigninHandler))
	router.POST("/ns/auth/signin", confirmNoLogged(authSigninHandlerPost))
	router.GET("/ns/auth/signout", authSignoutHandler)

	// Auth - password.
	router.GET("/ns/auth/password/recovery", confirmNoLogged(passwordRecoveryHandler))
	router.POST("/ns/auth/password/recovery", confirmNoLogged(passwordRecoveryHandlerPost))
	router.GET("/ns/auth/password/reset", confirmNoLogged(passwordResetHandler))

	// User.
	router.GET("/ns/user/account", checkPermission(userAccountHandler, ""))
	router.GET("/ns/user/change/name", checkPermission(userChangeNameHandler, ""))
	router.POST("/ns/user/change/name", checkPermission(userChangeNameHandlerPost, ""))
	router.GET("/ns/user/change/email", checkPermission(userChangeEmailHandler, ""))
	router.POST("/ns/user/change/email", checkPermission(userChangeEmailHandlerPost, ""))
	router.GET("/ns/user/change/email-confirmation/:uuid", checkPermission(userChangeEmailConfirmationHandler, ""))
	router.GET("/ns/user/change/mobile", checkPermission(userChangeMobileHandler, ""))
	router.POST("/ns/user/change/mobile", checkPermission(userChangeMobileHandlerPost, ""))

	// Entrance.
	router.GET("/ns/user_add", userAddHandler)

	// Student.
	router.GET("/ns/student/all", checkPermission(allStudentHandler, "editStudent"))
	router.GET("/ns/student/new", checkPermission(newStudentHandler, "editStudent"))
	router.POST("/ns/student/new", checkPermission(newStudentHandlerPost, "editStudent"))
	router.GET("/ns/student/id/:id", checkPermission(studentByIdHandler, "editStudent"))

	// start server
	router.ServeFiles("/static/*filepath", http.Dir("./static/"))
	log.Println("listen port", port)
	// Why log.Fall work here?
	// log.Fatal(http.ListenAndServe(":"+port, router))
	log.Fatal(http.ListenAndServe(":"+port, newLogger(router)))
}

/**************************************************************************************************
* Middleware
**************************************************************************************************/

// Handle with session.
type handleS func(w http.ResponseWriter, req *http.Request, p httprouter.Params, session *SessionData)

// Get session middleware.
func getSession(h handleS) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		// Get session.
		session, err := sessions.GetSession(req)
		if err != nil {
			log.Fatal(err)
		}
		h(w, req, p, session)
	}
}

// Check permission middleware.
func checkPermission(h handleS, permission string) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		// Get session.
		session, err := sessions.GetSession(req)
		if err != nil {
			log.Fatal(err)
		}
		// Not signed.
		if session == nil {
			http.Redirect(w, req, "/ns/auth/signin", http.StatusSeeOther)
			return
		}
		// Have the permission.
		if permission == "" || session.CheckPermission(permission) {
			h(w, req, p, session)
			return
		}
		// No Permission.
		// fmt.Fprintln(w, "Not allowed")
		data := struct {
			Session     *SessionData
			HeadMessage string
		}{Session: session}
		err = tmplDeniedAccess.ExecuteTemplate(w, "deniedAccess.tpl", data)
		HandleError(w, err)
	}
}

// Check if not logged.
func confirmNoLogged(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		// Get session.
		session, err := sessions.GetSession(req)
		if err != nil {
			log.Fatal(err)
		}
		// Not signed.
		if session == nil {
			h(w, req, p)
			return
		}
		// fmt.Fprintln(w, "Not allowed")
		data := struct{ Session *SessionData }{session}
		err = tmplDeniedAccess.ExecuteTemplate(w, "deniedAccess.tpl", data)
		HandleError(w, err)

	}
}

// Api Authorization.
func checkApiAuthorization(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		user, pass, ok := req.BasicAuth()
		if ok && user == zunkaServerUser() && pass == zunkaServerPass() {
			h(w, req, p)
			return
		}
		log.Printf("Unauthorized access, %v %v, user: %v, pass: %v, ok: %v", req.Method, req.URL.Path, user, pass, ok)
		log.Printf("authorization      , %v %v, user: %v, pass: %v", req.Method, req.URL.Path, zunkaServerUser(), zunkaServerPass())
		// Unauthorised.
		w.Header().Set("WWW-Authenticate", `Basic realm="Please enter your username and password for this service"`)
		w.WriteHeader(401)
		w.Write([]byte("Unauthorised.\n"))
		return
	}
}

/**************************************************************************************************
* Logger middleware
**************************************************************************************************/

// Logger struct.
type logger struct {
	handler http.Handler
}

// Handle interface.
// todo - why DELETE is logging twice?
func (l *logger) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	start := time.Now()
	l.handler.ServeHTTP(w, req)
	log.Printf("%s %s %v", req.Method, req.URL.Path, time.Since(start))
	// log.Printf("header: %v", req.Header)
}

// New logger.
func newLogger(h http.Handler) *logger {
	return &logger{handler: h}
}
