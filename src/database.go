package cloud

import (
	"database/sql"
	"encoding/json"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	flock "github.com/theckman/go-flock"
)

// Database is the main structure for holding the information
// pertaining to the name of the database.
type Database struct {
	name     string
	db       *sql.DB
	fileLock *flock.Flock
}

func convertName(name string) string {
	return path.Join(DataFolder, base58.FastBase58Encoding([]byte(strings.ToLower(name)))+".db")
}

// Open will open the database for transactions by first aquiring a filelock.
func Open(name string, forceName ...bool) (d *Database, err error) {
	d = new(Database)

	if len(forceName) > 0 && forceName[0] {
		d.name = path.Join(DataFolder, name)
	} else {
		d.name = convertName(name)
	}

	// check if it is a new database
	newDatabase := false
	if _, err := os.Stat(d.name); os.IsNotExist(err) {
		newDatabase = true
	}

	// // if read-only, throw error if the database does not exist
	// if newDatabase && len(readOnly) > 0 && readOnly[0] {
	// 	err = fmt.Errorf("database '%s' does not exist", name)
	// 	return
	// }

	// obtain a lock on the database
	d.fileLock = flock.NewFlock(d.name + ".lock")
	for {
		locked, err := d.fileLock.TryLock()
		if err == nil && locked {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// open sqlite3 database
	d.db, err = sql.Open("sqlite3", d.name)
	if err != nil {
		return
	}

	// create new database tables if needed
	if newDatabase {
		log.Debug("making new database")
		err = d.MakeTables()
		if err != nil {
			return
		}
		log.Debug("made tables")

		for uuid := range characteristicDefinitions {
			err = d.AddID("sensor", characteristicDefinitions[uuid].Name, characteristicDefinitions[uuid].ID)
			if err != nil {
				return
			}
		}

		for i, activity := range possibleActivities {
			err = d.AddID("activity", activity, i)
			if err != nil {
				return
			}
		}
		d.Add("activity", 0, 0)
	}

	return
}

// Close will close the database connection and remove the filelock.
func (d *Database) Close() (err error) {
	// close filelock
	err = d.fileLock.Unlock()
	if err != nil {
		log.Error(err)
	} else {
		os.Remove(d.name + ".lock")
	}

	// close database
	err2 := d.db.Close()
	if err2 != nil {
		err = err2
		log.Error(err)
	}
	return
}

func (d *Database) MakeTables() (err error) {
	sqlStmt := `create table keystore (key text not null primary key, value text);`
	_, err = d.db.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrap(err, "MakeTables")
		log.Error(err)
		return
	}
	sqlStmt = `create index keystore_idx on keystore(key);`
	_, err = d.db.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrap(err, "MakeTables")
		log.Error(err)
		return
	}
	sqlStmt = `create table sensors (id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, timestamp TIMESTAMP, sensor_id INTEGER, value INTEGER);`
	_, err = d.db.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrap(err, "MakeTables")
		log.Error(err)
		return
	}
	sqlStmt = `CREATE TABLE sensor_ids (id INTEGER PRIMARY KEY, name TEXT);`
	_, err = d.db.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrap(err, "MakeTables")
		log.Error(err)
		return
	}
	sqlStmt = `create table activities (id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, timestamp TIMESTAMP, activity_id INTEGER);`
	_, err = d.db.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrap(err, "MakeTables")
		log.Error(err)
		return
	}
	sqlStmt = `CREATE TABLE activity_ids (id INTEGER PRIMARY KEY, name TEXT);`
	_, err = d.db.Exec(sqlStmt)
	if err != nil {
		err = errors.Wrap(err, "MakeTables")
		log.Error(err)
		return
	}
	return
}

// Set will set a value in the database, when using it like a keystore.
func (d *Database) Set(key string, value interface{}) (err error) {
	var b []byte
	b, err = json.Marshal(value)
	if err != nil {
		return err
	}
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Wrap(err, "Set")
	}
	stmt, err := tx.Prepare("insert or replace into keystore(key,value) values (?, ?)")
	if err != nil {
		return errors.Wrap(err, "Set")
	}
	defer stmt.Close()

	_, err = stmt.Exec(key, string(b))
	if err != nil {
		return errors.Wrap(err, "Set")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "Set")
	}

	// d.logger.Log.Debugf("set '%s' to '%s'", key, string(b))
	return
}

// Get will retrieve the value associated with a key.
func (d *Database) Get(key string, v interface{}) (err error) {
	stmt, err := d.db.Prepare("select value from keystore where key = ?")
	if err != nil {
		return errors.Wrap(err, "problem preparing SQL")
	}
	defer stmt.Close()
	var result string
	err = stmt.QueryRow(key).Scan(&result)
	if err != nil {
		return errors.Wrap(err, "problem getting key")
	}

	err = json.Unmarshal([]byte(result), &v)
	if err != nil {
		return
	}
	// d.logger.Log.Debugf("got %s from '%s'", string(result), key)
	return
}

func (d *Database) AddID(kind string, name string, id int) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Wrap(err, "Set")
	}
	var stmt *sql.Stmt
	if kind == "sensor" {
		stmt, err = tx.Prepare("insert into sensor_ids(id,name) values (?, ?)")
	} else if kind == "activity" {
		stmt, err = tx.Prepare("insert into activity_ids(id,name) values (?, ?)")
	} else {
		err = errors.New("no such kind: " + kind)
	}
	if err != nil {
		return errors.Wrap(err, "Set")
	}
	defer stmt.Close()

	_, err = stmt.Exec(id, name)
	if err != nil {
		return errors.Wrap(err, "Set")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "Set")
	}
	return
}

func (d *Database) Add(kind string, id int, value int, timestamp ...time.Time) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Wrap(err, "AddSensor")
	}
	var stmt *sql.Stmt
	if kind == "sensor" {
		stmt, err = tx.Prepare("insert into sensors(timestamp,sensor_id,value) values (?, ?,?)")
	} else if kind == "activity" {
		stmt, err = tx.Prepare("insert into activities(timestamp,activity_id) values (?, ?)")
	} else {
		err = errors.New("no such kind: " + kind)
	}
	if err != nil {
		return errors.Wrap(err, "AddSensor")
	}
	defer stmt.Close()

	if kind == "sensor" {
		if len(timestamp) > 0 {
			_, err = stmt.Exec(timestamp[0], id, value)
		} else {
			_, err = stmt.Exec(time.Now(), id, value)
		}
	} else if kind == "activity" {
		if len(timestamp) > 0 {
			_, err = stmt.Exec(timestamp[0], id)
		} else {
			_, err = stmt.Exec(time.Now(), id)
		}
	}

	if err != nil {
		return errors.Wrap(err, "AddSensor")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "AddSensor")
	}
	return
}

func (d *Database) GetLatestActivity() (activity string, err error) {
	stmt, err := d.db.Prepare("SELECT activity_ids.name FROM activities INNER JOIN activity_ids ON activities.activity_id=activity_ids.id ORDER BY timestamp DESC LIMIT 1")
	if err != nil {
		return "", errors.Wrap(err, "problem preparing SQL")
	}
	defer stmt.Close()
	err = stmt.QueryRow().Scan(&activity)
	if err != nil {
		return "", errors.Wrap(err, "problem getting key")
	}
	return
}
