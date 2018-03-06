package cloud

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/schollz/jsonstore"
	"github.com/schollz/utils"
)

var hubs map[string]*Hub
var hubSync sync.Mutex
var hubs2 map[string]*Hub
var hub2Sync sync.Mutex
var apikeys *jsonstore.JSONStore

var ServerAddress, Port string
var UseSSL bool

// Run initiates the cloud server
func Run() (err error) {
	defer log.Flush()

	// make data folder
	os.MkdirAll(DataFolder, 0755)

	// load current apikeys
	apikeys, err = jsonstore.Open(path.Join(DataFolder, "apikeys.json.gz"))
	if err != nil {
		apikeys = new(jsonstore.JSONStore)
	}
	go func() {
		for {
			jsonstore.Save(apikeys, path.Join(DataFolder, "apikeys.json.gz"))
			time.Sleep(10 * time.Second)
		}
	}()

	// make websocket hubs
	hubs = make(map[string]*Hub)
	hubs2 = make(map[string]*Hub)
	go garbageCollectWebsockets()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")
	r.Use(middleWareHandler(), gin.Recovery())
	r.HEAD("/", handlerOK)
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"Message": "",
		})
	})
	r.POST("/", func(c *gin.Context) {
		message, err := checkLogin(c.PostForm("inputEmail"), c.PostForm("inputPassword"))
		if err != nil {
			c.HTML(http.StatusOK, "login.html", gin.H{
				"Message": err.Error(),
			})
		} else {
			log.Debugf("redirecting to %s", "/realtime?apikey="+message)
			c.Redirect(http.StatusMovedPermanently, "/realtime?apikey="+message)
		}
	})
	r.GET("/realtime", func(c *gin.Context) {
		apikey := c.DefaultQuery("apikey", "")
		username, err := authenticate(apikey)
		if err != nil {
			c.String(http.StatusOK, "not authenticated")
			return
		}
		c.HTML(http.StatusOK, "realtime.tmpl", gin.H{
			"Username":      username,
			"SSL":           UseSSL,
			"ServerAddress": ServerAddress,
			"APIKey":        apikey,
		})
	})
	r.GET("/ws", handleWebsockets)          // handle websockets
	r.GET("/ws2", handleWebsockets2)        // handle websockets
	r.POST("/sensor", handlePostSensorData) // handle sensor data
	r.POST("/activity", handlerPostActivity)
	r.POST("/login", handlerPostLogin)
	r.OPTIONS("/activity", handlerOK)

	log.Infof("Running at http://0.0.0.0:" + Port)
	err = r.Run(":" + Port)
	return
	return
}

func garbageCollectWebsockets() {
	for {
		time.Sleep(1 * time.Second)
		hubSync.Lock()
		namesToDelete := make(map[string]struct{})
		for name := range hubs {
			// log.Debugf("hub %s has %d clients", name, len(hubs[name].clients))
			if len(hubs[name].clients) == 0 {
				namesToDelete[name] = struct{}{}
				hubs[name].deleted = true
			}
		}
		for name := range namesToDelete {
			log.Debugf("deleting hub for %s", name)
			delete(hubs, name)
		}
		hubSync.Unlock()
		hub2Sync.Lock()
		namesToDelete = make(map[string]struct{})
		for name := range hubs2 {
			// log.Debugf("hub %s has %d clients", name, len(hubs[name].clients))
			if len(hubs2[name].clients) == 0 {
				namesToDelete[name] = struct{}{}
				hubs2[name].deleted = true
			}
		}
		for name := range namesToDelete {
			log.Debugf("deleting hub for %s", name)
			delete(hubs2, name)
		}
		hub2Sync.Unlock()
	}
}

func handlerOK(c *gin.Context) { // handler for the uptime robot
	c.String(http.StatusOK, "OK")
}

func checkLogin(username, password string) (message string, err error) {
	db, err := Open("server.db", true)
	if err != nil {
		err = errors.Wrap(err, "could not open db")
		return
	}
	defer db.Close()

	var hashedPassword string
	errGet := db.Get("user:"+username, &hashedPassword)
	if errGet != nil {
		log.Debugf("making new user '%s'", username)
		// add user
		hashedPassword, err = HashPassword(password)
		if err != nil {
			return
		}
		err = db.Set("user:"+username, hashedPassword)
	} else {
		log.Debugf("checking user '%s'", username)
		err = CheckPasswordHash(hashedPassword, password)
		if err != nil {
			err = errors.New("incorrect password")
		}
	}
	if err == nil {
		message = utils.RandStringBytesMaskImprSrc(6)
		apikeys.Set(message, username)
	}
	return
}

func handlerPostLogin(c *gin.Context) {
	message, err := func(c *gin.Context) (message string, err error) {

		var postedJSON LoginJSON
		err = c.ShouldBindJSON(&postedJSON)
		if err != nil {
			err = errors.New("could not bind")
			return
		}
		if len(postedJSON.Username) == 0 {
			err = errors.New("username cannot be empty")
			return
		}
		if len(postedJSON.Password) == 0 {
			err = errors.New("password cannot be empty")
			return
		}
		message, err = checkLogin(postedJSON.Username, postedJSON.Password)
		return
	}(c)

	if err != nil {
		message = err.Error()
	}

	log.Debug(message)
	log.Debug(err)
	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"success": err == nil,
	})

}

func authenticate(apikey string) (username string, err error) {
	// log.Debugf("authenticating %s with %s", c.Request.RequestURI, apikey)
	err = apikeys.Get(apikey, &username)
	if err != nil {
		err = errors.New("incorrect api key")
	}
	return
}

func handlerPostActivity(c *gin.Context) {
	message, err := func(c *gin.Context) (message string, err error) {
		type PostActivity struct {
			APIKey   string `json:"a" binding:required`
			Value    string `json:"v" binding:required`
			Retrieve bool   `json:"r" binding:required`
		}
		var postedJSON PostActivity
		err = c.ShouldBindJSON(&postedJSON)
		if err != nil {
			err = errors.Wrap(err, "incorrect payload")
			return
		}
		username, err := authenticate(postedJSON.APIKey)
		if err != nil {
			return
		}

		db, err := Open(username)
		if err != nil {
			err = errors.Wrap(err, "could not open db")
			return
		}
		defer db.Close()

		if postedJSON.Retrieve {
			message, err = db.GetLatestActivity()
			return
		}

		id := 0
		for i, activity := range PossibleActivities {
			if activity == postedJSON.Value {
				id = i
				break
			}
		}
		err = db.Add("activity", id, 0)
		if err != nil {
			return
		}
		message = fmt.Sprintf("set activity to '%s'", postedJSON.Value)
		return
	}(c)
	if err != nil {
		message = err.Error()
	}
	log.Debug(message)
	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"success": err == nil,
	})
}

func handleWebsockets(c *gin.Context) {
	apikey := c.DefaultQuery("apikey", "")
	username, err := authenticate(apikey)
	if err != nil {
		c.String(http.StatusOK, err.Error())
		return
	}
	log.Debugf("%s joined", username)
	username = convertName(username)
	hubSync.Lock()
	if _, ok := hubs[username]; !ok {
		log.Debugf("created hub %s", username)
		hubs[username] = newHub(username)
		go hubs[username].run()
		time.Sleep(50 * time.Millisecond)
	}
	hubSync.Unlock()
	hubs[username].serveWs(c.Writer, c.Request)
}

func handleWebsockets2(c *gin.Context) {
	apikey := c.DefaultQuery("apikey", "")
	_, err := authenticate(apikey)
	if err != nil {
		c.String(http.StatusForbidden, "not authenticated")
		return
	}
	hub2Sync.Lock()
	if _, ok := hubs2[apikey]; !ok {
		hubs2[apikey] = newHub(apikey)
		go hubs2[apikey].run()
		time.Sleep(50 * time.Millisecond)
	}
	hub2Sync.Unlock()
	hubs2[apikey].serveWs(c.Writer, c.Request)
}

func handlePostSensorData(c *gin.Context) {
	message, err := func(c *gin.Context) (message string, err error) {
		var postedData PostSensorData
		err = c.ShouldBindJSON(&postedData)
		if err != nil {
			return
		}

		message = "ok"
		err = postData(postedData)
		return
	}(c)

	if err != nil {
		log.Warn(err)
		message = err.Error()
	}
	sr := ServerResponse{
		Message: message,
		Success: err == nil,
	}
	c.JSON(http.StatusOK, sr)
}

func postData(postedData PostSensorData) (err error) {
	username, err := authenticate(postedData.APIKey)
	if err != nil {
		return
	}
	// add to database
	postedData.username = username
	postedData.timestampConverted = time.Unix(0, 1000000*postedData.Timestamp).UTC()
	go func(postedData PostSensorData) {
		db, err := Open(postedData.username)
		if err != nil {
			return
		}
		defer db.Close()
		db.Add("sensor", postedData.SensorID, postedData.SensorValue, postedData.timestampConverted)
	}(postedData)

	// broadcast to connected websockets
	go func(postedData PostSensorData) {
		name := convertName(postedData.username)
		hubSync.Lock()
		defer hubSync.Unlock()
		if _, ok := hubs[name]; !ok {
			log.Debugf("not found %s", name)
			return
		}
		s := make(map[string]int)
		s[CharacteristicIDToName[postedData.SensorID]] = postedData.SensorValue
		bPayload, err2 := json.Marshal(s)
		if err2 != nil {
			log.Warn(err2)
			return
		}
		hubs[name].broadcast <- bPayload
	}(postedData)

	return
}

func postData2(postedData PostWebsocket) (err error) {
	username, err := authenticate(postedData.apikey)
	if err != nil {
		return
	}
	if len(postedData.Sensors) == 0 {
		return errors.New("no data")
	}

	// add to database
	postedData.username = username
	postedData.timestampConverted = time.Unix(0, 1000000*postedData.Timestamp).UTC()
	go func(postedData PostWebsocket) {
		db, err := Open(postedData.username)
		if err != nil {
			return
		}
		defer db.Close()
		for sensorID := range postedData.Sensors {
			db.Add("sensor", sensorID, postedData.Sensors[sensorID], postedData.timestampConverted)
		}
	}(postedData)

	// broadcast to connected websockets
	go func(postedData PostWebsocket) {
		name := convertName(postedData.username)
		if _, ok := hubs[name]; !ok {
			return
		}

		s := make(map[string]int)
		for sensorID := range postedData.Sensors {
			s[CharacteristicIDToName[sensorID]] = postedData.Sensors[sensorID]
		}
		bPayload, err2 := json.Marshal(s)
		if err2 != nil {
			log.Warn(err2)
			return
		}
		hubs[name].broadcast <- bPayload
	}(postedData)

	return
}

func middleWareHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		// Add base headers
		addCORS(c)
		// Run next function
		c.Next()
		// Log request
		log.Infof("%v %v %v %s", c.Request.RemoteAddr, c.Request.Method, c.Request.URL, time.Since(t))
	}
}

func addCORS(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Max-Age", "86400")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Max")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
}

func contentType(filename string) string {
	switch {
	case strings.Contains(filename, ".css"):
		return "text/css"
	case strings.Contains(filename, ".jpg"):
		return "image/jpeg"
	case strings.Contains(filename, ".png"):
		return "image/png"
	case strings.Contains(filename, ".js"):
		return "application/javascript"
	case strings.Contains(filename, ".xml"):
		return "application/xml"
	}
	return "text/html"
}
