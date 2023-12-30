package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var services = map[string]string{
    "user":    "http://user-service:8001",
    "patient": "http://patient-service:8002",
    // Add other services here
}

func initLogger() {
    // Initialize logrus settings here, like setting log format or level
    log.SetFormatter(&log.TextFormatter{})
    log.SetReportCaller(true)
}

func main() {
	initLogger()

    router := gin.Default()

    router.GET("/health", healthCheck)
    router.POST("/createAccount", createAccount)
    router.POST("/login", loginUser)
    router.PUT("/updateUser/:userID", updateUser) // New route for updating user data

    router.Run(":8080")
}

func healthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

func createAccount(c *gin.Context) {
    serviceProxyWithServiceName(c, "user", "/createAccount")
}

func loginUser(c *gin.Context) {
    serviceProxyWithServiceName(c, "user", "/login")
}

func updateUser(c *gin.Context) {
    userID := c.Param("userID")
    serviceProxyWithServiceName(c, "user", "/updateUser/" + userID)
}

func retry(attempts int, sleep time.Duration, function func() error) error {
    for i := 0; i < attempts; i++ {
        if err := function(); err != nil {
            log.Warnf("Attempt %d failed: %s", i+1, err)
            time.Sleep(sleep)
            continue
        }
        return nil
    }
    return fmt.Errorf("after %d attempts, last error: %s", attempts, function())
}

func serviceProxyWithServiceName(c *gin.Context, serviceName, path string) {
    log.Infof("Proxying request to service: %s, endpoint: %s", serviceName, path)

    serviceURL, ok := services[serviceName]
    if !ok {
        log.Errorf("Service %s not found", serviceName)
        c.JSON(http.StatusNotFound, gin.H{"message": "Service not found"})
        return
    }

    url, err := url.Parse(serviceURL)
    if err != nil {
        log.Errorf("Failed to parse service URL: %s, error: %s", serviceURL, err)
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse service URL"})
        return
    }

    c.Request.URL.Path = path

    proxy := httputil.NewSingleHostReverseProxy(url)

    // Use the retry mechanism
    if err := retry(3, time.Second, func() error {
        rec := httptest.NewRecorder()
        proxy.ServeHTTP(rec, c.Request)
        if rec.Code >= http.StatusInternalServerError {
            // Assuming 5xx codes are retry-able
            return fmt.Errorf("server error: %v", rec.Code)
        }
        c.Writer.WriteHeader(rec.Code)
        c.Writer.Write(rec.Body.Bytes())
        return nil
    }); err != nil {
        log.Errorf("Failed to process request after retries: %s", err)
        c.JSON(http.StatusBadGateway, gin.H{"message": "Failed to process request"})
    }
}