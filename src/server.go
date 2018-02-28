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
var apikeys *jsonstore.JSONStore

// Run initiates the cloud server
func Run(port string) (err error) {
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
	go garbageCollectWebsockets()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")
	r.Use(middleWareHandler(), gin.Recovery())
	r.HEAD("/", handlerOK)
	r.GET("/realtime", func(c *gin.Context) {
		username := c.DefaultQuery("username", "zack")
		password := c.DefaultQuery("password", "1234")
		if err = authenticate(c, username, password); err != nil {
			c.String(http.StatusOK, "not authenticated")
			return
		}
		c.HTML(http.StatusOK, "realtime.tmpl", gin.H{
			"title":    "Main website",
			"Name":     username,
			"Password": password,
		})
	})
	r.GET("/ws", handleWebsockets)          // handle websockets
	r.POST("/sensor", handlePostSensorData) // handle sensor data
	r.POST("/activity", handlerPostActivity)
	r.POST("/login", handlerPostLogin)
	r.OPTIONS("/activity", handlerOK)

	log.Infof("Running at http://0.0.0.0:" + port)
	err = r.Run(":" + port)
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
	}
}

func handlerOK(c *gin.Context) { // handler for the uptime robot
	c.String(http.StatusOK, "OK")
}

func handlerPostLogin(c *gin.Context) {
	message, err := func(c *gin.Context) (message string, err error) {
		type LoginJSON struct {
			Username string `json:"u" binding:required`
			Password string `json:"p" binding:required`
		}
		var postedJSON LoginJSON
		err = c.ShouldBindJSON(&postedJSON)
		if err != nil {
			return
		}

		db, err := Open("server.db", true)
		if err != nil {
			err = errors.Wrap(err, "could not open db")
			return
		}
		defer db.Close()

		var hashedPassword string
		errGet := db.Get("user:"+postedJSON.Username, &hashedPassword)
		if errGet != nil {
			log.Debugf("making new user '%s'", postedJSON.Username)
			// add user
			hashedPassword, err = HashPassword(postedJSON.Password)
			if err != nil {
				return
			}
			err = db.Set("user:"+postedJSON.Username, hashedPassword)
		} else {
			log.Debugf("checking user '%s'", postedJSON.Username)
			err = CheckPasswordHash(hashedPassword, postedJSON.Password)
			if err != nil {
				err = errors.New("incorrect password")
			}
		}
		if err == nil {
			message = utils.RandStringBytesMaskImprSrc(6)
			apikeys.Set(message, postedJSON.Username)
		}
		return
	}(c)

	if err != nil {
		message = err.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"message": message,
		"success": err == nil,
	})

}

func authenticate(c *gin.Context, apikey string) (username string, err error) {
	log.Debugf("authenticating %s with %s", c.Request.RequestURI, apikey)
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
		username, err := authenticate(c, postedJSON.APIKey)
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
		for i, activity := range possibleActivities {
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
	username, err := authenticate(c, apikey)
	if err != nil {
		c.String(http.StatusOK, err.Error())
		return
	}
	username = convertName(username)
	hubSync.Lock()
	if _, ok := hubs[username]; !ok {
		hubs[username] = newHub(username)
		go hubs[username].run()
		time.Sleep(50 * time.Millisecond)
	}
	hubSync.Unlock()
	hubs[username].serveWs(c.Writer, c.Request)
}

func handlePostSensorData(c *gin.Context) {
	message, err := func(c *gin.Context) (message string, err error) {
		type postSensorData struct {
			APIKey      string `json:"a" binding:"required"`
			SensorID    int    `json:"s" binding:"required"`
			SensorValue int    `json:"v" binding:"required"`
			Timestamp   int64  `json:"t" binding:"required"`
			// these are set later
			username           string
			timestampConverted time.Time
		}
		var postedData postSensorData
		err = c.ShouldBindJSON(&postedData)
		if err != nil {
			return
		}
		username, err := authenticate(c, postedData.APIKey)
		if err != nil {
			return
		}

		// add to database
		postedData.username = username
		postedData.timestampConverted = time.Unix(0, 1000000*postedData.Timestamp).UTC()
		go func(postedData postSensorData) {
			db, err := Open(postedData.username)
			if err != nil {
				return
			}
			defer db.Close()
			db.Add("sensor", postedData.SensorID, postedData.SensorValue, postedData.timestampConverted)
		}(postedData)

		// broadcast to connected websockets
		go func(postedData postSensorData) {
			name := convertName(postedData.username)
			if _, ok := hubs[name]; !ok {
				return
			}
			bPayload, err2 := json.Marshal(sensorData{
				Name: characteristicIDToName[postedData.SensorID],
				Data: postedData.SensorValue,
			})
			if err2 != nil {
				log.Warn(err2)
				return
			}
			hubs[name].broadcast <- bPayload
		}(postedData)

		message = "ok"
		return
	}(c)

	if err != nil {
		log.Warn(err)
		message = err.Error()
	}
	sr := serverResponse{
		Message: message,
		Success: err == nil,
	}
	c.JSON(http.StatusOK, sr)
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
