package cloud

// DataFolder is the data folder to save the data
var DataFolder = "."

var possibleActivities = []string{"none", "walking", "running", "eating", "playing", "sleeping", "barking"}

var characteristicIDToName = map[int]string{
	0: "temperature",
	1: "humidity",
	2: "ambient_light",
	3: "pressure",
	4: "motion",
	5: "battery",
}

// Define characteristics
var characteristicDefinitions = map[string]characteristicDefinition{
	"00002a6e-0000-1000-8000-00805f9b34fb": {
		Name: "temperature", ValueType: "uint16_t", ID: 0,
		SkipSteps: 100,
	},
	"00002a6f-0000-1000-8000-00805f9b34fb": {
		Name: "humidity", ValueType: "uint8_t", ID: 1,
		SkipSteps: 100,
	},
	"c24229aa-d7e4-4438-a328-c2c548564643": {
		Name: "ambient_light", ValueType: "uint32_t", ID: 2,
		SkipSteps: 2,
	},
	"2f256c42-cdef-4378-8e78-694ea0f53ea8": {
		Name: "pressure", ValueType: "uint16_t", ID: 3,
		SkipSteps: 100,
	},
	"15e438b8-558e-4b1f-992f-23f90a8c129b": {
		Name: "motion", ValueType: "uint16_t", ID: 4,
		SkipSteps: 1,
	},
	"00002a19-0000-1000-8000-00805f9b34fb": {
		Name: "battery", ValueType: "uint8_t", ID: 5,
		SkipSteps: 50,
	},
}

type characteristicDefinition struct {
	Name      string
	ValueType string
	ID        int
	SkipSteps int
}

type sensorData struct {
	Name string `json:"name"`
	Data int    `json:"data"`
}

type serverResponse struct {
	Message string `json:"m"`
	Success bool   `json:"s"`
}
