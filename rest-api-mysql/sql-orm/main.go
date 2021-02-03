package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/mentarie/Iqra_backend/rest-api-mysql/sql-orm/config"
	"github.com/mentarie/Iqra_backend/rest-api-mysql/sql-orm/database"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *sql.DB
var err error

func getConfig() (config.Config, error) {
	viper.AddConfigPath(".")
	viper.SetConfigType("yml")
	viper.SetConfigName("config.yml")

	if err := viper.ReadInConfig(); err != nil {
		return config.Config{}, err
	}

	var cfg config.Config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		return config.Config{}, err
	}

	return cfg, nil
}

func initDB(dbConfig config.Database) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", dbConfig.User, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.DbName, dbConfig.Config)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(
		&database.User{})
	log.Println("db successfully connected")

	return db, nil
}

func handleRequest(con Connection) {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/users", con.GetUsersHandler).Methods("GET")
	router.HandleFunc("/users", con.CreateUserHandler).Methods("POST")
	router.HandleFunc("/users/{id}", con.GetUserHandler).Methods("GET")
	router.HandleFunc("/users/{id}", con.UpdateUserHandler).Methods("PUT")
	router.HandleFunc("/users/{id}", con.DeleteUser).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":8081", router))
}

type Connection struct {
	db *gorm.DB
}
type User struct {
	Id       int    `json:"id" gorm:"primary_key"`
	Id_user  string `json:"id_user"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (con *Connection) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	var user database.User
	if err := json.Unmarshal(body, &user); err != nil {
		WrapAPIError(w, r, fmt.Sprintf("Error while unmarshaling data : ", err.Error()), http.StatusBadRequest)
		return
	}

	if err := database.CreateUser(user, con.db); err != nil {
		WrapAPIError(w, r, fmt.Sprintf("Error while creating user : ", err.Error()), http.StatusBadRequest)
	} else {
		WrapAPISuccess(w, r, "success", http.StatusOK)
	}
}

func (con *Connection) GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	if user, err := database.GetUsers(con.db); err != nil {
		log.Println("Error getting user data ", err.Error())
		return
	} else {
		WrapAPIData(w, r, user, http.StatusOK, "success")
	}
}

func (con *Connection) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	if id, err := strconv.Atoi(params["id"]); err != nil {
		log.Println("Error while converting integer")
		return
	} else {
		if user, err := database.GetUser(id, con.db); err != nil {
			log.Println("Error getting user data ", err.Error())
			return
		} else {
			WrapAPIData(w,r, user, http.StatusOK, "success")
		}
	}

}

func (con *Connection) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	var user database.User
	json.Unmarshal(body, &user)

	if id, err := strconv.Atoi(params["id"]); err != nil {
		log.Println("Error while converting integer")
		return
	} else {
		if err := database.UpdateUser(id, user, con.db); err != nil {
			WrapAPIError(w,r, fmt.Sprintf("Error while updating user : ", err.Error()), http.StatusBadRequest)
		} else {
			WrapAPISuccess(w, r, "success", http.StatusOK)
		}
	}

}

func (con *Connection) DeleteUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	if id, err := strconv.Atoi(params["id"]); err != nil {
		log.Println("Error while converting integer")
		return
	} else {
		if err := database.DeleteUser(id, con.db); err != nil {
			WrapAPIError(w,r, fmt.Sprintf("Error while deleting user : ", err.Error()), http.StatusBadRequest)
		} else {
			WrapAPISuccess(w, r, "success", http.StatusOK)
		}
	}
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Println(err)
		return
	}

	db, err := initDB(cfg.Database)
	if err != nil {
		log.Println(err)
		return
	}

	var con = Connection{db}

	handleRequest(con)
}
