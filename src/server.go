package cloud

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

var hubs map[string]*Hub
var hubSync sync.Mutex

// Run initiates the cloud server
func Run(port string) (err error) {
	defer log.Flush()

	// make data folder
	os.MkdirAll(DataFolder, 0755)

	// make websocket hubs
	hubs = make(map[string]*Hub)
	go garbageCollectWebsockets()

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleWareHandler(), gin.Recovery())
	r.HEAD("/", handlerOK)

	r.GET("/ws", handleWebsockets)          // handle websockets
	r.POST("/sensor", handlePostSensorData) // handle sensor data
	r.POST("/activity", handlerPostActivity)
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

func handlerPostActivity(c *gin.Context) {
	message, err := func(c *gin.Context) (message string, err error) {
		type PostActivity struct {
			Username string `json:"u" binding:required`
			Password string `json:"p" binding:required`
			Activity string `json:"a" binding:required`
			Retrieve bool   `json:"r" binding:required`
		}
		var postedJSON PostActivity
		err = c.ShouldBindJSON(&postedJSON)
		if err != nil {
			return
		}

		db, err := Open(postedJSON.Username)
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
			if activity == postedJSON.Activity {
				id = i
				break
			}
		}
		err = db.Add("activity", id, 0)
		if err != nil {
			return
		}
		message = fmt.Sprintf("set activity to '%s'", postedJSON.Activity)
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

func handleWebsockets(c *gin.Context) {
	name := c.DefaultQuery("name", "")
	if name == "" {
		c.String(http.StatusOK, "OK")
		return
	} else {
		name = convertName(name)
	}
	hubSync.Lock()
	if _, ok := hubs[name]; !ok {
		hubs[name] = newHub(name)
		go hubs[name].run()
		time.Sleep(50 * time.Millisecond)
	}
	hubSync.Unlock()
	hubs[name].serveWs(c.Writer, c.Request)
}

func handlePostSensorData(c *gin.Context) {
	message, err := func(c *gin.Context) (message string, err error) {
		type postSensorData struct {
			Username           string `json:"u" binding:"required"`
			Password           string `json:"p" binding:"required"`
			SensorID           int    `json:"s" binding:"required"`
			SensorValue        int    `json:"v" binding:"required"`
			Timestamp          int64  `json:"t" binding:"required"`
			TimestampConverted time.Time
		}
		var postedData postSensorData
		err = c.ShouldBindJSON(&postedData)
		if err != nil {
			return
		}

		// add to database
		postedData.TimestampConverted = time.Unix(0, 1000000*postedData.Timestamp).UTC()
		go func(json postSensorData) {
			db, err := Open(postedData.Username)
			if err != nil {
				return
			}
			defer db.Close()
			db.Add("sensor", postedData.SensorID, postedData.SensorValue, postedData.TimestampConverted)
		}(postedData)

		// broadcast to connected websockets
		go func(postedData postSensorData) {
			name := convertName(postedData.Username)
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
