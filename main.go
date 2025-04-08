package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

const uploadDir = "./uploads"

const (
	username = "admin"
	password = "123456"
)

var token = ""

func main() {
	// 创建上传目录
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		panic("Failed to create upload directory: " + err.Error())
	}

	r := gin.Default()

	// 静态文件服务
	r.Static("/static", "./static")

	// 路由
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	r.POST("/login", loginHandler)

	// 上传和下载路由添加鉴权中间件
	authorized := r.Group("/")
	authorized.Use(authMiddleware)
	{
		authorized.POST("/upload", uploadHandler)
		authorized.GET("/download", downloadHandler)
		authorized.GET("/uploads", uploadsHandler)
		authorized.DELETE("/delete", deleteHandler)
	}

	// 加载HTML模板
	r.LoadHTMLGlob("./static/*.html")

	r.Run(":8080")
}

func uploadHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, "Failed to read file: %s", err.Error())
		return
	}

	destination := filepath.Join(uploadDir, file.Filename)
	if err := c.SaveUploadedFile(file, destination); err != nil {
		c.String(http.StatusInternalServerError, "Failed to save file: %s", err.Error())
		return
	}

	c.String(http.StatusOK, "File uploaded successfully: %s", file.Filename)
}

func downloadHandler(c *gin.Context) {
	filename := c.Query("file")
	if filename == "" {
		c.String(http.StatusBadRequest, "File name is required")
		return
	}

	filePath := filepath.Join(uploadDir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// 设置正确的 Content-Type 和文件名
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Header("Content-Type", "application/octet-stream")
	c.File(filePath)
}

func uploadsHandler(c *gin.Context) {
	files, err := os.ReadDir(uploadDir)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read upload directory: %s", err.Error())
		return
	}

	var fileNames []string
	for _, file := range files {
		if !file.IsDir() {
			fileNames = append(fileNames, file.Name())
		}
	}

	c.JSON(http.StatusOK, fileNames)
}

func loginHandler(c *gin.Context) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if credentials.Username == username && credentials.Password == password {
		token = "secure-token" // 简单的静态 token
		c.JSON(http.StatusOK, gin.H{"message": "Login successful", "token": token})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
	}
}

func authMiddleware(c *gin.Context) {
	clientToken := c.GetHeader("Authorization")
	if clientToken != token {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}
	c.Next()
}

func deleteHandler(c *gin.Context) {
	filename := c.Query("file")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File name is required"})
		return
	}

	filePath := filepath.Join(uploadDir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	if err := os.Remove(filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}
