package main

import (
	"net/http"
	"os"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)


type User struct {
	gorm.Model
	Username     string
	Password string
	Email        string
	Role         string
}

type Doctor struct {
	gorm.Model
	UserID          uint
	ClinicID        uint
	Specialty       string
	Qualifications  string
	LicenseNumber   string
	ContactNumber   string
	Email           string
	OfficeHours     string
	ProfilePicture  string
	Biography       string
	LanguagesSpoken string
	Ratings         float64
	ConsultationFee float64
	User            User `gorm:"foreignKey:UserID"`
}

type Secretary struct {
	gorm.Model
	UserID          uint
	ClinicID        uint
	AssignedDoctorID uint
	ContactNumber   string
	Email           string
	WorkHours       string
	AssignedTasks   string
	AccessLevel     int
	TrainingCompleted bool
	User            User `gorm:"foreignKey:UserID"`
}



func initDB() *gorm.DB {
	// Replace with your PostgreSQL DSN.
	dsn := os.Getenv("DB_DSN")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Auto migrate tables
	db.AutoMigrate(&User{}, &Doctor{}, &Secretary{})

	return db
}

func main() {
	_ = godotenv.Load()
	router := gin.Default()
	db := initDB()

	router.POST("/signup/doctor", SignUpDoctor(db))
	router.POST("/signup/secretary", SignUpSecretary(db))
	router.POST("/login", LoginUser(db))

	router.Run(":8001")
}

// SignUpDoctor creates a new doctor account
func SignUpDoctor(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var doctor Doctor
		if err := c.ShouldBindJSON(&doctor); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Hash the password
		hashedPassword, err := HashPassword(doctor.User.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		doctor.User.Password = hashedPassword

		// Save the new doctor to the database
		if err := db.Create(&doctor).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create doctor account"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Doctor signed up successfully", "doctor": doctor})
	}
}
// SignUpSecretary creates a new secretary account
func SignUpSecretary(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var secretary Secretary
		if err := c.ShouldBindJSON(&secretary); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Hash the password
		hashedPassword, err := HashPassword(secretary.User.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		secretary.User.Password = hashedPassword

		// Save the new secretary to the database
		if err := db.Create(&secretary).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create secretary account"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Secretary signed up successfully", "secretary": secretary})
	}
}

// HashPassword hashes given password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// CheckPassword compares a hashed password with a possible plaintext equivalent
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}


const TokenDuration = time.Hour  

func generateToken(user User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(TokenDuration).Unix(),
	})

	var jwt_secret = []byte(os.Getenv("JWT_SECRET"))
	tokenString, err := token.SignedString(jwt_secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// LoginUser authenticates the user and provides a JWT token
func LoginUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var loginInfo struct {
			Username string
			Password string
		}

		if err := c.ShouldBindJSON(&loginInfo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user User
		if err := db.Where("username = ?", loginInfo.Username).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect username or password"})
			return
		}

		if !CheckPasswordHash(loginInfo.Password, user.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect username or password"})
			return
		}

		token, err := generateToken(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Logged in successfully", "token": token})
	}
}