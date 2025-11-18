package handlers

import (
	"net/http"
	"strconv"

	"obsync/internal/models"
	"obsync/internal/store"

	"github.com/gin-gonic/gin"
)

type RegisterDeviceRequest struct {
	Repo        string `json:"repo" binding:"required"`
	DeviceID    string `json:"device_id" binding:"required"`
	DisplayName string `json:"display_name"`
}

// @Summary Register device
// @Description Register a new device and receive a device token
// @Tags device
// @Accept json
// @Produce json
// @Param body body RegisterDeviceRequest true "device info"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /v1/register-device [post]
func RegisterDeviceHandler(c *gin.Context, db *store.SQLiteStore) {
	var req RegisterDeviceRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d, err := db.CreateDevice(req.DeviceID, req.DisplayName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"device_token": d.Token, "device_id": d.DeviceID})
}

type PostChangesRequest struct {
	DeviceToken string          `json:"device_token" binding:"required"`
	Changes     []models.Change `json:"changes" binding:"required"`
}

// @Summary Push changes
// @Description Push local changes to the server
// @Tags changes
// @Accept json
// @Produce json
// @Param repo path string true "repo"
// @Param body body PostChangesRequest true "changes"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /v1/repos/{repo}/changes [post]
func PostChangesHandler(c *gin.Context, db *store.SQLiteStore) {
	repo := c.Param("repo")
	var req PostChangesRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// validate token
	if _, err := db.GetDeviceByToken(req.DeviceToken); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid device token"})
		return
	}
	// attach repo to changes and set timestamps
	for i := range req.Changes {
		req.Changes[i].Repo = repo
		if req.Changes[i].Timestamp.IsZero() {
			req.Changes[i].Timestamp = models.TimeNow()
		}
	}
	if err := db.SaveChanges(req.Changes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// @Summary List changes
// @Description List changes since an ID
// @Tags changes
// @Accept json
// @Produce json
// @Param repo path string true "repo"
// @Param since query int false "since id"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /v1/repos/{repo}/changes [get]
func ListChangesHandler(c *gin.Context, db *store.SQLiteStore) {
	repo := c.Param("repo")
	sinceStr := c.Query("since")
	var sinceID uint64 = 0
	if sinceStr != "" {
		if v, err := strconv.ParseUint(sinceStr, 10, 64); err == nil {
			sinceID = v
		}
	}
	changes, err := db.ListChangesSince(repo, uint(sinceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"changes": changes})
}
