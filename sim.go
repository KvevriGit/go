package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"log"
	"net/http"
	"os"
)

func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func GetHTML() *goquery.Document {
	res, err := http.Get("https://confluence.hflabs.ru/pages/viewpage.action?pageId=1181220999")
	if err != nil {
		log.Fatal(err)
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	defer res.Body.Close()
	return doc
}

func SheetCreate() (ctx context.Context, spreadsheetId string, srv *sheets.Service) {
	ctx = context.Background()
	b, err := os.ReadFile("client_secret_667021337938-g4m2ia5utpna91cjqragfb4l6cv23e7c.apps.googleusercontent.com.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err = sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	rb := &sheets.Spreadsheet{}

	kek, _ := srv.Spreadsheets.Create(rb).Context(ctx).Do()
	fmt.Printf("%#v\n", kek)
	return ctx, kek.SpreadsheetId, srv
}

func Overwrite(ctx context.Context, spreadsheetId string, srv *sheets.Service, val [][]interface{}) {
	rb := &sheets.BatchUpdateValuesRequest{ValueInputOption: "USER_ENTERED"}
	rb.Data = append(rb.Data, &sheets.ValueRange{Range: "sheet1!A1", Values: val})
	_, err := srv.Spreadsheets.Values.BatchUpdate(spreadsheetId, rb).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Done.")
}

func main() {
	var vals [][]interface{}
	leafsplit := func(leaf *goquery.Selection) (string, string) {
		return leaf.First().Text(), leaf.First().Next().Text()
	}
	foreachleaf := func(index int, leaf *goquery.Selection) {
		a, b := leafsplit(leaf.Find("td"))
		vals = append(vals, []interface{}{a, b})
	}
	documentPointer := GetHTML()
	table := documentPointer.Find("table").Find("tbody").Find("tr") //.Find("td")
	table.Each(foreachleaf)

	ctx, spreadsheetId, srv := SheetCreate()
	Overwrite(ctx, spreadsheetId, srv, vals)
}
