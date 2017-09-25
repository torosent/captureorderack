package models

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Azure/go-autorest/autorest/utils"
	"github.com/Microsoft/ApplicationInsights-Go/appinsights"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// The order map
var (
	OrderList map[string]*Order
)

var (
	database string
	password string
	status   string
)

var username string
var address []string
var isAzure = true

var hosts string

var insightskey = os.Getenv("INSIGHTSKEY")
var eventURL = os.Getenv("EVENTURL")
var eventPolicyName = os.Getenv("EVENTPOLICYNAME")
var eventPolicyKey = os.Getenv("EVENTPOLICYKEY")

// Order represents the order json
type Order struct {
	ID                string  `required:"false" description:"CosmoDB ID - will be autogenerated"`
	EmailAddress      string  `required:"true" description:"Email address of the customer"`
	PreferredLanguage string  `required:"false" description:"Preferred Language of the customer"`
	Product           string  `required:"false" description:"Product ordered by the customer"`
	Total             float64 `required:"false" description:"Order total"`
	Source            string  `required:"false" description:"Source channel e.g. App Service, Container instance, K8 cluster etc"`
	Status            string  `required:"true" description:"Order Status"`
}

func init() {
	OrderList = make(map[string]*Order)
}

func AddOrder(order Order) (orderId string) {

	return orderId
}

// AddOrderToMongoDB Add the order to MondoDB
func AddOrderToMongoDB(order Order) (orderId string) {

	NewOrderID := bson.NewObjectId()
	order.ID = NewOrderID.Hex()
	order.Status = "Open"
	if order.Source == "" || order.Source == "string" {
		order.Source = os.Getenv("SOURCE")
	}

	database = utils.GetEnvVarOrExit("DATABASE")
	password = utils.GetEnvVarOrExit("PASSWORD")

	// DialInfo holds options for establishing a session with a MongoDB cluster.
	dialInfo := &mgo.DialInfo{
		Addrs:    []string{fmt.Sprintf("%s.documents.azure.com:10255", database)}, // Get HOST + PORT
		Timeout:  60 * time.Second,
		Database: database, // It can be anything
		Username: database, // Username
		Password: password, // PASSWORD
		DialServer: func(addr *mgo.ServerAddr) (net.Conn, error) {
			return tls.Dial("tcp", addr.String(), &tls.Config{})
		},
	}

	// Create a session which maintains a pool of socket connections
	// to our MongoDB.
	session, err := mgo.DialWithInfo(dialInfo)

	if err != nil {
		fmt.Printf("Can't connect to mongo, go error %v\n", err)
		status = "Can't connect to mongo, go error %v\n"
		os.Exit(1)
	}

	defer session.Close()

	// SetSafe changes the session safety mode.
	// If the safe parameter is nil, the session is put in unsafe mode, and writes become fire-and-forget,
	// without error checking. The unsafe mode is faster since operations won't hold on waiting for a confirmation.
	// http://godoc.org/labix.org/v2/mgo#Session.SetMode.
	session.SetSafe(&mgo.Safe{})

	// get collection
	collection := session.DB(database).C("orders")

	// insert Document in collection
	err = collection.Insert(order)

	if err != nil {
		log.Fatal("Problem inserting data: ", err)
		status = "CProblem inserting data, go error %v\n"
		return ""
	}

	//	Let's write only if we have a key
	if insightskey != "" {
		t := time.Now()
		client := appinsights.NewTelemetryClient(insightskey)
		client.TrackEvent("Capture Order " + order.Source + ": " + order.ID)
		client.TrackTrace(t.String())
	}

	// Now let's place this on the eventhub
	if eventURL != "" {
		AddOrderToEventHub(order.ID)
	}
	return order.ID
}

// AddOrderToEventHub adds it to an event hub
func AddOrderToEventHub(orderId string) {

	t := time.Now()
	hostname, err := os.Hostname()
	SaS := createSharedAccessToken(strings.TrimSpace(eventURL), strings.TrimSpace(eventPolicyName), strings.TrimSpace(eventPolicyKey))

	tr := &http.Transport{DisableKeepAlives: false}
	req, _ := http.NewRequest("POST", eventURL, strings.NewReader("{'order':"+"'"+orderId+"', 'source':"+"'"+os.Getenv("SOURCE")+"', 'time':"+"'"+t.String()+"'"+", 'status':"+"'"+"Open"+"'"+", 'hostname':"+"'"+hostname+"'}"))
	req.Header.Set("Authorization", SaS)
	req.Close = false

	res, err := tr.RoundTrip(req)
	if err != nil {
		fmt.Println(res, err)
	}

}

func createSharedAccessToken(uri string, saName string, saKey string) string {

	if len(uri) == 0 || len(saName) == 0 || len(saKey) == 0 {
		return "Missing required parameter"
	}

	encoded := template.URLQueryEscaper(uri)
	now := time.Now().Unix()
	week := 60 * 60 * 24 * 7
	ts := now + int64(week)
	signature := encoded + "\n" + strconv.Itoa(int(ts))
	key := []byte(saKey)
	hmac := hmac.New(sha256.New, key)
	hmac.Write([]byte(signature))
	hmacString := template.URLQueryEscaper(base64.StdEncoding.EncodeToString(hmac.Sum(nil)))

	result := "SharedAccessSignature sr=" + encoded + "&sig=" +
		hmacString + "&se=" + strconv.Itoa(int(ts)) + "&skn=" + saName
	return result
}
