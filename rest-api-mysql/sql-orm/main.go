package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/mentarie/Iqra_backend/rest-api-mysql/sql-orm/config"
	"github.com/mentarie/Iqra_backend/rest-api-mysql/sql-orm/database"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
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
		&database.User{},
		&database.Iqra{},
		&database.Submission{})
	log.Println("db successfully connected")

	return db, nil
}

func handleRequest(con Connection) {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/users", con.GetUsersHandler).Methods("GET")
	router.HandleFunc("/create", con.CreateUserHandler).Methods("POST")
	router.HandleFunc("/login", con.LoginHandler).Methods("POST")
	router.HandleFunc("/token/refresh", con.Refresh).Methods("POST")
	router.HandleFunc("/submission", con.UploadFileHandler).Methods("POST")
	router.HandleFunc("/delete", con.DeleteUser).Methods("DELETE")
	router.HandleFunc("/update", con.UpdateUserHandler).Methods("PUT")

	log.Fatal(http.ListenAndServe(":8081", router))
}

type Connection struct {
	db *gorm.DB
}
type User struct {
	gorm.Model
	Id		       	uint64 `json:"id" gorm:"primaryKey"`
	Username 		string `json:"username"`
	Email    		string `json:"email"`
	Password 		string `json:"password"`
}
type Iqra struct {
	gorm.Model
	Id				uint64	`json:"id" gorm:"primaryKey"`
	Jilid	 		string	`json:"jilid"`
	Halaman			string	`json:"halaman"`
	Section			string	`json:"section"`
	File_iqra 		string	`json:"file_name"`
}
type Submission struct {
	gorm.Model
	Id			    uint64  `json:"id" gorm:"primaryKey"`
	Id_user_refer   uint64  `json:"id_user_refer" gorm:"foreignKey:Id_user"`
	Id_iqra_refer   uint64  `json:"id_iqra_refer" gorm:"foreignKey:File_iqra"`
	Accuracy        float64 `json:"accuracy"`
	Confidence      float64 `json:"confidence"`
	Actual_result   string  `json:"actual_result"`
	Expected_result string  `json:"expected_result"`
}

func (con *Connection) LoginHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	var user database.User
	json.Unmarshal(body, &user)

	//compare the user from the request, with the one we defined:
	if _, err, id := database.ValidateLogin(user, con.db); err != nil {
		WrapAPIError(w, r, fmt.Sprintf("Please provide valid login details", err.Error()), http.StatusBadRequest)
		return
	} else {
		//log.Println(database.ValidateLogin(user, con.db))
		//log.Println(id)
		userData, err := database.GetUser(id, con.db)
		if err!= nil{
			WrapAPIError(w, r, fmt.Sprintf("Error while unmarshaling data : ", err.Error()), http.StatusBadRequest)
			return
		}

		token, err := database.CreateToken(id)
		if err != nil {
			WrapAPIError(w, r, fmt.Sprintf("Error while unmarshaling data : ", err.Error()), http.StatusBadRequest)
			return
		}
		tokens := map[string]string{
			"access_token":  token.AccessToken,
			"refresh_token": token.RefreshToken,
			"id": strconv.FormatUint(userData.Id, 10),
			"username": userData.Username,
			"email": userData.Email,
		}
		WrapAPIData(w, r, tokens, http.StatusOK, "success")
	}
	return
}

func (con *Connection) Refresh(w http.ResponseWriter, r *http.Request) {
	mapToken := map[string]string{}
	refreshToken := mapToken["refresh_token"]

	//verify the token
	os.Setenv("REFRESH_SECRET", "mcmvmkmsdnfsdmfdsjf") //this should be in an env file
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		//Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("REFRESH_SECRET")), nil
	})
	//if there is an error, the token must have expired
	if err != nil {
		WrapAPIError(w,r, fmt.Sprintf("Refresh token expired", err.Error()), http.StatusBadRequest)
		return
	}
	//is token valid?
	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		WrapAPIError(w,r, fmt.Sprintf("Token invalid", err.Error()), http.StatusBadRequest)
		return
	}
	//Since token is valid, get the uuid:
	claims, ok := token.Claims.(jwt.MapClaims) //the token claims should conform to MapClaims
	if ok && token.Valid {
		userId, err := strconv.ParseUint(fmt.Sprintf("%.f", claims["user_id"]), 10, 64)
		if err != nil {
			WrapAPIError(w,r, fmt.Sprintf("Error occurred", err.Error()), http.StatusBadRequest)
			return
		}
		//Create new pairs of refresh and access tokens
		ts, createErr := database.CreateToken(userId)
		if  createErr != nil {
			WrapAPIError(w,r, fmt.Sprintf("status forbidden", err.Error()), http.StatusBadRequest)
			return
		}
		tokens := map[string]string{
			"access_token":  ts.AccessToken,
			"refresh_token": ts.RefreshToken,
		}
		WrapAPIData(w, r, tokens, http.StatusOK, "token updated")
	} else {
		WrapAPIError(w,r, fmt.Sprintf("refresh expired", err.Error()), http.StatusBadRequest)
	}
}

func (con *Connection) CreateUserHandler(w http.ResponseWriter, r *http.Request){
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

func (con *Connection) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	var user database.User
	json.Unmarshal(body, &user)

	//compare the user from the request, with the one we defined:
	if _, err, id := database.Validate(user.Id, con.db); err != nil {
		WrapAPIError(w, r, fmt.Sprintf("Please provide valid login details", err.Error()), http.StatusBadRequest)
		return
	} else {
		//log.Println(id)
		_, err, user := database.UpdateUser(id, user, con.db)
		if err != nil {
			WrapAPIError(w, r, fmt.Sprintf("Error while unmarshaling data : ", err.Error()), http.StatusBadRequest)
			return
		} else {
			userData := map[string]string{
				"id": strconv.FormatUint(id, 10),
				"username": user.Username,
				"email": user.Email,
				"password": user.Password,
			}
			WrapAPIData(w, r, userData, http.StatusOK, "data updated")
		}
	}
	return
}

func (con *Connection) DeleteUser(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	var user database.User
	json.Unmarshal(body, &user)

	if _, err, id := database.Validate(user.Id, con.db); err != nil {
		WrapAPIError(w, r, fmt.Sprintf("Please provide valid login details", err.Error()), http.StatusBadRequest)
		return
	} else {
		//log.Println(database.Validate(user, con.db))
		//log.Println(user.Username)
		err := database.DeleteUser(id, con.db)
		if err != nil {
			WrapAPIError(w, r, fmt.Sprintf("Error while unmarshaling data : ", err.Error()), http.StatusBadRequest)
			return
		} else {
			WrapAPISuccess(w, r, "success", http.StatusOK)
		}
	}
	return
}

func (con *Connection) UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	//ambil data rekaman untuk validasi
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	var user database.User
	json.Unmarshal(body, &user)

	//ambil data audio
	//parse multiparty?


	//validasi dan response
	if _, err, id := database.Validate(user.Id, con.db); err != nil {
		WrapAPIError(w, r, fmt.Sprintf("Please provide valid login details", err.Error()), http.StatusBadRequest)
		return
	} else {
		err, resultData := database.UploadFile(id, user, con.db)
		if err != nil {
			WrapAPIError(w, r, fmt.Sprintf("Error while unmarshaling data : ", err.Error()), http.StatusBadRequest)
			return
		} else {
			WrapAPIData(w, r, resultData, http.StatusOK, "data updated")
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
