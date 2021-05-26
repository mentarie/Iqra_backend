package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/h2non/filetype"
	"github.com/mentarie/Iqra_backend/rest-api-mysql/sql-orm/config"
	"github.com/mentarie/Iqra_backend/rest-api-mysql/sql-orm/database"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	_ "mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", dbConfig.User, dbConfig.Password,
			dbConfig.Host, dbConfig.Port, dbConfig.DbName, dbConfig.Config)
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
	router.HandleFunc("/submissions/{id_user_refer}", con.GetSubmissionsHandler).Methods("GET")
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
	File_iqra 		string	`json:"file_iqra"`
}
type Submission struct {
	gorm.Model
	Id			    uint64  `json:"id" gorm:"primaryKey"`
	Id_user_refer   uint64  `json:"id_user_refer"`
	Id_iqra_refer   uint64  `json:"id_iqra_refer"`
	Accuracy        float64 `json:"accuracy"`
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
	//upload size and path
	const maxUploadSize = 1 * 1024 * 1024 // 1 mb
	const uploadPath = "./spectrograms"

	if r.Method != "POST" {
		WrapAPIError(w,r,"Bad request method", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		fmt.Printf("Could not parse multipart form: %v\n", err)
		WrapAPIError(w,r,"unable to parse form", http.StatusInternalServerError)
		return
	}

	// parse and validate file and post parameters
	file, fileHeader, err := r.FormFile("iqra-file-rekaman")
	if err != nil {
		WrapAPIError(w,r,"invalid file form key", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileSize := fileHeader.Size
	fmt.Printf("File size (bytes): %v\n", fileSize)
	if fileSize > maxUploadSize {
		WrapAPIError(w,r,"max file size is 1 MB",http.StatusBadRequest)
		return
	}

	fileBytes, err := ioutil.ReadAll(file)
	kind, _ := filetype.Match(fileBytes)
	if kind == filetype.Unknown {
		WrapAPIError(w,r,"unknown file type" + kind.MIME.Value, http.StatusBadRequest)
		return
	}

	fmt.Printf("File type: %s. MIME: %s\n", kind.Extension, kind.MIME.Value)
	if err != nil {
		WrapAPIError(w,r,"invalid file reading",http.StatusBadRequest)
		return
	}

	switch kind.MIME.Value {
	case "video/3gpp":
		break
	default:
		WrapAPIError(w,r,"wrong file type : " + kind.MIME.Value, http.StatusBadRequest)
		return
	}

	fileName := fileHeader.Filename
	newPath := filepath.Join(uploadPath, fileName)

	//split file name untuk dapat type
	s := strings.Split(fileHeader.Filename, ".")
	fileType := s[1]
	fmt.Println(fileType)

	//split file name untuk dapat userid
	beforeSplitNama := s[0]
	sn := strings.Split(beforeSplitNama, "-")
	userId, recordFileName := sn[0], sn[1]

	//directory name
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dir)

	// write file
	newFile, err := os.Create(newPath)
	if err != nil {
		WrapAPIError(w,r,"error writing file", http.StatusInternalServerError)
		return
	}
	defer newFile.Close()

	if _, err := newFile.Write(fileBytes); err != nil || newFile.Close() != nil {
		WrapAPIError(w,r,"error writing file", http.StatusInternalServerError)
		return
	}
	//WrapAPISuccess(w,r,"success uploading file",http.StatusOK)

	//convert .3gp to .mp3
	mp3_prg := "ffmpeg"
	mp3_arg1 := "-i"
	mp3_arg2 := dir+"/spectrograms/"+fileHeader.Filename //susuaikan nama file yg dah ada
	mp3_arg3 := "-c:a"
	mp3_arg4 := "libmp3lame"
	mp3_arg5 := dir+"/spectrograms/"+recordFileName+".mp3" //sesuaikan nama file yg akan disimpan

	mp3_cmd := exec.Command(mp3_prg,mp3_arg1,mp3_arg2,mp3_arg3,mp3_arg4,mp3_arg5)
	mp3_stdout, err := mp3_cmd.Output()
	log.Println(mp3_cmd)

	if err != nil {
		fmt.Println("ERROR CONVERT MP3 : " + err.Error())
		return
	}
	fmt.Println(string(mp3_stdout))

	//convert .mp3 to .jpg
	jpg_prg := "ffmpeg"
	jpg_arg1 := "-i"
	jpg_arg2 := dir+"/spectrograms/"+recordFileName+".mp3" //susuaikan nama file yg dah ada
	jpg_arg3 := "-lavfi"
	jpg_arg4 := "showspectrumpic=s=1024x512:legend=disabled"
	jpg_arg5 := dir+"/spectrograms/"+recordFileName+".jpg" //sesuaikan nama file yg akan disimpan

	jpg_cmd := exec.Command(jpg_prg,jpg_arg1,jpg_arg2,jpg_arg3,jpg_arg4,jpg_arg5)
	jpg_stdout, err := jpg_cmd.Output()

	if err != nil {
		fmt.Println("ERROR CONVERT JPG : " + err.Error())
		return
	}
	fmt.Println(string(jpg_stdout))

	//remove file .3gp
	rm_3gp := "rm"
	rm2_3gp := dir+"/spectrograms/"+beforeSplitNama+".3gp"
	fmt.Println(rm2_3gp)

	rm_cmd_3gp := exec.Command(rm_3gp,rm2_3gp)
	rm_stdout_3gp, err := rm_cmd_3gp.Output()

	if err != nil {
		fmt.Println("ERROR REMOVE FILE 3GP : " + err.Error())
		return
	}
	fmt.Println(string(rm_stdout_3gp))

	//remove file .mp3
	rm_mp3 := "rm"
	rm2_mp3 := dir+"/spectrograms/"+recordFileName+".mp3"

	rm_cmd_mp3 := exec.Command(rm_mp3,rm2_mp3)
	rm_stdout_mp3, err := rm_cmd_mp3.Output()

	if err != nil {
		fmt.Println("ERROR REMOVE FILE MP3 : " + err.Error())
		return
	}
	fmt.Println(string(rm_stdout_mp3))

	//save path
	final_file_path := dir+"/spectrograms/"+recordFileName+".jpg"
	fmt.Println(final_file_path)

	//kirim data dan get response dari GCP AUtoML
	if score, resultnama, err := VisionClassificationPredict(final_file_path); err != nil {
		log.Println("Error getting user data ", err.Error())
		return
	} else {
		//convert id user
		s := userId
		id_user_refer, err := strconv.ParseUint(s, 10, 64)
		if err == nil {
			fmt.Printf("%d of type %T", id_user_refer, id_user_refer)
		}

		//ambil data id iqra
		iqraData, err := database.GetIqra(recordFileName, con.db)
		if err!= nil{
			WrapAPIError(w, r, fmt.Sprintf("Error while unmarshaling data : ", err.Error()), http.StatusBadRequest)
			return
		}

		//isi submisi
		var hasil database.Submission
		hasil.Id_user_refer = id_user_refer
		hasil.Id_iqra_refer = iqraData
		hasil.Accuracy = float64(score)
		hasil.Actual_result = resultnama
		hasil.Expected_result = recordFileName

		//validate sebelum save data
		if _, err := database.ValidateDataSubmission(id_user_refer, iqraData, con.db); err != nil {
			//save ke tabel submission
			if userData, err := database.SaveSubmission(hasil, con.db); err != nil {
				WrapAPIError(w, r, fmt.Sprintf("Error while save submission data : ", err.Error()), http.StatusBadRequest)
			} else {
				log.Println(userData)
				WrapAPIData(w, r, userData, http.StatusOK, "data saved")
			}
		} else {
			//update data jika sudah ada
			if userData, err := database.UpdateSubmission(hasil, con.db); err != nil {
				WrapAPIError(w, r, fmt.Sprintf("Error while save submission data : ", err.Error()), http.StatusBadRequest)
			} else {
				log.Println(userData)
				WrapAPIData(w, r, userData, http.StatusOK, "data updated")
			}
		}

		//delete gambar spectrogramnya
		rm_jpg := "rm"
		rm2_jpg := dir+"/spectrograms/"+recordFileName+".jpg"

		rm_cmd_jpg := exec.Command(rm_jpg,rm2_jpg)
		rm_stdout_jpg, err := rm_cmd_jpg.Output()

		if err != nil {
			fmt.Println("ERROR REMOVE FILE JPG : " + err.Error())
			return
		}
		fmt.Println(string(rm_stdout_jpg))
	}
}

func (con *Connection) GetSubmissionsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	//var submission database.Submission
	log.Println(params)

	id := params["id_user_refer"]
	//convert id user
	s := id
	id_user, err := strconv.ParseUint(s, 10, 64)
	if err == nil {
		fmt.Printf("%d of type %T", id_user, id_user)
	}

	if result, err := database.GetSubmissions(id_user, con.db); err != nil {
		log.Println("Error getting user's submission data ", err.Error())
		return
	} else {
		log.Println(result)
		WrapAPIData(w, r, result, http.StatusOK, "success")
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
