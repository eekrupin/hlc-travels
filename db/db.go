package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/eekrupin/hlc-travels/models"
	"github.com/eekrupin/hlc-travels/modules"
	"gopkg.in/reform.v1"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)
import _ "github.com/go-sql-driver/mysql"

type Config struct {
	Host         string
	Port         int
	Password     string
	User         string
	DBName       string
	MaxOpenConns int
	MaxIdleConns int
}

/*var DB *memdb.MemDB

func Open() (*memdb.MemDB, error) {

	// Create the DB schema
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"user": &memdb.TableSchema{
				Name: "user",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.UintFieldIndex{Field: "Id"},
					},
					"age": &memdb.IndexSchema{
						Name:    "age",
						Unique:  false,
						Indexer: &memdb.IntFieldIndex{Field: "Age"},
					},
					"gender": &memdb.IndexSchema{
						Name:    "gender",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "Gender"},
					},
				},
			},
			"location": &memdb.TableSchema{
				Name: "location",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.UintFieldIndex{Field: "Id"},
					},
					"city": &memdb.IndexSchema{
						Name:    "city",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "City"},
					},
					"place": &memdb.IndexSchema{
						Name:    "place",
						Unique:  false,
						Indexer: &memdb.StringFieldIndex{Field: "Place"},
					},
				},
			},
			"visit": &memdb.TableSchema{
				Name: "visit",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.UintFieldIndex{Field: "Id"},
					},
					"user": &memdb.IndexSchema{
						Name:    "user",
						Unique:  false,
						Indexer: &memdb.IntFieldIndex{Field: "User"},
					},
					"location": &memdb.IndexSchema{
						Name:    "location",
						Unique:  false,
						Indexer: &memdb.IntFieldIndex{Field: "Location"},
					},
				},
			},
		},
	}

	dbConnection, err := memdb.NewMemDB(schema)
	return dbConnection, err

}*/

var DB *sql.DB
var RDB *reform.DB

var queryMap = make(map[string]string)

func Open(c *Config, dataBase string) (dbConnection *sql.DB, err error) {
	dataSourceName := fmt.Sprint(c.User, ":", c.Password, "@tcp(", c.Host, ":", c.Port, ")/", dataBase, "?parseTime=true&multiStatements=true&clientFoundRows=true") //"username:password@tcp(127.0.0.1:3306)/test"
	log.Println("dataSourceName: ", dataSourceName)
	//dataSourceName = "root:12345@tcp(mysql:3306)/travels"
	//fmt.Println("dataSourceName: ", dataSourceName)
	for n := 1; n <= 5; n++ {
		dbConnection, err = sql.Open("mysql", dataSourceName)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		err = dbConnection.Ping()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
	}

	if err != nil {
		return nil, fmt.Errorf("Error while connecting to db %v", err)
	}

	return dbConnection, nil
}

//var CRMDB *gorm.DB

//func Open(c *Config) (db *gorm.DB, err error) {
//	connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s", c.Host, c.User, c.Password, c.Port, c.DBName)
//
//	dbConnection, err := gorm.Open("mssql", connString)
//	if err != nil {
//		return nil, fmt.Errorf("Error while connecting to db %v", err)
//	}
//	dbConnection.DB().SetMaxIdleConns(10)
//	dbConnection.DB().SetMaxOpenConns(100)
//
//	return dbConnection, nil
//}

func InitSchema() {
	_, err := DB.Exec("CREATE SCHEMA IF NOT EXISTS `travels` DEFAULT CHARACTER SET utf8")
	if err != nil {
		panic(err)
	}
}

func InitDB() {

	var err error

	//_, err = DB.Exec("USE `travels`")
	//if err != nil {
	//	panic(err)
	//}

	queryInitDB := GetQuery("initDB")

	queries := strings.Split(queryInitDB, ";")
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		_, err = DB.Exec(query)
		if err != nil {
			panic(err)
		}
	}
	//LoadData()
}

func LoadData() {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err.Error())
	}
	files, err := modules.Unzip(pwd+"//tmp/data//data.zip", pwd+"//tmp//data//output")
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	for _, file := range files {
		b, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatal(err.Error())
		}
		if strings.Contains(file, "users") {
			wg.Add(1)
			go loadUsers(b, &wg)
		} else if strings.Contains(file, "locations") {
			wg.Add(1)
			go loadLocations(b, &wg)
		} else if strings.Contains(file, "visits") {
			wg.Add(1)
			go loadVisits(b, &wg)
		}
	}
	wg.Wait()
	log.Println("Load init data complete")
}

func loadLocations(b []byte, wg *sync.WaitGroup) {
	var dataLocations dataLocations
	err := json.Unmarshal(b, &dataLocations)
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, location := range dataLocations.Locations {
		loadLocation(location, wg)
	}
	wg.Done()
}

func loadLocation(location *models.Location, wg *sync.WaitGroup) {
	err := RDB.Save(location)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func loadUsers(b []byte, wg *sync.WaitGroup) {
	var dataUsers dataUsers
	err := json.Unmarshal(b, &dataUsers)
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, userRaw := range dataUsers.Users {
		loadUser(userRaw, wg)
	}
	wg.Done()
}

func loadUser(userRaw *models.UserRaw, wg *sync.WaitGroup) {
	user := models.UserFromRaw(userRaw)
	err := RDB.Save(user)
	if err != nil {
		log.Fatal(err.Error())
	}

}

func loadVisits(b []byte, wg *sync.WaitGroup) {
	var dataVisits dataVisits
	err := json.Unmarshal(b, &dataVisits)
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, visitRaw := range dataVisits.Visits {
		loadVisit(visitRaw, wg)
	}
	wg.Done()
}

func loadVisit(visitRaw *models.VisitRaw, wg *sync.WaitGroup) {
	visit := models.VisitFromRaw(visitRaw)
	err := RDB.Save(visit)
	if err != nil {
		log.Fatal(err.Error())
	}
}

type dataLocations struct {
	Locations []*models.Location
}

type dataUsers struct {
	Users []*models.UserRaw
}

type dataVisits struct {
	Visits []*models.VisitRaw
}

//func PanicExec(query string, args ...interface{}) sql.Result{
//
//}

func GetQuery(query string) (value string) {
	value = queryMap[query]
	if value == "" {
		fatalText := "On GetQuery '" + query + "' error: "
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatal(fatalText, err.Error())
		}
		b, err := ioutil.ReadFile(pwd + "//queries//" + query + ".sql")
		if err != nil {
			log.Fatal(fatalText, err.Error())
		}
		value = string(b)
	}
	return
}
