package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/douglasmg7/aldoutil"
	"github.com/julienschmidt/httprouter"
)

// Product list.
func aldoProductsHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params, session *SessionData) {
	data := struct {
		Session     *SessionData
		HeadMessage string
		Products    []aldoutil.Product
	}{session, "", []aldoutil.Product{}}

	err = dbAldo.Select(&data.Products, "SELECT * FROM product order by dealer_price LIMIT 100 ")
	HandleError(w, err)

	err = tmplAldoProducts.ExecuteTemplate(w, "aldoProducts.tmpl", data)
	HandleError(w, err)
}

// Product item.
func aldoProductHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params, session *SessionData) {
	data := struct {
		Session     *SessionData
		HeadMessage string
		Product     aldoutil.Product
	}{session, "", aldoutil.Product{}}

	err = dbAldo.Get(&data.Product, "SELECT * FROM product WHERE code=?", ps.ByName("code"))
	HandleError(w, err)

	err = tmplAldoProduct.ExecuteTemplate(w, "aldoProduct.tmpl", data)
	HandleError(w, err)
}

// Add product to zunka site.
func aldoProductHandlerPost(w http.ResponseWriter, req *http.Request, ps httprouter.Params, session *SessionData) {
	data := struct {
		Session     *SessionData
		HeadMessage string
		Product     aldoutil.Product
	}{session, "", aldoutil.Product{}}

	// Get product.
	err = dbAldo.Get(&data.Product, "SELECT * FROM product WHERE code=?", ps.ByName("code"))
	HandleError(w, err)

	// Set store product.
	storeProduct := aldoutil.StoreProduct{}
	storeProduct.DealerName = "Aldo"
	storeProduct.DealerProductId = strconv.Itoa(data.Product.Id)

	// Title.
	storeProduct.DealerProductTitle = strings.Title(strings.ToLower(data.Product.Description))

	// Category.
	storeProduct.DealerProductCategory = strings.Title(strings.ToLower(data.Product.Category))

	// Maker.
	storeProduct.DealerProductMaker = strings.Title(strings.ToLower(data.Product.Brand))

	// Description.
	data.Product.TechnicalDescription = strings.TrimSpace(data.Product.TechnicalDescription)

	// Description.
	// Split by <br/>
	techDescs := regexp.MustCompile(`\<\s*br\s*\/*\>`).Split(data.Product.TechnicalDescription, -1)
	// Remove html tags.
	// Remove blank intens.
	reg := regexp.MustCompile(`\<[^>]*\>`)
	buffer := bytes.Buffer{}
	for _, val := range techDescs {
		val = strings.TrimSpace(reg.ReplaceAllString(val, ""))
		if val != "" {
			buffer.WriteString(strings.ReplaceAll(val, ":", ";"))
			buffer.WriteString("\n")
		}
	}
	storeProduct.DealerProductDesc = strings.TrimSpace(buffer.String())

	// Months in days.
	storeProduct.DealerProductWarrantyDays = data.Product.WarrantyPeriod * 30
	// Length in cm.
	storeProduct.DealerProductDeep = int(math.Ceil(float64(data.Product.Length) / 10))
	// Width in cm.
	storeProduct.DealerProductWidth = int(math.Ceil(float64(data.Product.Width) / 10))
	// Height in cm.
	storeProduct.DealerProductHeight = int(math.Ceil(float64(data.Product.Height) / 10))
	// Weight in grams.
	storeProduct.DealerProductWeight = data.Product.Weight
	// Price.
	storeProduct.DealerProductPrice = int(data.Product.DealerPrice)
	// Last update.
	storeProduct.DealerProductLastUpdate = data.Product.ChangedAt
	// Active.
	storeProduct.DealerProductActive = data.Product.Availability

	// Convert to json.
	reqBody, err := json.Marshal(storeProduct)
	HandleError(w, err)

	// Log request.
	// log.Println("request body: " + string(reqBody))

	// Request product add.
	client := &http.Client{}
	req, err = http.NewRequest("POST", zunkaSiteHost()+"/setup/product/add", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	HandleError(w, err)
	req.SetBasicAuth(zunkaSiteUser(), zunkaSitePass())
	res, err := client.Do(req)
	HandleError(w, err)

	// res, err := http.Post("http://localhost:3080/setup/product/add", "application/json", bytes.NewBuffer(reqBody))
	defer res.Body.Close()
	HandleError(w, err)

	// Result.
	resBody, err := ioutil.ReadAll(res.Body)
	HandleError(w, err)
	// No 200 status.
	if res.StatusCode != 200 {
		HandleError(w, errors.New(fmt.Sprintf("Error ao solicitar a criação do produto no servidor zunka.\n\nstatus: %v\n\nbody: %v", res.StatusCode, string(resBody))))
		// HandleError(w, errors.New(fmt.Fprintf("Error ao solicitar a criação do produto no servidor zunka.\n\nStatus: %v\nError: %v", res.StatusCode, resBody)))
		return
	}
	// Mongodb id from created product.
	data.Product.MongodbId = string(resBody)
	// Remove suround double quotes.
	data.Product.MongodbId = data.Product.MongodbId[1 : len(data.Product.MongodbId)-1]

	// Update product with _id from mongodb store.
	stmt, err := dbAldo.Prepare(`UPDATE product SET mongodbId = $1 WHERE id = $2;`)
	HandleError(w, err)
	defer stmt.Close()
	_, err = stmt.Exec(data.Product.MongodbId, data.Product.Id)
	HandleError(w, err)

	// Render template.
	err = tmplAldoProduct.ExecuteTemplate(w, "aldoProduct.tmpl", data)
	HandleError(w, err)
}

// Remove mongodb id from Product.
func aldoProductMongodbIdHandlerDelete(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	// Update mongodb store id.
	stmt, err := dbAldo.Prepare(`UPDATE product SET mongodbId = $1 WHERE id = $2;`)
	HandleError(w, err)
	defer stmt.Close()
	_, err = stmt.Exec("", ps.ByName("code"))
	HandleError(w, err)
	// w.Write([]byte("OK"))
	w.WriteHeader(200)
}

// All categories.
func aldoCategAllHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params, session *SessionData) {
	data := struct {
		Session     *SessionData
		HeadMessage string
		Categories  []string
	}{session, "", []string{}}
	data.Categories = aldoutil.ReadCategoryList(path.Join(listPath, "categAll.list"))
	err = tmplAldoCategoryAll.ExecuteTemplate(w, "aldoCategoryAll.tmpl", data)
	HandleError(w, err)
}

// Selected categories.
func aldoCategSelHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params, session *SessionData) {
	data := struct {
		Session     *SessionData
		HeadMessage string
		Categories  string
	}{session, "", ""}
	data.Categories = strings.Join(aldoutil.ReadCategoryList(path.Join(listPath, "categSel.list")), "\n")
	err = tmplAldoCategorySel.ExecuteTemplate(w, "aldoCategorySel.tmpl", data)
	HandleError(w, err)
}

// Save selected categories.
func aldoCategSelHandlerPost(w http.ResponseWriter, req *http.Request, _ httprouter.Params, session *SessionData) {
	err := aldoutil.WriteCategoryListFromString(req.FormValue("categories"), path.Join(listPath, "categSel.list"))
	HandleError(w, err)

	// categories := strings.Split(strings.ReplaceAll(req.FormValue("categories"), " ", ""), "\n")
	// fmt.Println("Categories:", categories)

	http.Redirect(w, req, "/ns/aldo/category/sel", http.StatusSeeOther)
	return
}

// Categories in use.
func aldoCategUseHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params, session *SessionData) {
	data := struct {
		Session     *SessionData
		HeadMessage string
		Categories  []string
	}{session, "", []string{}}
	data.Categories = aldoutil.ReadCategoryList(path.Join(listPath, "categUse.list"))
	err = tmplAldoCategoryUse.ExecuteTemplate(w, "aldoCategoryUse.tmpl", data)
	HandleError(w, err)
}
