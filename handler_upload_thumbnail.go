package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)
	file, header_pointer, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failure in getting file", err)
	}
	defer file.Close()

	mediaType := header_pointer.Header.Get("Content-Type")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to convert file", err)
		return
	}
	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error Fetching Video", err)
		return
	}
	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User not authorized", err)
	}

	mime, _, _ := mime.ParseMediaType(mediaType)

	fileType := ""
	switch mime {
	case "image/png":
		fileType = ".png"
	case "image/jpg":
		fileType = ".jpg"
	case "image/jpeg":
		fileType = ".jpeg"
	default:
		respondWithError(w, 500, "invalid media type", nil)
		return
	}

	byteSlice := make([]byte, 32)
	rand.Read(byteSlice)
	randName := base64.RawURLEncoding.EncodeToString(byteSlice)

	thumbName := randName + fileType
	filePath := filepath.Join(cfg.assetsRoot, thumbName)
	log.Printf("filePath: %v", filePath)

	filePointer, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failure in getting file", err)
		return
	}
	defer filePointer.Close()

	_, err = io.Copy(filePointer, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failure in getting file", err)
		return
	}

	thumbnailURLPath := "/assets/" + thumbName

	videoData.ThumbnailURL = &thumbnailURLPath

	err = cfg.db.UpdateVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoData)
}
