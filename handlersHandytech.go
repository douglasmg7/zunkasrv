package main

import (
	// "bytes"

	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	// "math"
	"net/http"
	"strconv"

	// "strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

///////////////////////////////////////////////////////////////////////////////////////////////////
// FILTERS
///////////////////////////////////////////////////////////////////////////////////////////////////
// Get filters.
func handytechFiltersHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params, session *SessionData) {
	data := struct {
		Session     *SessionData
		HeadMessage string
		Filters     HandytechFilters
		Validation  HandytechFiltersValidation
	}{session, "", *handytechFilters, HandytechFiltersValidation{"", ""}}

	// Render filters page.
	err = tmplHandytechFilters.ExecuteTemplate(w, "handytechFilters.gohtml", data)
	HandleError(w, err)
}

// Save filter.
func handytechFiltersHandlerPost(w http.ResponseWriter, req *http.Request, _ httprouter.Params, session *SessionData) {
	filters := HandytechFilters{}
	validation := HandytechFiltersValidation{}

	// defer req.Body.Close()
	// body, err := ioutil.ReadAll(req.Body)
	// HandleError(w, err)
	// log.Printf("receive body: %v", string(body))

	// Validate min price.
	filters.MinPrice = req.PostFormValue("minPrice")
	log.Printf("filters.MinPrice: %s", filters.MinPrice)
	_, err = strconv.ParseUint(filters.MinPrice, 10, 64)
	if err != nil {
		log.Printf("err: %s", err)
		validation.MinPrice = "número inválido"
	}

	// Validate max price.
	filters.MaxPrice = req.PostFormValue("maxPrice")
	_, err = strconv.ParseUint(filters.MaxPrice, 10, 64)
	if err != nil {
		validation.MaxPrice = "número inválido"
	}

	// Some invalid fields.
	if validation.MinPrice != "" && validation.MaxPrice != "" {
		data := struct {
			Session     *SessionData
			HeadMessage string
			Filters     HandytechFilters
			Validation  HandytechFiltersValidation
		}{session, "", filters, validation}

		log.Printf("sending data: %v", data)

		// Render page.
		err = tmplHandytechFilters.ExecuteTemplate(w, "handytechFilters.gohtml", data)
		HandleError(w, err)
	} else {
		// Save filters and go to products page.
		handytechFilters.MinPrice = filters.MinPrice
		handytechFilters.MaxPrice = filters.MaxPrice
		err = handytechFilters.Save()
		HandleError(w, err)
		http.Redirect(w, req, "/ns/handytech/products", http.StatusSeeOther)
	}

	// http.Redirect(w, req, "/ns/handytech/filters", http.StatusSeeOther)
	// w.WriteHeader(200)
	// w.Write([]byte("500 - Something bad happened!"))
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// PRODUCT
///////////////////////////////////////////////////////////////////////////////////////////////////
// Product list.
func handytechProductsHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params, session *SessionData) {
	data := struct {
		Session     *SessionData
		HeadMessage string
		Products    []HandytechProduct
		ValidDate   time.Time
	}{session, "", []HandytechProduct{}, time.Now().Add(-VALID_DATE)}

	// Get products.
	// log.Println(fmt.Sprintf(
	// "SELECT * FROM product WHERE category IN (%s) AND %s ORDER BY description", handytechSelectedCategories.SqlCategories, handytechFilters.SqlFilter))

	// With sql filter.
	// err = dbHandytech.Select(&data.Products, fmt.Sprintf(
	// "SELECT * FROM product WHERE categoria IN (%s) AND fabricante IN (%s) AND %s ORDER BY desc_item",
	// handytechSelectedCategories.SqlCategories, handytechSelectedMakers.SqlMakers, handytechFilters.SqlFilter))

	err = dbHandytech.Select(&data.Products, fmt.Sprintf(
		"SELECT * FROM product WHERE categoria IN (%s) AND fabricante IN (%s) ORDER BY desc_item",
		handytechSelectedCategories.SqlCategories, handytechSelectedMakers.SqlMakers))

	// err = dbHandytech.Select(&data.Products, fmt.Sprintf("SELECT * FROM product ORDER BY desc_item"))

	HandleError(w, err)

	log.Println("Products count:", len(data.Products))

	err = tmplHandytechProducts.ExecuteTemplate(w, "handytechProducts.gohtml", data)
	HandleError(w, err)
}

// Product item.
func handytechProductHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params, session *SessionData) {
	data := struct {
		Session                 *SessionData
		HeadMessage             string
		Product                 *HandytechProduct
		Status                  string
		ShowButtonCreateProduct bool
		SimiliarZunkaProducts   []ZunkaSiteProductRx
		SameEANZunkaProducts    []ZunkaSiteProductRx
	}{session, "", &HandytechProduct{}, "", false, []ZunkaSiteProductRx{}, []ZunkaSiteProductRx{}}

	// Get product.
	err = dbHandytech.Get(data.Product, "SELECT * FROM product WHERE it_codigo=?", ps.ByName("it_codigo"))
	HandleError(w, err)

	// Status.
	data.Status = data.Product.Status()
	// Process images.
	data.Product.ProcessArquivoImagem()

	// Show option to create product on zunkasite.
	if !data.Product.ZunkaProductId.Valid || data.Product.ZunkaProductId.String == "" {
		data.ShowButtonCreateProduct = true

		// Similar titles.
		chanProductsSimilarTitles := make(chan []ZunkaSiteProductRx)
		go getProductsSimilarTitles(chanProductsSimilarTitles, data.Product.DescItem.String)

		// Same EAN.
		// todo - search product data for ean.
		ean := ""
		if len(ean) > 0 {
			chanProductsSameEAN := make(chan []ZunkaSiteProductRx)
			go getProductsSameEAN(chanProductsSameEAN, ean)
			data.SimiliarZunkaProducts, data.SameEANZunkaProducts = <-chanProductsSimilarTitles, <-chanProductsSameEAN
		} else {
			data.SimiliarZunkaProducts = <-chanProductsSimilarTitles
		}
		// Debug.Printf("SimiliarZunkaPorudcts: %v", data.SimiliarZunkaProducts)
	}

	// Render template.
	err = tmplHandytechProduct.ExecuteTemplate(w, "handytechProduct.gohtml", data)
	HandleError(w, err)
}

// Add product to zunka site.
func handytechProductHandlerPost(w http.ResponseWriter, req *http.Request, ps httprouter.Params, session *SessionData) {
	req.ParseForm()
	HandleError(w, err)

	// Get product.
	product := HandytechProduct{}
	// Get product.
	err = dbHandytech.Get(&product, "SELECT * FROM product WHERE id_codigo=?", ps.ByName("it_codigo"))
	HandleError(w, err)

	// Set store product.
	// storeProduct := aldoutil.StoreProduct{}
	storeProduct := ZunkaSiteProductTx{}

	storeProduct.ProductIdTemplate = req.FormValue("similar-product")
	storeProduct.DealerName = "Handytech"
	// storeProduct.DealerProductId = product.Code

	// // Title.
	// // storeProduct.DealerProductTitle = strings.Title(strings.ToLower(product.Description))
	// storeProduct.DealerProductTitle = product.Description
	// // Category.
	// storeProduct.DealerProductCategory = strings.Title(strings.ToLower(product.Category))
	// // Maker.
	// storeProduct.DealerProductMaker = strings.Title(strings.ToLower(product.Maker))
	// // Description.
	// storeProduct.DealerProductDesc = strings.TrimSpace(product.TechnicalDescription)
	// // log.Println("product.TechnicalDescription:", product.TechnicalDescription)
	// // log.Println("storeProduct.DealerProductDesc:", storeProduct.DealerProductDesc)
	// // Image.
	// storeProduct.DealerProductImagesLink = product.UrlImage

	// // Months in days.
	// storeProduct.DealerProductWarrantyDays = product.WarrantyMonth * 30
	// // Length in cm.
	// storeProduct.DealerProductDeep = int(math.Ceil(float64(product.LengthMm) / 10))
	// // Width in cm.
	// storeProduct.DealerProductWidth = int(math.Ceil(float64(product.WidthMm) / 10))
	// // Height in cm.
	// storeProduct.DealerProductHeight = int(math.Ceil(float64(product.HeightMm) / 10))
	// // Weight in grams.
	// storeProduct.DealerProductWeight = product.WeightG
	// // Price.
	// storeProduct.DealerProductPrice = int(product.PriceSale)
	// // Suggestion price.
	// storeProduct.DealerProductFinalPriceSuggestion = int(product.PriceSale + product.PriceSale/3)
	// // Last update.
	// storeProduct.DealerProductLastUpdate = product.ChangedAt
	// // Active.
	// storeProduct.DealerProductActive = product.Availability && product.Active
	// // Origin.
	// storeProduct.DealerProductLocation = product.StockOrigin
	// // Stock.
	// storeProduct.StoreProductQtd = product.StockQty
	// // Ean.
	// storeProduct.Ean = product.Ean

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
		HandleError(w, errors.New(fmt.Sprintf("Error ao solicitar a criação do produto handytech no servidor zunka.\n\nstatus: %v\n\nbody: %v", res.StatusCode, string(resBody))))
		return
	}
	// Mongodb id from created product.
	product.ZunkaProductId.String = string(resBody)
	// Remove suround double quotes.
	product.ZunkaProductId.String = product.ZunkaProductId.String[1 : len(product.ZunkaProductId.String)-1]

	// Update product with _id from mongodb store and set checked_at.
	stmt, err := dbHandytech.Prepare(`UPDATE product SET zunka_product_id = $1, checked_at=$2 WHERE code = $3;`)
	HandleError(w, err)
	defer stmt.Close()
	_, err = stmt.Exec(product.ZunkaProductId, time.Now(), product.ItCodigo)
	HandleError(w, err)

	// Render product page.
	// http.Redirect(w, req, "/ns/handytech/product/"+product.Code, http.StatusSeeOther)

	// Back to product list.
	http.Redirect(w, req, "/ns/handytech/products", http.StatusSeeOther)
}

// Aldo product checked.
func handytechProductCheckedHandlerPost(w http.ResponseWriter, req *http.Request, ps httprouter.Params, session *SessionData) {
	productCode := ps.ByName("code")
	// Update product status_cleaned_at field.
	stmt, err := dbHandytech.Prepare(`UPDATE product SET checked_at=$1 WHERE code = $2;`)
	HandleError(w, err)
	defer stmt.Close()
	_, err = stmt.Exec(time.Now(), productCode)
	HandleError(w, err)

	// Render categories page.
	http.Redirect(w, req, "/ns/handytech/product/"+ps.ByName("code"), http.StatusSeeOther)
}

// Remove mongodb id from Product.
func handytechProductZunkaProductIdHandlerDelete(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	// Update mongodb store id.
	stmt, err := dbHandytech.Prepare(`UPDATE product SET zunka_product_id = $1 WHERE code = $2;`)
	HandleError(w, err)
	defer stmt.Close()
	// fmt.Println("code:", ps.ByName("code"))
	_, err = stmt.Exec("", ps.ByName("code"))
	// _, err = stmt.Exec(ps.ByName("code"), "")
	HandleError(w, err)
	// w.Write([]byte("OK"))
	w.WriteHeader(200)
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// CATEGORY
///////////////////////////////////////////////////////////////////////////////////////////////////
// Get categories.
func handytechCategoriesHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params, session *SessionData) {
	data := struct {
		Session     *SessionData
		HeadMessage string
		Categories  []HandytechCategory
	}{session, "", []HandytechCategory{}}

	// sql := fmt.Sprintf("SELECT categoria as name, count(categoria) as products_qty, false as selected FROM product "+
	// "WHERE  %s GROUP BY categoria ORDER BY name", handytechFilters.SqlFilter)
	sql := fmt.Sprintf("SELECT categoria as name, count(categoria) as products_qty, false as selected FROM product " +
		"GROUP BY categoria ORDER BY name")
	// log.Printf("sql: %v", sql)
	err = dbHandytech.Select(&data.Categories, sql)
	HandleError(w, err)

	m := make(map[string]bool)
	for _, selectedCategory := range handytechSelectedCategories.Categories {
		m[selectedCategory] = true
	}
	// log.Printf("m: %v", m)
	for i := range data.Categories {
		if m[data.Categories[i].Name] {
			data.Categories[i].Selected = true
		}
	}

	// log.Printf("data: %v", len(data.Categories))
	// Render page.
	err = tmplHandytechCategories.ExecuteTemplate(w, "handytechCategories.gohtml", data)
	HandleError(w, err)
}

// Save categories.
func handytechCategoriesHandlerPost(w http.ResponseWriter, req *http.Request, _ httprouter.Params, session *SessionData) {

	req.ParseForm()
	handytechSelectedCategories.Categories = []string{}
	for key := range req.PostForm {
		handytechSelectedCategories.Categories = append(handytechSelectedCategories.Categories, key)
	}
	handytechSelectedCategories.UpdateSqlCategories()
	err := handytechSelectedCategories.Save()
	HandleError(w, err)
	http.Redirect(w, req, "/ns/handytech/products", http.StatusSeeOther)
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// MAKERS
///////////////////////////////////////////////////////////////////////////////////////////////////
// Get categories.
func handytechMakersHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params, session *SessionData) {
	data := struct {
		Session     *SessionData
		HeadMessage string
		Makers      []HandytechMaker
	}{session, "", []HandytechMaker{}}

	// sql := fmt.Sprintf("SELECT fabricante as name, count(fabricante) as products_qty, false as selected FROM product "+
	// "WHERE  %s GROUP BY fabricante ORDER BY name", handytechFilters.SqlFilter)
	sql := fmt.Sprintf("SELECT fabricante as name, count(fabricante) as products_qty, false as selected FROM product " +
		"GROUP BY fabricante ORDER BY name")
	// log.Printf("sql: %v", sql)
	err = dbHandytech.Select(&data.Makers, sql)
	HandleError(w, err)

	m := make(map[string]bool)
	for _, selectedMaker := range handytechSelectedMakers.Makers {
		m[selectedMaker] = true
	}
	// log.Printf("m: %v", m)
	for i := range data.Makers {
		if m[data.Makers[i].Name] {
			data.Makers[i].Selected = true
		}
	}

	// Render page.
	err = tmplHandytechMakers.ExecuteTemplate(w, "handytechMakers.gohtml", data)
	HandleError(w, err)
}

// Save categories.
func handytechMakersHandlerPost(w http.ResponseWriter, req *http.Request, _ httprouter.Params, session *SessionData) {

	req.ParseForm()
	handytechSelectedMakers.Makers = []string{}
	for key := range req.PostForm {
		handytechSelectedMakers.Makers = append(handytechSelectedMakers.Makers, key)
	}
	handytechSelectedMakers.UpdateSqlMakers()
	err := handytechSelectedMakers.Save()
	HandleError(w, err)
	http.Redirect(w, req, "/ns/handytech/products", http.StatusSeeOther)
}
