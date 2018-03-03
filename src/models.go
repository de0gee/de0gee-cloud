package cloud

import "time"

// DataFolder is the data folder to save the data
var DataFolder = "."

var PossibleActivities = []string{"none", "walking", "running", "eating", "playing", "sleeping", "barking"}

var CharacteristicIDToName = map[int]string{
	1:  "temperature",
	2:  "humidity",
	3:  "ambient_light",
	4:  "pressure",
	5:  "battery",
	6:  "motion",
	7:  "accelerometer_x",
	8:  "accelerometer_y",
	9:  "accelerometer_z",
	10: "gyroscope_x",
	11: "gyroscope_y",
	12: "gyroscope_z",
	13: "magnetometer_x",
	14: "magnetometer_y",
	15: "magnetometer_z",
}

// Define characteristics
var CharacteristicDefinitions = []CharacteristicDefinition{
	{
		UUID: "00002a6e-0000-1000-8000-00805f9b34fb",
		Name: "temperature", ValueType: "uint16_t", ID: 1,
		SkipSteps: 100,
	},
	{
		UUID: "00002a6f-0000-1000-8000-00805f9b34fb",
		Name: "humidity", ValueType: "uint8_t", ID: 2,
		SkipSteps: 100,
	},
	{
		UUID: "c24229aa-d7e4-4438-a328-c2c548564643",
		Name: "ambient_light", ValueType: "uint32_t", ID: 3,
		SkipSteps: 2,
	},
	{
		UUID: "2f256c42-cdef-4378-8e78-694ea0f53ea8",
		Name: "pressure", ValueType: "uint16_t", ID: 4,
		SkipSteps: 100,
	},
	{
		UUID: "00002a19-0000-1000-8000-00805f9b34fb",
		Name: "battery", ValueType: "uint8_t", ID: 5,
		SkipSteps: 50,
	},
	{
		UUID: "15e438b8-558e-4b1f-992f-23f90a8c129b",
		Name: "motion", ValueType: "uint16_t", ID: 6,
		SkipSteps: 1,
	},
	{
		UUID: "ae840385-b08a-4334-8433-b571573c24ed",
		Name: "accelerometer_x", ValueType: "special", ID: 7,
		SkipSteps: 1,
	},
	{
		UUID: "ae840385-b08a-4334-8433-b571573c24ed",
		Name: "accelerometer_y", ValueType: "", ID: 8,
		SkipSteps: 1,
	},
	{
		UUID: "ae840385-b08a-4334-8433-b571573c24ed",
		Name: "accelerometer_z", ValueType: "", ID: 9,
		SkipSteps: 1,
	},
	{
		UUID: "b61263e0-745b-493a-b45d-41b98c6931ae",
		Name: "gyroscope_x", ValueType: "special", ID: 10,
		SkipSteps: 1,
	},
	{
		UUID: "b61263e0-745b-493a-b45d-41b98c6931ae",
		Name: "gyroscope_y", ValueType: "", ID: 11,
		SkipSteps: 1,
	},
	{
		UUID: "b61263e0-745b-493a-b45d-41b98c6931ae",
		Name: "gyroscope_z", ValueType: "", ID: 12,
		SkipSteps: 1,
	},
	{
		UUID: "6ad90cc5-bceb-4f82-955d-67065647feb1",
		Name: "magnetometer_x", ValueType: "special", ID: 13,
		SkipSteps: 1,
	},
	{
		UUID: "6ad90cc5-bceb-4f82-955d-67065647feb1",
		Name: "magnetometer_y", ValueType: "", ID: 14,
		SkipSteps: 1,
	},
	{
		UUID: "6ad90cc5-bceb-4f82-955d-67065647feb1",
		Name: "magnetometer_z", ValueType: "", ID: 15,
		SkipSteps: 1,
	},
}

type LoginJSON struct {
	Username string `json:"u" binding:required`
	Password string `json:"p" binding:required`
}

type PostSensorData struct {
	APIKey      string `json:"a,omitempty" binding:"required"`
	SensorID    int    `json:"s" binding:"required"`
	SensorValue int    `json:"v" binding:"required"`
	Timestamp   int64  `json:"t" binding:"required"`
	// these are set later
	username           string
	timestampConverted time.Time
}

type CharacteristicDefinition struct {
	UUID      string
	Name      string
	ValueType string
	ID        int
	SkipSteps int
}

type SensorData struct {
	Name string `json:"name"`
	Data int    `json:"data"`
}

type ServerResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}
