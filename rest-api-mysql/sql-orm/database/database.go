package database

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/twinj/uuid"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var db *gorm.DB //database
var err error

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
	File_iqra		string	`json:"file_iqra"`
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
type TokenDetails struct {
	AccessToken  string
	RefreshToken string
	AccessUuid   string
	RefreshUuid  string
	AtExpires    int64
	RtExpires    int64
}
type AccessDetails struct {
	AccessUuid string
	UserId   uint64
}

func GetUsers(db *gorm.DB) ([]User, error) {
	var users []User
	if err := db.Find(&users).Error; err != nil {
		log.Println("failed to get data :", err.Error())
		return nil, err
	}
	return users, nil
}

func GetUser(id uint64, db *gorm.DB) (User, error) {
	var user User
	if err := db.Where(&User{
		Id: id,
	}).Find(&user).Error; err != nil {
		log.Println("failed to get data :", err.Error())
		return User{}, err
	}
	return user, nil
}

func CreateUser(user User, db *gorm.DB) error {
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	log.Println("Success insert data")
	return nil
}

func UpdateUser(id uint64, user User, db *gorm.DB) (error, error, User) {
	if err := db.Model(&User{}).Where(&User{
		Id: id,
	}).Updates(&user).Error; err != nil {
		return err, nil, user
	}
	return nil, nil, user
}

func DeleteUser(id uint64, db *gorm.DB) error {
	var user User
	if err := db.Model(&User{}).Where(&User{
		Id: id,
	}).Find(&user).Error; err != nil {
		return err
	}
	if err := db.Delete(&user).Error; err != nil {
		return err
	}
	return nil
}

func GetIqra(recordFileName string, db *gorm.DB) (uint64, error) {
	var iqra Iqra
	var status bool

	if err := db.Where(&Iqra{
		File_iqra: recordFileName,
	}).First(&iqra).Error; err != nil {
		log.Println("failed to get data :", err.Error())
		return 0, err
	} else {
		status = true
		log.Println(status)
	}
	return iqra.Id, err
}

func ValidateDataSubmission(Id_user_refer uint64, Id_iqra_refer uint64, db *gorm.DB) (bool, error) {
	//data yang dipake buat validasi
	var submission Submission
	var status bool

	if err := db.Where(&Submission{
		Id_user_refer: Id_user_refer,
		Id_iqra_refer: Id_iqra_refer,
	}).First(&submission).Error; err != nil {
		log.Println("failed to get data :", err.Error())
		return false, err
	} else {
		status = true
	}

	return status, err
}

func SaveSubmission(submission Submission, db *gorm.DB) (Submission, error) {
	if err := db.Create(&submission).Error; err != nil {
		return submission, err
	}
	return submission, nil
}

func UpdateSubmission(submission Submission, db *gorm.DB) (Submission,error) {
	if err := db.Model(&Submission{}).Where(&Submission{
		Id_user_refer: submission.Id_user_refer,
		Id_iqra_refer: submission.Id_iqra_refer,
	}).Updates(&submission).Error; err != nil {
		return submission, err
	}
	return submission,err
}

func GetSubmissions(id uint64, db *gorm.DB) ([]Submission, error) {
	var submissions []Submission

	if err := db.Where(&Submission{
		Id_user_refer: id,
	}).Find(&submissions).Error; err != nil {
		log.Println("failed to get data :", err.Error())
		return submissions, err
	}

	return submissions, err
}

//Validate user detail
func Validate(id uint64, db *gorm.DB) (bool, error, uint64) {
	//data yang dipake buat validasi
	var uservalidation User
	var status bool

	log.Println(id)
	//data dari tabel saat ini bandingin datanya, kalo ada return true, kalo belum ada return false
	if err := db.Where(&User{
		Id: id,
	}).First(&uservalidation).Error; err != nil {
		log.Println("failed to get data :", err.Error())
		return false, err, uservalidation.Id
	} else {
		status = true
		//log.Println(status)
		log.Println(uservalidation.Id)
	}
	//nilai apa yang mau dikembalikan
	return status, nil, uservalidation.Id
}

//Validate login
func ValidateLogin(user User, db *gorm.DB) (bool, error, uint64) {
	//data yang dipake buat validasi
	var uservalidation User
	var status bool

	//data dari tabel saat ini bandingin datanya, kalau sudah ada return false, kalo belum ada return true
	if err := db.Where(&User{
		Username: user.Username,
		Password: user.Password,
	}).First(&uservalidation).Error; err != nil {
		log.Println("failed to get data :", err.Error())
		return false, err , uservalidation.Id
	} else {
		status = true
	}
	//nilai apa yang mau dikembalikan
	return status, nil, uservalidation.Id
}

func CreateToken(userid uint64) (*TokenDetails, error) {
	//2
	td := &TokenDetails{}
	td.AtExpires = time.Now().Add(time.Minute * 15).Unix()
	td.AccessUuid = uuid.NewV4().String()

	td.RtExpires = time.Now().Add(time.Hour * 24 * 7).Unix()
	td.RefreshUuid = uuid.NewV4().String()

	//1
	//Creating Access Token
	os.Setenv("ACCESS_SECRET", "jdnfksdmfksd") //this should be in an env file
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["user_id"] = userid
	atClaims["exp"] = time.Now().Add(time.Minute * 15).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	td.AccessToken, err = at.SignedString([]byte(os.Getenv("ACCESS_SECRET")))
	if err != nil {
		return nil, err
	}
	//3
	//Creating Refresh Token
	os.Setenv("REFRESH_SECRET", "mcmvmkmsdnfsdmfdsjf") //this should be in an env file
	rtClaims := jwt.MapClaims{}
	rtClaims["refresh_uuid"] = td.RefreshUuid
	rtClaims["user_id"] = userid
	rtClaims["exp"] = td.RtExpires
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	td.RefreshToken, err = rt.SignedString([]byte(os.Getenv("REFRESH_SECRET")))
	if err != nil {
		return nil, err
	}
	//nilai apa yang mau dikembalikan ke func login?
	return td, nil
}

func ExtractToken(r *http.Request) string {
	bearToken := r.Header.Get("Authorization")
	//normally Authorization the_token_xxx
	strArr := strings.Split(bearToken, " ")
	if len(strArr) == 2 {
		return strArr[1]
	}
	return ""
}

func VerifyToken(r *http.Request) (*jwt.Token, error) {
	tokenString := ExtractToken(r)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		//Make sure that the token method conform to "SigningMethodHMAC"
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("ACCESS_SECRET")), nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

func TokenValid(r *http.Request) error {
	token, err := VerifyToken(r)
	if err != nil {
		return err
	}
	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		return err
	}
	return nil
}

func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := TokenValid(c.Request)
		if err != nil {
			c.JSON(http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}
		c.Next()
	}
}

func ValidateEmail(user User, db *gorm.DB) (bool, error) {
	//data yang dipake buat validasi
	var uservalidation User
	var status bool

	//data dari tabel saat ini bandingin datanya, kalau sudah ada return false, kalo belum ada return true
	if err := db.Where(&User{
		Email: user.Email,
	}).First(&uservalidation).Error; err != nil {
		log.Println("failed to get data :", err.Error())
		return false, err
	} else {
		status = true
	}
	//nilai apa yang mau dikembalikan
	return status, nil
}